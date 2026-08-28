// The desktop implementation of the platform contract. It lives outside
// static/ because build:web copies that directory wholesale and embed.go embeds
// it wholesale, so a desktop module parked there would ship to the published
// web viewer and into the CLI binary. The desktop build assembles it into its
// own frontend directory as platform.js.
import { Dialogs, Events, System, Window } from '/wails/runtime.js';
import { FileService, ModelService, WindowService } from '../bindings/github.com/hpcsc/emod/internal/desktop/index.js';

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

// Writing the file where the user keeps it is the thing a native shell can do
// and a browser cannot. A path means the caller already knows where, so nothing
// is asked; without one the picker decides, and a cancelled picker answers no
// file rather than throwing, because cancelling is not a failure. The answer
// carries the path the service resolved rather than the one handed in, so a
// caller adopting it as its save target gets an absolute one.
function saveFile(suggestedName, content, path) {
  return (path ? Promise.resolve(path) : promptForSaveLocation(suggestedName))
    .then(function(target) {
      if (!target) {
        return null;
      }
      return FileService.Write(target, content).then(function(raw) {
        const saved = JSON.parse(raw);
        if (saved.error) {
          throw new Error(saved.error);
        }
        return { name: saved.name, path: saved.path };
      });
    });
}

function promptForSaveLocation(suggestedName) {
  return Dialogs.SaveFile({
    Title: 'Save model',
    Filename: suggestedName,
    CanCreateDirectories: true,
    Filters: [{ DisplayName: 'emod models', Pattern: '*.emod;*.json' }],
  });
}

// document.title names a browser tab and nothing at all here: the shell's title
// bar is the native window's, which only the host can rename.
//
// The name and the marker arrive separately and the window has one title to
// hold both, so each is remembered and the title is composed rather than
// overwritten by whichever was set last.
let windowName = '';
let windowModified = false;

function setWindowTitle(title) {
  windowName = title;
  Window.SetTitle(markedTitle());
}

// The viewer states the answer whenever it is freshly known, which is on every
// keystroke; the shell and the window are told only when it moves.
//
// The shell is told on every platform, because closing a window happens outside
// the page and the decision about whether that would discard work cannot wait
// on a round trip into it.
function setWindowModified(modified) {
  if (modified === windowModified) {
    return;
  }
  windowModified = modified;
  WindowService.SetModified(modified);
  if (!marksItsOwnWindow()) {
    Window.SetTitle(markedTitle());
  }
}

// macOS puts a dot in the close button for an edited document, which the shell
// does itself off the answer above — so a star in the title would say the same
// thing twice, in a place macOS reserves for the document's name.
function marksItsOwnWindow() {
  return System.IsMac();
}

function markedTitle() {
  if (!windowModified || marksItsOwnWindow()) {
    return windowName;
  }
  return windowName === '' ? '*' : '* ' + windowName;
}

// A question dialog answers with the label that was pressed, so the labels are
// what the outcomes are read off. Anything else — a dialog dismissed some other
// way, a platform that ignored the labels, a dialog that could not be shown at
// all — is cancel, because it is the only outcome that neither writes the
// user's file nor throws their edits away.
const UNSAVED_EDIT_OUTCOMES = { Save: 'save', Discard: 'discard', Cancel: 'cancel' };

function resolveUnsavedEdits() {
  return Dialogs.Question({
    Title: 'Unsaved changes',
    Message: 'This model has changes that have not been saved.',
    Buttons: [
      { Label: 'Save', IsDefault: true },
      { Label: 'Discard' },
      { Label: 'Cancel', IsCancel: true },
    ],
  }).then(function(pressed) {
    return UNSAVED_EDIT_OUTCOMES[pressed] || 'cancel';
  }).catch(function() {
    return 'cancel';
  });
}

let fileOpenedHandler = null;

function onFileOpened(handler) {
  fileOpenedHandler = handler;
}

let saveRequestedHandler = null;

function onSaveRequested(handler) {
  saveRequestedHandler = handler;
}

// Save and Save As differ only in whether the file already open is written back
// to, so one registration answers both and the request says which. Both names
// are pinned against the shell by internal/desktop's event-name guard.
Events.On('file:save-requested', function() { return requestSave(false); });
Events.On('file:save-as-requested', function() { return requestSave(true); });

// The handler's promise is handed back rather than dropped, so a failure inside
// the viewer's own save surfaces instead of becoming an unhandled rejection.
function requestSave(chooseLocation) {
  if (!saveRequestedHandler) {
    return undefined;
  }
  return saveRequestedHandler({ chooseLocation: chooseLocation });
}

// The shell's File menu carries the only Open control, so it reaches the
// frontend by emitting this. The name is pinned against the Go side by
// internal/desktop's event-name guard, because nothing else connects the two.
Events.On('file:open-requested', promptForFile);

// Requests are numbered because the picker and the read are both asynchronous
// and nothing stops a second Open being asked for while the first is still in
// flight — without this the file whose chain resolves last wins, which is not
// necessarily the one chosen last.
let latestRequest = 0;

function promptForFile() {
  const request = ++latestRequest;

  return Dialogs.OpenFile({
    Title: 'Open model',
    CanChooseFiles: true,
    CanChooseDirectories: false,
    Filters: [{ DisplayName: 'emod models', Pattern: '*.emod;*.json' }],
  }).then(function(path) {
    // A cancelled picker answers with no path, and cancelling has to leave the
    // model on screen exactly as it was — so nothing is delivered at all.
    if (!path) {
      return null;
    }
    return FileService.Read(path).then(JSON.parse);
  }).catch(function(err) {
    // Only the picker and the read are caught here. Letting this cover the
    // delivery too would re-enter the viewer's own handler with its own thrown
    // message, reporting a frontend bug to the user as a file-read failure.
    return { error: err.message || String(err) };
  }).then(function(opened) {
    if (opened && request === latestRequest) {
      deliverFile(opened);
    }
  });
}

function deliverFile(opened) {
  if (fileOpenedHandler) {
    fileOpenedHandler(opened);
  }
}

// Nothing hands this window a model at startup, so it always opens empty. A
// shell launched by opening a file has one to supply here.
function initialState() {
  return Promise.resolve(null);
}

export { parseEmod, exportEmod, droppedFile, saveFile, setWindowTitle, setWindowModified, resolveUnsavedEdits, onFileOpened, onSaveRequested, initialState, ready, isReady };
