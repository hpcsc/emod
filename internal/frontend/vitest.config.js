import { defineConfig } from 'vitest/config'
import { resolve } from 'path'

export default defineConfig({
  resolve: {
    alias: {
      // platform.desktop.js imports the bindings the desktop build generates
      // beside it. They exist only in the assembled frontend, so the source
      // tree cannot load that module without a stand-in.
      '../bindings/github.com/hpcsc/emod/internal/desktop/index.js':
        resolve(import.meta.dirname, 'tests/bindings-stub.js'),
      // The Wails asset server answers /wails/runtime.js from a compiled-in
      // constant, so the path resolves in the running app and nowhere else.
      '/wails/runtime.js':
        resolve(import.meta.dirname, 'tests/wails-runtime-stub.js'),
    },
  },
  test: {
    environment: 'jsdom',
    include: ['**/*.test.js'],
    // One jsdom per worker, and the default is one worker per core: the suite
    // costs a gigabyte on an 8-core machine to save a second.
    pool: 'threads',
    poolOptions: { threads: { maxThreads: 2 } },
  },
})
