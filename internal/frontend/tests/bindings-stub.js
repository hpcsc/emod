// Stands in for the bindings `wails3 generate bindings` writes into the desktop
// app's assembled frontend. platform.desktop.js reaches them at ../bindings/,
// which resolves only once the build has copied that module into static/ with
// the generated tree beside it — so from the source tree the module cannot load
// at all without this. vitest.config.js aliases the generated path here.
export const ModelService = {
  ParseEmod: () => Promise.resolve('{"diagnostics":[],"diagram":{"nodes":[],"edges":[]}}'),
  ExportJSON: () => Promise.resolve('{"diagnostics":[],"model":{}}'),
  ExportEmod: () => Promise.resolve('{"emod":""}'),
};
