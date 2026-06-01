export const L = {
  marginX: 40, marginY: 60,
  sliceWidth: 280, boxWidth: 240, boxHeight: 55,
  sliceGap: 40, contextGap: 70,
  laneHeight: 190, laneGap: 30,
  swimlaneHdr: 44, swimlanePad: 10,
  swimlaneGap: 20, aggLabelH: 22,
  sideGap: 12,
};

export const DRAG_THRESHOLD = 5;

export const MINIMAP_W = 180;
export const MINIMAP_H = 120;
export const MINIMAP_PAD = 2;

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
