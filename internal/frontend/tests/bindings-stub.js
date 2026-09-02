// Stands in for the bindings `wails3 generate bindings` writes into the desktop
// app's assembled frontend. platform.desktop.js reaches them at ../bindings/,
// which resolves only once the build has copied that module into static/ with
// the generated tree beside it — so from the source tree the module cannot load
// at all without this. vitest.config.js aliases the generated path here.
// Tests drive these: whatever a test assigns to `answers` is what the binding
// returns, and `calls` records what the desktop platform sent across.
export const calls = [];
export const answers = {
  ParseEmod: '{}', ExportJSON: '{}', ExportEmod: '{}', Read: '{}', Write: '{}',
  SetModified: undefined, Record: undefined,
};

export const ModelService = {
  ParseEmod: (arg) => { calls.push(['ParseEmod', arg]); return Promise.resolve(answers.ParseEmod); },
  ExportJSON: (arg) => { calls.push(['ExportJSON', arg]); return Promise.resolve(answers.ExportJSON); },
  ExportEmod: (arg) => { calls.push(['ExportEmod', arg]); return Promise.resolve(answers.ExportEmod); },
};

export const FileService = {
  Read: (arg) => { calls.push(['Read', arg]); return Promise.resolve(answers.Read); },
  Write: (path, content) => { calls.push(['Write', path, content]); return Promise.resolve(answers.Write); },
};

// SetModified is driven through the same bag as every other binding, the way
// the sibling runtime stub does: an Error rejects the call, a promise holds it
// open until the test releases it, and anything else resolves at once.
// `calls` records what was sent, which for a binding is not the same question
// as what arrived: the shell serves each call its own goroutine, so two in
// flight at once land in whichever order they finish. `landed` is that second
// order, and it is the only one a test can hold the frontend to.
export const landed = [];

export const WindowService = {
  SetModified: (modified) => {
    calls.push(['SetModified', modified]);
    const answer = answers.SetModified;
    if (answer instanceof Error) {
      return Promise.reject(answer);
    }
    const held = answer instanceof Promise ? answer : Promise.resolve();

    return held.then(() => { landed.push(modified); });
  },
};

// Record is driven the same way as SetModified, and for the same reason keeps
// its own landing order: `recorded` is the sequence of paths as the shell would
// hold them, which is the only order a test can hold the frontend to.
export const recorded = [];

export const RecentFiles = {
  Record: (path) => {
    calls.push(['Record', path]);
    const answer = answers.Record;
    if (answer instanceof Error) {
      return Promise.reject(answer);
    }
    const held = answer instanceof Promise ? answer : Promise.resolve();

    return held.then(() => { recorded.push(path); });
  },
};
