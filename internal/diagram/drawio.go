// Package diagram renders AST models as diagrams.
package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// Layout constants.
const (
	marginX    = 40
	marginY    = 60
	sliceWidth = 280
	boxWidth   = 240
	boxHeight  = 55
	sliceGap   = 40
	contextGap  = 70
	laneHeight = 190
	laneGap    = 30
)

// Color constants for element types.
const (
	fillEvent    = "#f8cecc"
	strokeEvent  = "#b85450"
	fillCommand  = "#dae8fc"
	strokeCommand = "#6c8ebf"
	fillView     = "#d5e8d4"
	strokeView   = "#82b366"
	fillTrigger  = "#ffffff"
	strokeTrigger = "#000000"
	fillExternal = "#f5f5f5"
	strokeExternal = "#666666"
	fillReactor  = "#e1d5e7"
	strokeReactor = "#9673a6"
)

// ExportDrawio converts a parsed AST model into draw.io XML (mxGraph format).
func ExportDrawio(model *ast.Model) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	var b strings.Builder

	entries := collectSlices(model)
	if len(entries) == 0 {
		return buildEmptyDiagram(model.Name), nil
	}

	nextID := 2
	allocID := func() int {
		id := nextID
		nextID++
		return id
	}

	xPos := marginX
	prevCtx := ""
	var ctxBounds []struct {
		name string
		x    int
		w    int
	}
	for i, entry := range entries {
		if i > 0 {
			if entry.ctxName != prevCtx {
				if len(ctxBounds) > 0 {
					ctxBounds[len(ctxBounds)-1].w = xPos - ctxBounds[len(ctxBounds)-1].x - contextGap
				}
				xPos += contextGap
				ctxBounds = append(ctxBounds, struct {
					name string
					x    int
					w    int
				}{name: entry.ctxName, x: xPos})
			} else {
				xPos += sliceGap
			}
		} else {
			ctxBounds = append(ctxBounds, struct {
				name string
				x    int
				w    int
			}{name: entry.ctxName, x: xPos})
		}
		xPos += sliceWidth
		prevCtx = entry.ctxName
	}
	if len(ctxBounds) > 0 {
		ctxBounds[len(ctxBounds)-1].w = xPos - ctxBounds[len(ctxBounds)-1].x
	}

	diagramW := xPos + marginX
	triggerLaneY := marginY
	cmdViewLaneY := triggerLaneY + laneHeight + laneGap
	eventLaneY := cmdViewLaneY + laneHeight + laneGap
	extLaneY := eventLaneY + laneHeight + laneGap

	// Write document header
	b.WriteString(xmlProlog(model.Name))
	b.WriteString(rootOpen())

	// Write three swimlanes
	topLaneID := allocID()
	b.WriteString(swimlaneCell(topLaneID, "UI / Triggers",
		marginX, triggerLaneY, diagramW-2*marginX, laneHeight))
	midLaneID := allocID()
	b.WriteString(swimlaneCell(midLaneID, "Commands / Views",
		marginX, cmdViewLaneY, diagramW-2*marginX, laneHeight))
	botLaneID := allocID()
	b.WriteString(swimlaneCell(botLaneID, "Events",
		marginX, eventLaneY, diagramW-2*marginX, laneHeight))
	extLaneID := allocID()
	b.WriteString(swimlaneCell(extLaneID, "External Systems",
		marginX, extLaneY, diagramW-2*marginX, laneHeight))

	// Context labels above the swimlanes
	for _, cb := range ctxBounds {
		cid := allocID()
		label := escapeXML(cb.name)
		st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontStyle=1;",
			fillExternal, strokeExternal)
		b.WriteString(vertexCell(cid, label, cb.x, marginY-30, cb.w-20, 22, st))
	}

	triggerCenterY := triggerLaneY + 30 + (laneHeight-30-boxHeight)/2
	midCenterY := cmdViewLaneY + 30 + (laneHeight-30-boxHeight)/2
	eventCenterY := eventLaneY + 30 + (laneHeight-30-boxHeight)/2
	extCenterY := extLaneY + 30 + (laneHeight-30-boxHeight)/2

	type namedElem struct {
		sliceIdx int
		name     string
		id       int
	}
	var elems []namedElem

	// Precompute X position per entry, accounting for context gaps
	sliceXFor := make(map[int]int)
	xp := marginX
	prev := ""
	for ei, entry := range entries {
		if ei > 0 {
			if entry.ctxName != prev {
				xp += contextGap
			} else {
				xp += sliceGap
			}
		}
		sliceXFor[ei] = xp
		xp += sliceWidth
		prev = entry.ctxName
	}

	// Place elements per slice
	for i, entry := range entries {
		s := entry.slice
		sliceX := sliceXFor[i]

		// --- Trigger (top lane) ---
		if s.Trigger != nil {
			id := allocID()
			x := sliceX + (sliceWidth-boxWidth)/2
			label := s.Trigger.Name
			if s.Trigger.Actor != "" {
				label = fmt.Sprintf("%s (%s)", s.Trigger.Name, s.Trigger.Actor)
			}
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
				fillTrigger, strokeTrigger)
			b.WriteString(vertexCell(id, label, x, triggerCenterY, boxWidth, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: s.Trigger.Name, id: id})
		}

		// --- Commands (middle lane) ---
		totalMid := len(s.Commands) + len(s.Views)
		usableW := sliceWidth - 20
		for ci, cmd := range s.Commands {
			id := allocID()
			itemW, x := itemLayout(usableW, totalMid, ci, sliceX)
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
				fillCommand, strokeCommand)
			b.WriteString(vertexCell(id, cmd.Name, x, midCenterY, itemW, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: cmd.Name, id: id})
		}

		// --- Views (middle lane) ---
		for vi, view := range s.Views {
			id := allocID()
			idx := len(s.Commands) + vi
			itemW, x := itemLayout(usableW, totalMid, idx, sliceX)
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
				fillView, strokeView)
			b.WriteString(vertexCell(id, view.Name, x, midCenterY, itemW, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: view.Name, id: id})
		}

		// --- Events (bottom lane, including translation events) ---
		usableW = sliceWidth - 20
		totalEvts := len(s.Events)
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				totalEvts++
			}
		}
		ei := 0
		for me, evt := range s.Events {
			id := allocID()
			itemW, x := itemLayout(usableW, totalEvts, me, sliceX)
			label := evt.Name
			if evt.ExternalName != "" {
				label = fmt.Sprintf("%s\\n[%s]", evt.Name, evt.ExternalName)
			}
			ei++
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
				fillEvent, strokeEvent)
			b.WriteString(vertexCell(id, label, x, eventCenterY, itemW, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: evt.Name, id: id})
		}
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				id := allocID()
				itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
				ei++
				st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
					fillEvent, strokeEvent)
				b.WriteString(vertexCell(id, tr.Event.Name, x, eventCenterY, itemW, boxHeight, st))
				elems = append(elems, namedElem{sliceIdx: i, name: tr.Event.Name, id: id})
			}
		}

		// --- Automations (compact boxes with gear indicator) ---
		for ai, auto := range s.Automations {
			id := allocID()
			autoW := boxWidth
			autoH := boxHeight * 3 / 4
			autoPadX := 10
			autoPadY := 15 + ai*(autoH+5)
			x := sliceX + autoPadX
			y := triggerLaneY + laneHeight - autoH - autoPadY
			label := fmt.Sprintf("⚙ %s", auto.Name)
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
				fillReactor, strokeReactor)
			b.WriteString(vertexCell(id, label, x, y, autoW-boxWidth/8, autoH, st))
			elems = append(elems, namedElem{sliceIdx: i, name: auto.Name, id: id})
		}

		// --- Translation reactors (in UI/Triggers lane, below automations) ---
		for ti, tr := range s.Translations {
			id := allocID()
			reactorW := boxWidth
			reactorH := boxHeight * 3 / 4
			padX := 10
			padY := 15 + (len(s.Automations)+ti)*(reactorH+5)
			x := sliceX + padX
			y := triggerLaneY + laneHeight - reactorH - padY
			label := fmt.Sprintf("⚙ %s", tr.Name)
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
				fillReactor, strokeReactor)
			b.WriteString(vertexCell(id, label, x, y, reactorW-boxWidth/8, reactorH, st))
			elems = append(elems, namedElem{sliceIdx: i, name: tr.Name, id: id})
		}

		// --- External system boxes (Translations) ---
		for ti, tr := range s.Translations {
			id := allocID()
			extW := 100
			extH := 45
			extX := sliceX + (sliceWidth-extW)/2
			extY := extCenterY - extH/2
			if ti > 0 {
				extY += ti * (extH + 8)
			}
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;dashed=1;",
				fillExternal, strokeExternal)
			b.WriteString(vertexCell(id, tr.ExternalSystem, extX, extY, extW, extH, st))
			elems = append(elems, namedElem{sliceIdx: i, name: tr.ExternalSystem, id: id})
		}
	}

	// --- Connections ---
	edgeStyle := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;"

	// Global element lookup across all slices (needed for cross-slice references)
	nameToID := make(map[string]int)
	for _, e := range elems {
		nameToID[e.name] = e.id
	}

	for _, entry := range entries {
		s := entry.slice

		// trigger -> command
		if s.Trigger != nil {
			tid := nameToID[s.Trigger.Name]
			for _, cmd := range s.Commands {
				cid := nameToID[cmd.Name]
				if tid > 0 && cid > 0 {
					b.WriteString(edgeCell(allocID(), edgeStyle, tid, cid))
				}
			}
		}

		// command -> event (via Flow entries)
		for _, flow := range s.Flows {
			cid := nameToID[flow.CommandName]
			eid := nameToID[flow.EventName]
			if cid > 0 && eid > 0 {
				b.WriteString(edgeCell(allocID(), edgeStyle, cid, eid))
			}
		}

		// event -> view (via subscribes) — cross-slice lookup
		for _, view := range s.Views {
			vid := nameToID[view.Name]
			if vid == 0 {
				continue
			}
			for _, sub := range view.Subscribes {
				eid := nameToID[sub]
				if eid > 0 {
					b.WriteString(edgeCell(allocID(), edgeStyle, eid, vid))
				}
			}
		}

		// event -> automation -> command — cross-slice lookup
		for _, auto := range s.Automations {
			eid := nameToID[auto.TriggerEvent]
			aid := nameToID[auto.Name]
			cid := nameToID[auto.Command]
			if eid > 0 && aid > 0 {
				b.WriteString(edgeCell(allocID(), edgeStyle, eid, aid))
			}
			if aid > 0 && cid > 0 {
				b.WriteString(edgeCell(allocID(), edgeStyle, aid, cid))
			}
		}

		// Translation: ext sys -> reactor -> command/event
		for _, tr := range s.Translations {
			extID := nameToID[tr.ExternalSystem]
			reactorID := nameToID[tr.Name]
			if extID == 0 || reactorID == 0 {
				continue
			}
			// reads: view -> external system
			if tr.Reads != "" {
				vid := nameToID[tr.Reads]
				if vid > 0 {
					b.WriteString(edgeCell(allocID(), edgeStyle, vid, extID))
				}
			}
			// external system -> reactor
			b.WriteString(edgeCell(allocID(), edgeStyle, extID, reactorID))
			// reactor -> command
			if tr.Command != "" {
				cid := nameToID[tr.Command]
				if cid > 0 {
					b.WriteString(edgeCell(allocID(), edgeStyle, reactorID, cid))
				}
			}
			// command -> event (translation implies command emits event)
			if tr.Command != "" && tr.Event != nil && tr.Event.Name != "" {
				cid := nameToID[tr.Command]
				eid := nameToID[tr.Event.Name]
				if cid > 0 && eid > 0 {
					b.WriteString(edgeCell(allocID(), edgeStyle, cid, eid))
				}
			}
		}
	}

	b.WriteString(rootClose())
	b.WriteString(xmlEpilog())

	return []byte(b.String()), nil
}

type sliceEntry struct {
	slice   *ast.Slice
	ctxName string
}

// collectSlices flattens all slices from the model into a list.
func collectSlices(model *ast.Model) []sliceEntry {
	var entries []sliceEntry
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, s := range agg.Slices {
				entries = append(entries, sliceEntry{slice: s, ctxName: ctx.Name})
			}
		}
	}
	return entries
}

// itemLayout computes item width and x position for elements within a slice.
func itemLayout(usableW int, numItems int, index int, sliceX int) (int, int) {
	itemW := boxWidth
	if numItems > 1 {
		itemW = (usableW - (numItems-1)*8) / numItems
		if itemW > boxWidth {
			itemW = boxWidth
		}
	}
	x := sliceX + 10 + index*(itemW+8)
	return itemW, x
}

// buildEmptyDiagram returns XML for a model with no slices.
func buildEmptyDiagram(name string) []byte {
	var b strings.Builder
	b.WriteString(xmlProlog(name))
	b.WriteString(rootOpen())
	b.WriteString(rootClose())
	b.WriteString(xmlEpilog())
	return []byte(b.String())
}

// --- XML cell builders ---

func xmlProlog(modelName string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"+
		`<mxfile host="emod" version="1.0">`+"\n"+
		`  <diagram name="%s">`+"\n"+
		`    <mxGraphModel dx="0" dy="0" grid="0" gridSize="10">`+"\n", modelName)
}

func rootOpen() string {
	return `      <root>` + "\n" +
		`        <mxCell id="0" />` + "\n" +
		`        <mxCell id="1" parent="0" />` + "\n"
}

func rootClose() string {
	return `      </root>` + "\n"
}

func xmlEpilog() string {
	return `    </mxGraphModel>` + "\n" +
		`  </diagram>` + "\n" +
		`</mxfile>` + "\n"
}

func swimlaneCell(id int, label string, x, y, w, h int) string {
	style := "swimlane;horizontal=0;startSize=30;container=1;collapsible=0;" +
		"rounded=1;whiteSpace=wrap;html=1;fillColor=#ffffff;strokeColor=#000000;"
	return fmt.Sprintf(`        <mxCell id="%d" value="%s" style="%s" vertex="1" parent="1">`+"\n"+
		`          <mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry" />`+"\n"+
		`        </mxCell>`+"\n", id, label, style, x, y, w, h)
}

func vertexCell(id int, value string, x, y, w, h int, style string) string {
	return fmt.Sprintf(`        <mxCell id="%d" value="%s" style="%s" vertex="1" parent="1">`+"\n"+
		`          <mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry" />`+"\n"+
		`        </mxCell>`+"\n", id, escapeXML(value), style, x, y, w, h)
}

func edgeCell(id int, style string, source, target int) string {
	return fmt.Sprintf(`        <mxCell id="%d" style="%s" edge="1" parent="1" source="%d" target="%d">`+"\n"+
		`          <mxGeometry relative="1" as="geometry" />`+"\n"+
		`        </mxCell>`+"\n", id, style, source, target)
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
