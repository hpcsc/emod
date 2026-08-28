// Stands in for /wails/runtime.js, which the Wails asset server answers from a
// compiled-in constant. Nothing serves that path in the source tree, so a module
// importing it cannot load at all without this. vitest.config.js aliases the
// specifier here. Tests drive these: `calls` records what the desktop platform
// asked the host for, and whatever a test assigns to `answers` is what the host
// replies.
export const calls = [];
export const answers = {};

export const Window = {
  SetTitle: (title) => { calls.push(['Window.SetTitle', title]); return Promise.resolve(); },
  Close: () => { calls.push(['Window.Close']); return Promise.resolve(); },
};

export const Application = {
  Quit: () => { calls.push(['Application.Quit']); return Promise.resolve(); },
};

// The real runtime reads the OS off a global the shell injects into the page,
// so a test picks which platform the module under test believes it is on by
// assigning answers.IsMac.
export const System = {
  IsMac: () => answers.IsMac === true,
};

// A test fires a host event by calling what the module subscribed with, which
// is the only path the shell itself has into the frontend.
export const listeners = {};

export const Events = {
  On: (name, callback) => { listeners[name] = callback; return () => { delete listeners[name]; }; },
};

export const Dialogs = {
  OpenFile: (options) => {
    calls.push(['Dialogs.OpenFile', options]);
    return answers.OpenFile instanceof Error
      ? Promise.reject(answers.OpenFile)
      : Promise.resolve(answers.OpenFile === undefined ? '' : answers.OpenFile);
  },
  SaveFile: (options) => {
    calls.push(['Dialogs.SaveFile', options]);
    return answers.SaveFile instanceof Error
      ? Promise.reject(answers.SaveFile)
      : Promise.resolve(answers.SaveFile === undefined ? '' : answers.SaveFile);
  },
  // A question dialog answers with the label of the button that was pressed.
  Question: (options) => {
    calls.push(['Dialogs.Question', options]);
    return answers.Question instanceof Error
      ? Promise.reject(answers.Question)
      : Promise.resolve(answers.Question === undefined ? '' : answers.Question);
  },
};
