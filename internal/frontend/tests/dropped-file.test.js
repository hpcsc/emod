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
  implementations.browser = (await import('../static/platform.browser.js')).droppedFiles;
  implementations.desktop = (await import('../desktop/platform.desktop.js')).droppedFiles;
});

function transferOf(...files) {
  return { files };
}

describe.each(['browser', 'desktop'])('%s drop read', (name) => {
  const droppedFiles = () => implementations[name];

  it('answers nothing when the drop carried no file', () => {
    expect(droppedFiles()(transferOf())).toEqual([]);
  });

  it('names a file without reading it', () => {
    const [handle] = droppedFiles()(transferOf(new File(['ignored'], 'hotel.emod')));

    expect(handle.name).toBe('hotel.emod');
    expect(typeof handle.read).toBe('function');
  });

  // Which of several files is opened is the viewer's decision and is written
  // once; answering only the first would take that choice away from it.
  it('answers every file the drop carried, in the order they arrived', () => {
    const handles = droppedFiles()(transferOf(
      new File(['a'], 'notes.txt'),
      new File(['b'], 'second.emod'),
      new File(['c'], 'third.json'),
    ));

    expect(handles.map((handle) => handle.name)).toEqual(['notes.txt', 'second.emod', 'third.json']);
  });

  // The same shape a host that opened a file answers, so one routine in the
  // viewer opens a dropped model and a chosen one. path is empty because a file
  // the browser hands over through a drop is content without a location.
  it('reads a file into the shape a host that opened one answers', async () => {
    const [handle] = droppedFiles()(transferOf(
      new File(['emod 1\nmodel "Hotel"\n'], 'hotel.emod'),
    ));

    await expect(handle.read()).resolves.toEqual({
      name: 'hotel.emod',
      path: '',
      content: 'emod 1\nmodel "Hotel"\n',
    });
  });

  // A rejection here would reach the viewer as an unhandled one: it opens what a
  // read answers without catching, because a reason is one of the answers.
  it('answers a reason rather than rejecting when the file cannot be read', async () => {
    const realFileReader = globalThis.FileReader;
    globalThis.FileReader = function BrokenFileReader() {
      this.readAsText = () => { this.onerror(); };
    };

    try {
      const [handle] = droppedFiles()(transferOf(new File(['x'], 'hotel.emod')));

      await expect(handle.read()).resolves.toEqual({ error: 'Failed to read file' });
    } finally {
      globalThis.FileReader = realFileReader;
    }
  });
});
