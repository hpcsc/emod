let isReady = false;
let readyResolve;
let readyReject;

const ready = new Promise((resolve, reject) => {
  readyResolve = resolve;
  readyReject = reject;
});

async function init() {
  try {
    const go = new Go();
    let inst;

    const response = await fetch('generated/emod.wasm');
    if (!response.ok) {
      throw new Error(
        'Failed to fetch WASM: ' + response.status + ' ' + response.statusText
      );
    }

    if (WebAssembly.instantiateStreaming) {
      const result = await WebAssembly.instantiateStreaming(
        response,
        go.importObject,
      );
      inst = result.instance;
    } else {
      const bytes = await response.arrayBuffer();
      const mod = await WebAssembly.compile(bytes);
      inst = (await WebAssembly.instantiate(mod, go.importObject)).instance;
    }

    go.run(inst);
    isReady = true;
    readyResolve();
  } catch (err) {
    readyReject(new Error('WASM initialization failed: ' + (err.message || err)));
  }
}

init();

function parseEmod(source) {
  if (!isReady) {
    return Promise.reject(new Error('WASM not ready yet'));
  }
  try {
    const jsonStr = globalThis.parseEmod(JSON.stringify({ source: source }));
    return Promise.resolve(JSON.parse(jsonStr));
  } catch (err) {
    return Promise.reject(err);
  }
}

function exportEmod(diagram) {
  if (!isReady) {
    return Promise.reject(new Error('WASM not ready yet'));
  }
  try {
    const result = JSON.parse(globalThis.exportEmod(JSON.stringify(diagram)));
    if (result.error) {
      return Promise.reject(new Error(result.error));
    }
    return Promise.resolve(result.emod);
  } catch (err) {
    return Promise.reject(err);
  }
}

// Naming the file and reading it are separate so a caller can refuse one by
// name without paying for its contents. path is always empty here: a file the
// browser hands over through a drop is content without a location, which is
// why the browser build can only ever offer a model back as a download.
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

// A browser can only offer the file back as a download, so suggestedName is
// all the say the caller gets over where it lands and the returned promise
// says the download started, never that it was kept. A host able to write to a
// path is handed one; there is none here, which is why the answer names no file
// for the caller to adopt as its own.
function saveFile(suggestedName, content) {
  const url = URL.createObjectURL(new Blob([content], { type: "text/plain" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = suggestedName;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
  return Promise.resolve();
}

function setWindowTitle(title) {
  document.title = title;
}

// A page has no window of its own to mark, and marking the tab would announce
// work the browser viewer offers no way to keep: it cannot write back to the
// file a drop came from, only offer the model as a download.
function setWindowModified() {}

// Nothing in a browser opens a file on the user's behalf. What a page can reach
// arrives through the drop handler, which yields contents and never a location,
// so this registers a handler that stays uncalled rather than being unfinished.
function onFileOpened() {}

// Nor does anything in a browser ask the page to save: there is no menu bar and
// no shell behind one. As with onFileOpened the handler is accepted and never
// called, so the shared viewer registers the same way whatever it runs on.
function onSaveRequested() {}

// The CLI's server substitutes this global into the page before the frontend
// loads, so it is already there by the time anything asks; the promise is for
// the hosts that have to go and fetch their state instead.
function initialState() {
  if (typeof INITIAL_DATA === 'undefined' || INITIAL_DATA === null) {
    return Promise.resolve(null);
  }
  return Promise.resolve(INITIAL_DATA);
}

export { parseEmod, exportEmod, droppedFile, saveFile, setWindowTitle, setWindowModified, onFileOpened, onSaveRequested, initialState, ready, isReady };
