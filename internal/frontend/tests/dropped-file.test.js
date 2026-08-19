import { describe, it, expect, vi, beforeAll } from 'vitest';

// The drop read is written once per platform implementation — extracting it
// into a shared module would leave an import that resolves from static/ after
// the desktop build assembles the file there but not from where it is kept, so
// the copies are held together here instead. This is also the only place the
// real FileReader path runs: viewer.test.js stubs the seam, and Playwright
// never drops a file.
vi.hoisted(() => {
  globalThis.Go = function Go() {
    this.importObject = {};
    this.run = function () {};
  };
  globalThis.WebAssembly = {
    instantiateStreaming: () => Promise.resolve({ instance: { exports: {} }, module: {} }),
  };
  globalThis.fetch = () => Promise.resolve({ ok: true });
});

const implementations = {};
beforeAll(async () => {
  implementations.browser = (await import('../static/platform.browser.js')).droppedFile;
  implementations.desktop = (await import('../desktop/platform.desktop.js')).droppedFile;
});

function transferOf(...files) {
  return { files };
}

describe.each(['browser', 'desktop'])('%s drop read', (name) => {
  const droppedFile = () => implementations[name];

  it('answers nothing when the drop carried no file', () => {
    expect(droppedFile()(transferOf())).toBeNull();
  });

  it('names the file without reading it', () => {
    const handle = droppedFile()(transferOf(new File(['ignored'], 'hotel.emod')));

    expect(handle.name).toBe('hotel.emod');
    expect(typeof handle.read).toBe('function');
  });

  it('leaves the path empty, because neither host supplies one yet', () => {
    expect(droppedFile()(transferOf(new File(['x'], 'a.emod'))).path).toBe('');
  });

  it('reads the file contents on demand', async () => {
    const handle = droppedFile()(transferOf(new File(['emod 1\nmodel "Hotel"\n'], 'hotel.emod')));

    await expect(handle.read()).resolves.toBe('emod 1\nmodel "Hotel"\n');
  });

  it('takes the first file when several are dropped at once', () => {
    const handle = droppedFile()(transferOf(
      new File(['a'], 'first.emod'),
      new File(['b'], 'second.emod'),
    ));

    expect(handle.name).toBe('first.emod');
  });
});
