import { describe, it, expect, vi, beforeAll } from 'vitest';

// What each platform's drop reader answers, held side by side because the two
// deliberately disagree: the browser reads the files the page was handed, and
// the desktop answers none, since its shell consumes the drag and resolves the
// real paths itself. This is also the only place the real FileReader path runs:
// viewer.test.js stubs the seam, and Playwright never drops a file.
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

describe('browser drop read', () => {
  const droppedFiles = () => implementations.browser;

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

// The shell's own drag destination takes the drag before the webview sees it, so
// on the platforms that behave that way there is nothing in the DOM event to
// read. Windows is the exception and still delivers one — which is exactly why
// this answers nothing there too, because the shell delivers those files as
// paths as well and reading both would open the dropped model twice.
describe('desktop drop read', () => {
  const droppedFiles = () => implementations.desktop;

  it('answers nothing, because the shell resolves a drop rather than the page', () => {
    expect(droppedFiles()(transferOf(
      new File(['emod 1\n'], 'hotel.emod'),
      new File(['{}'], 'diagram.json'),
    ))).toEqual([]);
  });
});
