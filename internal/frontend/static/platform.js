// The platform seam. Every shared module reaches the Go core and the host's
// files through this specifier, and which implementation it resolves to is
// decided when a distribution is assembled: the CLI and web bundles ship this
// file as it stands, while the desktop build overwrites it in its own assembled
// copy. Importing platform.browser.js directly from a shared module would fork
// the frontend into a browser-only one and a desktop one.
export { parseEmod, exportEmod, droppedFile, saveFile, setWindowTitle, onFileOpened, initialState, ready, isReady } from './platform.browser.js';
