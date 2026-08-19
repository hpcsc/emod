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
};
