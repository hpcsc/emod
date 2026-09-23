import '../generated/wasm_exec.js';

async function init() {
  const go = new Go();
  let inst;

  const response = await fetch('../generated/emod.wasm');
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
}

init().then(
  function() {
    globalThis.postMessage({ started: true });
  },
  function(err) {
    globalThis.postMessage({ started: false, error: err.message || String(err) });
  },
);

globalThis.onmessage = function(e) {
  const { id, name, input } = e.data;
  try {
    globalThis.postMessage({ id: id, output: globalThis[name](input) });
  } catch (err) {
    globalThis.postMessage({ id: id, error: err.message || String(err) });
  }
};
