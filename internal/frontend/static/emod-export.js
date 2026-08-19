import { ready, exportEmod } from './platform.js';

function exportToEmodString(store) {
  return ready.then(function() {
    return exportEmod({
      model_name: store.modelName,
      nodes: store.nodes,
      edges: store.edges,
    });
  });
}

export const Export = {
  exportToEmodString: exportToEmodString,
};
