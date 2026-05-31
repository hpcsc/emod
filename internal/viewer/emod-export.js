function indent(level) {
  return '  '.repeat(level);
}

function getNodesByType(nodes, type) {
  var result = [];
  for (var i = 0; i < nodes.length; i++) {
    if (nodes[i].type === type) result.push(nodes[i]);
  }
  return result;
}

function getChildNodes(nodes, parentId, type) {
  var result = [];
  for (var i = 0; i < nodes.length; i++) {
    if (nodes[i].type === type && nodes[i].parentId === parentId) {
      result.push(nodes[i]);
    }
  }
  return result;
}

function fieldColumnWidths(fields) {
  var nameWidth = 0;
  var typeWidth = 0;
  for (var i = 0; i < fields.length; i++) {
    var f = fields[i];
    if (f.name.length > nameWidth) nameWidth = f.name.length;
    if (f.type.length > typeWidth) typeWidth = f.type.length;
  }
  return { nameWidth: nameWidth, typeWidth: typeWidth };
}

function writeFields(lines, fields, level) {
  if (!fields || fields.length === 0) return;
  var widths = fieldColumnWidths(fields);
  lines.push(indent(level) + 'fields {');
  for (var i = 0; i < fields.length; i++) {
    var f = fields[i];
    var namePadded = f.name;
    for (var j = f.name.length; j < widths.nameWidth; j++) namePadded += ' ';
    if (f.modifier) {
      var typePadded = f.type;
      for (var j = f.type.length; j < widths.typeWidth; j++) typePadded += ' ';
      lines.push(indent(level + 1) + namePadded + ' ' + typePadded + ' ' + f.modifier);
    } else {
      lines.push(indent(level + 1) + namePadded + ' ' + f.type);
    }
  }
  lines.push(indent(level) + '}');
}

function writeEventBlock(lines, evt, level) {
  lines.push(indent(level) + 'event ' + evt.name + ' {');
  if (evt.source === 'external' && evt.external_name) {
    lines.push(indent(level + 1) + 'source external "' + evt.external_name + '"');
  }
  writeFields(lines, evt.fields, level + 1);
  lines.push(indent(level) + '}');
}

function exportToEmodString(store) {
  var lines = [];
  var nodeById = {};
  for (var i = 0; i < store.nodes.length; i++) {
    nodeById[store.nodes[i].id] = store.nodes[i];
  }

  // Model name
  lines.push('model "' + store.modelName + '"');

  // Actors
  var actors = getNodesByType(store.nodes, 'actor');
  for (var i = 0; i < actors.length; i++) {
    lines.push('');
    lines.push('actor "' + actors[i].label + '"');
  }

  // Contexts
  var contexts = getNodesByType(store.nodes, 'context');
  for (var ci = 0; ci < contexts.length; ci++) {
    var ctx = contexts[ci];
    lines.push('');
    lines.push('context "' + ctx.label + '" {');

    var aggs = getChildNodes(store.nodes, ctx.id, 'aggregate');
    for (var ai = 0; ai < aggs.length; ai++) {
      var agg = aggs[ai];
      if (ai > 0) lines.push('');
      lines.push(indent(1) + 'aggregate "' + agg.label + '" {');

      var slices = getChildNodes(store.nodes, agg.id, 'slice');
      for (var si = 0; si < slices.length; si++) {
        var sl = slices[si];
        if (si > 0) lines.push('');
        lines.push(indent(2) + 'slice "' + sl.label + '" {');

        var inner = 3;
        var needsBlank = false;

        // Trigger
        var triggers = getChildNodes(store.nodes, sl.id, 'trigger');
        for (var ti = 0; ti < triggers.length; ti++) {
          var trg = triggers[ti];
          if (needsBlank || ti > 0) lines.push('');
          lines.push(indent(inner) + 'trigger ' + trg.kind + ' "' + trg.label + '" {');
          if (trg.actor) lines.push(indent(inner + 1) + 'actor ' + trg.actor);
          if (trg.reads) lines.push(indent(inner + 1) + 'reads ' + trg.reads);
          lines.push(indent(inner) + '}');
          needsBlank = true;
        }

        // Commands
        var commands = getChildNodes(store.nodes, sl.id, 'command');
        for (var ci = 0; ci < commands.length; ci++) {
          var cmd = commands[ci];
          if (needsBlank || ci > 0) lines.push('');
          lines.push(indent(inner) + 'command ' + cmd.label + ' {');
          writeFields(lines, cmd.fields, inner + 1);
          lines.push(indent(inner) + '}');
          needsBlank = true;
        }

        // Events
        var events = getChildNodes(store.nodes, sl.id, 'event');
        for (var ei = 0; ei < events.length; ei++) {
          var evt = events[ei];
          if (needsBlank || ei > 0) lines.push('');
          lines.push(indent(inner) + 'event ' + evt.label + ' {');
          if (evt.source === 'external' && evt.external_name) {
            lines.push(indent(inner + 1) + 'source external "' + evt.external_name + '"');
          }
          writeFields(lines, evt.fields, inner + 1);
          lines.push(indent(inner) + '}');
          needsBlank = true;
        }

        // Views
        var views = getChildNodes(store.nodes, sl.id, 'view');
        for (var vi = 0; vi < views.length; vi++) {
          var view = views[vi];
          if (needsBlank || vi > 0) lines.push('');
          lines.push(indent(inner) + 'view ' + view.label + ' {');
          writeFields(lines, view.fields, inner + 1);
          if (view.subscribes && view.subscribes.length > 0) {
            lines.push(indent(inner + 1) + 'subscribes [' + view.subscribes.join(', ') + ']');
          }
          lines.push(indent(inner) + '}');
          needsBlank = true;
        }

        // Automations
        var automations = getChildNodes(store.nodes, sl.id, 'automation');
        for (var ami = 0; ami < automations.length; ami++) {
          var auto = automations[ami];
          if (needsBlank || ami > 0) lines.push('');
          lines.push(indent(inner) + 'automation ' + auto.label + ' {');
          if (auto.trigger_event) lines.push(indent(inner + 1) + 'trigger ' + auto.trigger_event);
          if (auto.command) lines.push(indent(inner + 1) + 'command ' + auto.command);
          if (auto.target_context) lines.push(indent(inner + 1) + 'target context ' + auto.target_context);
          lines.push(indent(inner) + '}');
          needsBlank = true;
        }

        // Translations
        var translations = getChildNodes(store.nodes, sl.id, 'translation');
        for (var tri = 0; tri < translations.length; tri++) {
          var trans = translations[tri];
          if (needsBlank || tri > 0) lines.push('');
          lines.push(indent(inner) + 'translation ' + trans.label + ' {');
          if (trans.external_system) lines.push(indent(inner + 1) + 'external_system "' + trans.external_system + '"');
          if (trans.reads) lines.push(indent(inner + 1) + 'reads ' + trans.reads);
          if (trans.command) lines.push(indent(inner + 1) + 'command ' + trans.command);
          if (trans.event) {
            writeEventBlock(lines, trans.event, inner + 1);
          }
          lines.push(indent(inner) + '}');
          needsBlank = true;
        }

        // Flows
        var flowEdges = [];
        for (var i = 0; i < store.edges.length; i++) {
          var edge = store.edges[i];
          if (edge.type === 'flow') {
            var srcNode = nodeById[edge.source];
            if (srcNode && srcNode.parentId === sl.id) {
              flowEdges.push(edge);
            }
          }
        }
        if (flowEdges.length > 0) {
          if (needsBlank) lines.push('');
          lines.push(indent(inner) + 'flow {');
          for (var fi = 0; fi < flowEdges.length; fi++) {
            var srcNode = nodeById[flowEdges[fi].source];
            var tgtNode = nodeById[flowEdges[fi].target];
            if (srcNode && tgtNode) {
              lines.push(indent(inner + 1) + 'command -> event: ' + srcNode.label + ' -> ' + tgtNode.label);
            }
          }
          lines.push(indent(inner) + '}');
        }

        lines.push(indent(2) + '}');
      }

      lines.push(indent(1) + '}');
    }

    lines.push('}');
  }

  // Trailing newline
  lines.push('');

  return lines.join('\n');
}

export const Export = {
  exportToEmodString: exportToEmodString,
};
