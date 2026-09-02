// The desktop implementation of the platform contract. It lives outside
// static/ because build:web copies that directory wholesale and embed.go embeds
// it wholesale, so a desktop module parked there would ship to the published
// web viewer and into the CLI binary. The desktop build assembles it into its
// own frontend directory as platform.js.
import { Application, Dialogs, Events, System, Window } from '/wails/runtime.js';
import { FileService, ModelService, RecentFiles, WindowService } from '../bindings/github.com/hpcsc/emod/internal/desktop/index.js';

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

// The shell's own drag destination takes a file drag before the webview can see
// it, so on the platforms that behave that way a DOM drop carries nothing to
// read. Windows is the exception and still delivers one — and it is answered the
// same way on purpose, because the shell resolves those files to paths as well
// and reading both would open the dropped model twice.
function droppedFiles() {
  return [];
}

// Every gesture that names a model to open is numbered — an Open and a drop
// alike — because each is asynchronous and nothing stops a second arriving while
// the first is in flight. Without this the file whose chain resolves last wins,
// which is not necessarily the one the user asked for last.
//
// The two are numbered together rather than apart: an Open resolves a picker and
// a read before it can deliver, a drop delivers as soon as the shell names the
// paths, so a counter per entry point would let an Open requested first land on
// top of a drop made after it.
let latestGesture = 0;

let filesDroppedHandler = null;

function onFilesDropped(handler) {
  filesDroppedHandler = handler;
}

// What the shell resolved a drop to, which is the thing a webview's own file
// list cannot carry: the real location of each file, and so the location a
// following Save writes back to. The name is pinned against the Go side by
// internal/desktop's event-name guard.
Events.On('file:dropped', function(event) { return deliverDrop(event.data); });

function deliverDrop(paths) {
  // The shell answers nothing when it resolved no file, and a drop it discarded
  // is not one the viewer has to be told about.
  if (!filesDroppedHandler || !paths || paths.length === 0) {
    return undefined;
  }
  // Claimed even though nothing here is read: naming the model is the whole of
  // this gesture, so it supersedes an Open still resolving its own.
  latestGesture++;

  return filesDroppedHandler(paths.map(droppedFileAt));
}

// Naming the file and reading it are separate so the viewer can refuse a drop by
// name without the shell going to disk for a model it will not open. The read
// answers the service's own record of the file rather than anything derived
// here, so the path a save writes back to is the one the service resolved.
function droppedFileAt(path) {
  return {
    name: baseName(path),
    read: function() {
      return FileService.Read(path).then(JSON.parse).catch(function(err) {
        return { error: err.message || String(err) };
      });
    },
  };
}

function baseName(path) {
  const lastSeparator = Math.max(path.lastIndexOf('/'), path.lastIndexOf('\\'));

  return lastSeparator === -1 ? path : path.slice(lastSeparator + 1);
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
  const moved = modified !== windowModified;
  windowModified = modified;
  if (moved) {
    showModifiedInTitle();
  }

  return tellShell(modified);
}

// What the shell was last asked to hold, and whether that ask has landed. Both
// start as nothing-delivered: this record is the page's and the shell's is the
// process's, so a reload resets one and not the other, and a page that assumed
// its first answer already delivered would never send it — leaving the shell
// holding whatever the page before it said.
let shellAsked = false;
let shellHeard = false;

// Answers whether the shell now holds this value. Only for the answers the
// viewer volunteers as the model changes; a decision the shell itself makes
// goes to sendToShell, which never skips.
function tellShell(modified) {
  if (modified === shellAsked && shellHeard) {
    return Promise.resolve(true);
  }

  return sendToShell(modified);
}

// Binding calls are separate HTTP requests the shell serves one goroutine each,
// so two in flight at once land in whichever order they finish rather than the
// order they were sent — and the loser is permanent, because the check above
// then reads the shell as already holding the value it does not. Queuing is
// what makes the last answer the last one written.
let shellQueue = Promise.resolve();

function sendToShell(modified) {
  shellAsked = modified;
  shellQueue = shellQueue.then(function() {
    return WindowService.SetModified(modified).then(function() {
      // Only the newest ask landing means the shell holds what the page last
      // said; an older one arriving after it does not.
      shellHeard = shellAsked === modified;

      return shellHeard;
    }, function() {
      shellHeard = false;

      return false;
    });
  });

  return shellQueue;
}

function showModifiedInTitle() {
  if (!marksItsOwnWindow() && windowName !== '') {
    Window.SetTitle(markedTitle());
  }
}

// macOS puts a dot in the close button for an edited document, which the shell
// does itself off the answer above — so a star in the title would say the same
// thing twice, in a place macOS reserves for the document's name.
function marksItsOwnWindow() {
  return System.IsMac();
}

// A window this has never named carries whatever the shell called it, and
// replacing that with a lone marker would destroy the only name it has.
function markedTitle() {
  if (!windowModified || marksItsOwnWindow() || windowName === '') {
    return windowName;
  }

  return '* ' + windowName;
}

// The shell keeps the list of models opened most recently, because the list
// outlives this page: a reload starts the page over while the shell holds what
// was opened for the life of the process. So nothing here remembers what the
// shell has already been told, and every file is sent. The calls are queued,
// because each is served its own goroutine and two in flight would land in
// whichever order they finished — putting the model opened first at the top.
// A refused recording is handed back to the viewer, which has somewhere to say
// so; a refusal must not hold the next file behind it.
let rememberQueue = Promise.resolve();

function rememberOpenedFile(path) {
  rememberQueue = rememberQueue.catch(function() {}).then(function() {
    return RecentFiles.Record(path);
  });

  return rememberQueue;
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

let leaveRequestedHandler = null;

function onLeaveRequested(handler) {
  leaveRequestedHandler = handler;
}

// The shell refuses a close or a quit while it holds unsaved work and asks here
// instead, because it cannot raise a dialog and wait for it inside a veto that
// has to answer immediately. What to do about the edits is the viewer's to
// decide — it owns the policy an arriving model already goes through, and
// deciding it again here would be a second copy reading this module's own
// shadow of viewer state.
Events.On('window:close-requested', function() { return leaveBy(Window.Close); });
Events.On('app:quit-requested', function() { return leaveBy(Application.Quit); });

function leaveBy(leave) {
  // No viewer registered means nobody can answer, and going anyway would
  // discard the very work the shell refused to leave over.
  if (!leaveRequestedHandler) {
    return undefined;
  }

  return Promise.resolve(leaveRequestedHandler()).then(function(cleared) {
    return cleared ? proceedTo(leave) : undefined;
  });
}

// The shell refused on the state it holds, so it has to have *heard* that the
// state has changed before the close is asked for again — asking first is not
// enough, and a close issued before it lands is refused by the very hook this
// answer authorised, leaving a window that cannot be closed.
//
// Sent rather than told, so the de-duplication above cannot skip it: the shell
// holds its answer for the life of the process while this module's record of it
// is reloaded with the page, so after a reload the two disagree and the record
// says the shell already knows. Skipping the call there vetoes the close, asks,
// is authorised, and vetoes again — forever, with no way to shut the window.
function proceedTo(leave) {
  return sendToShell(false).then(function(heard) {
    // A shell that did not hear this still refuses, so asking would achieve
    // nothing; the window stays open, still saying it holds unsaved work, and
    // the next attempt tries again.
    if (!heard) {
      return undefined;
    }
    windowModified = false;
    showModifiedInTitle();

    return leave();
  });
}

// The shell's File menu carries the only Open control, so it reaches the
// frontend by emitting this. The name is pinned against the Go side by
// internal/desktop's event-name guard, because nothing else connects the two.
Events.On('file:open-requested', promptForFile);

function promptForFile() {
  return openNamedBy(function() {
    return Dialogs.OpenFile({
      Title: 'Open model',
      CanChooseFiles: true,
      CanChooseDirectories: false,
      Filters: [{ DisplayName: 'emod models', Pattern: '*.emod;*.json' }],
    }).then(function(path) {
      // A cancelled picker answers with no path, and cancelling has to leave
      // the model on screen exactly as it was — so nothing is delivered at all.
      if (!path) {
        return null;
      }
      return FileService.Read(path).then(JSON.parse);
    });
  });
}

// An entry chosen from the shell's Open Recent menu, which reaches the frontend
// the way Open does. The read goes through the list's own service rather than
// FileService, because only it knows to forget a file that has gone and to say
// so in its reason. Delivered exactly as a chosen file is, so the viewer cannot
// tell the two apart — the same unsaved-edits question, the same save target.
// The name is pinned against the shell by internal/desktop's event-name guard.
Events.On('file:open-recent-requested', function(event) { return openRecent(event.data); });

function openRecent(path) {
  return openNamedBy(function() {
    return RecentFiles.Open(path).then(JSON.parse);
  });
}

// A gesture that has to name a model and then read it before it can deliver.
// name answers the opened document, null for a gesture that named nothing, or
// fails; what it read is delivered only if no later gesture has been made
// meanwhile.
function openNamedBy(name) {
  const gesture = ++latestGesture;

  return name().catch(function(err) {
    // Only the naming and the read are caught here. Letting this cover the
    // delivery too would re-enter the viewer's own handler with its own thrown
    // message, reporting a frontend bug to the user as a file-read failure.
    return { error: err.message || String(err) };
  }).then(function(opened) {
    if (opened && gesture === latestGesture) {
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

export { parseEmod, exportEmod, droppedFiles, saveFile, setWindowTitle, setWindowModified, rememberOpenedFile, resolveUnsavedEdits, onFileOpened, onFilesDropped, onSaveRequested, onLeaveRequested, initialState, ready, isReady };
