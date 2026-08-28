// Stands in for the bindings `wails3 generate bindings` writes into the desktop
// app's assembled frontend. platform.desktop.js reaches them at ../bindings/,
// which resolves only once the build has copied that module into static/ with
// the generated tree beside it — so from the source tree the module cannot load
// at all without this. vitest.config.js aliases the generated path here.
// Tests drive these: whatever a test assigns to `answers` is what the binding
// returns, and `calls` records what the desktop platform sent across.
export const calls = [];
export const answers = { ParseEmod: '{}', ExportJSON: '{}', ExportEmod: '{}', Read: '{}', Write: '{}' };

export const ModelService = {
  ParseEmod: (arg) => { calls.push(['ParseEmod', arg]); return Promise.resolve(answers.ParseEmod); },
  ExportJSON: (arg) => { calls.push(['ExportJSON', arg]); return Promise.resolve(answers.ExportJSON); },
  ExportEmod: (arg) => { calls.push(['ExportEmod', arg]); return Promise.resolve(answers.ExportEmod); },
};

export const FileService = {
  Read: (arg) => { calls.push(['Read', arg]); return Promise.resolve(answers.Read); },
  Write: (path, content) => { calls.push(['Write', path, content]); return Promise.resolve(answers.Write); },
};

// A test that has to see what happens between telling the shell something and
// the shell hearing it holds the answer here until it releases the gate.
let setModifiedGate = null;

export function gateSetModified(gate) {
  setModifiedGate = gate;
}

export const WindowService = {
  SetModified: (modified) => {
    calls.push(['SetModified', modified]);
    return setModifiedGate ? setModifiedGate.then(() => undefined) : Promise.resolve();
  },
};
