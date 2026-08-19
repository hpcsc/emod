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
// says the download started, never that it was kept.
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

export { parseEmod, exportEmod, droppedFile, saveFile, ready, isReady };
