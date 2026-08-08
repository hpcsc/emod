export const L = {
  marginX: 40, marginY: 60,
  sliceWidth: 280, boxWidth: 240, boxHeight: 55,
  sliceGap: 40, contextGap: 70,
  laneHeight: 190, laneGap: 30,
  swimlaneHdr: 44, swimlanePad: 10,
  swimlaneGap: 20, aggLabelH: 22,
  sideGap: 12,
  slicePad: 20, sliceHdrH: 28, sliceTopPad: 36,
  arrowHitWidth: 16,
  portGap: 4, portLen: 12, portHalf: 6, portHitR: 11,
};

// The sides a connection can be drawn from, each with the unit vector pointing
// out of the block. Rendering shapes the arrowhead along it and the interaction
// code reads the same list back, so a port that renders can always be grabbed.
export const PORT_DIRECTIONS = [
  { name: "top",    dx: 0,  dy: -1 },
  { name: "right",  dx: 1,  dy: 0  },
  { name: "bottom", dx: 0,  dy: 1  },
  { name: "left",   dx: -1, dy: 0  },
];

// The kinds of prose a construct can carry, in the order a construct carrying
// more than one shows them. Rendering writes each kind as the `data-marker` of
// the mark that stands for it, and the tooltip code resolves a hovered mark
// back through the same list, so a mark that renders can always be read.
export const PROSE_KINDS = ["description", "comments"];

export const DRAG_THRESHOLD = 5;

export const MINIMAP_W = 180;
export const MINIMAP_H = 120;
export const MINIMAP_PAD = 2;

export const nodePalette = {
  trigger:     { fill: '#ffffff', stroke: '#333333', hoverFill: '#e0e0e0', highlightFill: '#f0f0f0' },
  command:     { fill: '#dae8fc', stroke: '#6c8ebf', hoverFill: '#b8cce8', highlightFill: '#d4e4f7' },
  event:       { fill: '#ffe6cc', stroke: '#d79b00', hoverFill: '#f5d0b0', highlightFill: '#ffe5c2' },
  view:        { fill: '#d5e8d4', stroke: '#82b366', hoverFill: '#bfd8be', highlightFill: '#d2e6d1' },
  automation:  { fill: '#e1d5e7', stroke: '#9673a6', hoverFill: '#c0b0cc', highlightFill: '#d5c8da' },
  translation: { fill: '#f5f5f5', stroke: '#666666', hoverFill: '#d0d0d0', highlightFill: '#e0e0e0' },
};

export const edgeConfig = {
  flow:               { cls: "flow-arrow",      marker: "url(#arrowhead)",         stroke: "#666666", dash: "" },
  subscription:       { cls: "sub-arrow",       marker: "url(#arrowhead-green)",   stroke: "#82b366", dash: "5,5" },
  automation_trigger: { cls: "auto-trg-arrow",  marker: "url(#arrowhead-red)",     stroke: "#b85450", dash: "5,5" },
  automation_command: { cls: "auto-cmd-arrow",  marker: "url(#arrowhead-red)",     stroke: "#b85450", dash: "" },
  trigger_command:    { cls: "trg-cmd-arrow",   marker: "url(#arrowhead-purple)",  stroke: "#9673a6", dash: "" },
  reads:              { cls: "reads-arrow",     marker: "url(#arrowhead)",         stroke: "#666666", dash: "" },
  translation_command:{ cls: "trans-cmd-arrow", marker: "url(#arrowhead-orange)", stroke: "#d79b00", dash: "" },
};

export const arrowClassMap = {
  flow: "flow-arrow",
  subscription: "sub-arrow",
  automation_trigger: "auto-trg-arrow",
  automation_command: "auto-cmd-arrow",
  trigger_command: "trg-cmd-arrow",
  reads: "reads-arrow",
  translation_command: "trans-cmd-arrow",
};
