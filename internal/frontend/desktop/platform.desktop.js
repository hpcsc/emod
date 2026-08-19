// The desktop implementation of the platform contract. It lives outside
// static/ because build:web copies that directory wholesale and embed.go embeds
// it wholesale, so a desktop module parked there would ship to the published
// web viewer and into the CLI binary. The desktop build assembles it into its
// own frontend directory as platform.js.
import { Window } from '/wails/runtime.js';
import { ModelService } from '../bindings/github.com/hpcsc/emod/internal/desktop/index.js';

// The Go core is linked into the binary, so there is nothing to fetch and
// nothing to wait for. The browser implementation's ready/isReady exist to gate
// on a WebAssembly module loading over HTTP; here they are already settled, and
// the shared modules keep gating on them without knowing the difference.
const ready = Promise.resolve();
const isReady = true;

function parseEmod(source) {
  return ModelService.ParseEmod(JSON.stringify({ source: source })).then(function(raw) {
    return JSON.parse(raw);
  });
}

function exportEmod(diagram) {
  return ModelService.ExportEmod(JSON.stringify(diagram)).then(function(raw) {
    const result = JSON.parse(raw);
    if (result.error) {
      throw new Error(result.error);
    }
    return result.emod;
  });
}

// A webview reads a dropped file the same way a browser does, so this is the
// browser implementation and path is left empty. A native drop can name the
// file's real location, which is the thing worth adding here — the empty path
// is a gap, not a property of the platform.
function droppedFile(dataTransfer) {
  const file = dataTransfer.files[0];
  if (!file) {
    return null;
  }
  return {
    name: file.name,
    path: '',
    read: function() {
      return new Promise(function(resolve, reject) {
        const reader = new FileReader();
        reader.onload = function(e) { resolve(e.target.result); };
        reader.onerror = function() { reject(new Error('Failed to read file')); };
        reader.readAsText(file);
      });
    },
  };
}

// Refusing is deliberate: writing the file to the host is the thing a native
// shell can do and a browser cannot, and half-answering it with a download
// would make the desktop build quietly worse than the one it replaces. It
// rejects rather than returning silently so the export handler's catch puts
// the reason on screen — a user who presses Export has to be told.
function saveFile() {
  return Promise.reject(new Error('Saving is not available in this build yet'));
}

// document.title names a browser tab and nothing at all here: the shell's title
// bar is the native window's, which only the host can rename.
function setWindowTitle(title) {
  Window.SetTitle(title);
}

// Nothing hands this window a model at startup, so it always opens empty. A
// shell launched by opening a file has one to supply here.
function initialState() {
  return Promise.resolve(null);
}

export { parseEmod, exportEmod, droppedFile, saveFile, setWindowTitle, initialState, ready, isReady };
