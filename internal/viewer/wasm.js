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

    const response = await fetch('static/emod.wasm');
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
    const jsonStr = globalThis.parseEmod(source);
    return Promise.resolve(JSON.parse(jsonStr));
  } catch (err) {
    return Promise.reject(err);
  }
}

export { parseEmod, ready, isReady };
