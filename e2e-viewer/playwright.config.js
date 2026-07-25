import { defineConfig, devices } from '@playwright/test';

const PORT = Number(process.env.PORT || 4173);
const baseURL = 'http://127.0.0.1:' + PORT;

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: process.env.CI ? 'list' : [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL,
    acceptDownloads: true,
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
      testIgnore: /touch\.spec\.js/,
    },
    {
      // Touch emulation is opt-in, and enabling it globally would change how
      // the pointer specs are dispatched, so the gestures get their own project.
      name: 'chromium-touch',
      use: { ...devices['Desktop Chrome'], hasTouch: true },
      testMatch: /touch\.spec\.js/,
    },
  ],
  webServer: {
    command: 'node server.mjs',
    url: baseURL,
    reuseExistingServer: !process.env.CI,
    stdout: 'ignore',
    stderr: 'pipe',
  },
});
