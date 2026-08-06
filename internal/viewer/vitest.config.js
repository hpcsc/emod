import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'jsdom',
    include: ['**/*.test.js'],
    // One jsdom per worker, and the default is one worker per core: the suite
    // costs a gigabyte on an 8-core machine to save a second.
    pool: 'threads',
    poolOptions: { threads: { maxThreads: 2 } },
  },
})
