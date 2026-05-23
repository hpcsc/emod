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
	marginY    = 40
	sliceWidth = 280
	boxWidth   = 240
	boxHeight  = 55
	sliceGap   = 40
	laneHeight = 150
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

	// Diagram dimensions
	diagramW := 2*marginX + len(entries)*sliceWidth + (len(entries)-1)*sliceGap
	triggerLaneY := marginY
	cmdViewLaneY := triggerLaneY + laneHeight + laneGap
	eventLaneY := cmdViewLaneY + laneHeight + laneGap

	nextID := 2
	allocID := func() int {
		id := nextID
		nextID++
		return id
	}

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

	// Center Y within each lane's content area (below 30px label)
	triggerCenterY := triggerLaneY + 30 + (laneHeight-30-boxHeight)/2
	midCenterY := cmdViewLaneY + 30 + (laneHeight-30-boxHeight)/2
	eventCenterY := eventLaneY + 30 + (laneHeight-30-boxHeight)/2

	type namedElem struct {
		sliceIdx int
		name     string
		id       int
	}
	var elems []namedElem

	elemID := func(sliceIdx int, name string) int {
		for _, e := range elems {
			if e.sliceIdx == sliceIdx && e.name == name {
				return e.id
			}
		}
		return 0
	}

	// Place elements per slice
	for i, entry := range entries {
		s := entry.slice
		sliceX := marginX + i*(sliceWidth+sliceGap)

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

		// --- Events (bottom lane) ---
		numEvts := len(s.Events)
		usableW = sliceWidth - 20
		for ei, evt := range s.Events {
			id := allocID()
			itemW, x := itemLayout(usableW, numEvts, ei, sliceX)
			label := evt.Name
			if evt.ExternalName != "" {
				label = fmt.Sprintf("%s\\n[%s]", evt.Name, evt.ExternalName)
			}
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
				fillEvent, strokeEvent)
			b.WriteString(vertexCell(id, label, x, eventCenterY, itemW, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: evt.Name, id: id})
		}

		// --- Automations (compact boxes with gear indicator) ---
		for ai, auto := range s.Automations {
			id := allocID()
			autoW := boxWidth / 2
			autoH := boxHeight / 2
			x := sliceX + sliceWidth - autoW - 10
			y := eventLaneY + laneHeight - autoH - 10
			if ai > 0 {
				x = sliceX + sliceWidth - (autoW+5)*(ai+1) - 10
			}
			label := fmt.Sprintf("⚙ %s", auto.Name)
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;",
				fillEvent, strokeEvent)
			b.WriteString(vertexCell(id, label, x, y, autoW, autoH, st))
			elems = append(elems, namedElem{sliceIdx: i, name: auto.Name, id: id})
		}

		// --- External system boxes (Translations) ---
		for ti, tr := range s.Translations {
			id := allocID()
			extW := 100
			extH := 45
			extX := sliceX + sliceWidth + 10
			extY := cmdViewLaneY + 10 + ti*(extH+8)
			if extX+extW > marginX+len(entries)*(sliceWidth+sliceGap) {
				// Place below the bottom lane if no room to the right
				extX = sliceX + 10
				extY = eventLaneY + laneHeight + 5
			}
			st := fmt.Sprintf("rounded=1;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;dashed=1;",
				fillExternal, strokeExternal)
			b.WriteString(vertexCell(id, tr.ExternalSystem, extX, extY, extW, extH, st))
			elems = append(elems, namedElem{sliceIdx: i, name: tr.ExternalSystem, id: id})
		}
	}

	// --- Connections ---
	edgeStyle := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;"

	for i, entry := range entries {
		s := entry.slice

		// trigger -> command
		if s.Trigger != nil {
			tid := elemID(i, s.Trigger.Name)
			for _, cmd := range s.Commands {
				cid := elemID(i, cmd.Name)
				if tid > 0 && cid > 0 {
					b.WriteString(edgeCell(allocID(), edgeStyle, tid, cid))
				}
			}
		}

		// command -> event (via Flow entries)
		for _, flow := range s.Flows {
			cid := elemID(i, flow.CommandName)
			eid := elemID(i, flow.EventName)
			if cid > 0 && eid > 0 {
				b.WriteString(edgeCell(allocID(), edgeStyle, cid, eid))
			}
		}

		// event -> view (via subscribes)
		for _, evt := range s.Events {
			eid := elemID(i, evt.Name)
			if eid == 0 {
				continue
			}
			for _, view := range s.Views {
				for _, sub := range view.Subscribes {
					if sub == evt.Name {
						vid := elemID(i, view.Name)
						if vid > 0 {
							b.WriteString(edgeCell(allocID(), edgeStyle, eid, vid))
						}
					}
				}
			}
		}

		// event -> automation -> command
		for _, auto := range s.Automations {
			eid := elemID(i, auto.TriggerEvent)
			aid := elemID(i, auto.Name)
			cid := elemID(i, auto.Command)
			if eid > 0 && aid > 0 {
				b.WriteString(edgeCell(allocID(), edgeStyle, eid, aid))
			}
			if aid > 0 && cid > 0 {
				b.WriteString(edgeCell(allocID(), edgeStyle, aid, cid))
			}
		}

		// Translation: command -> external system -> event
		for _, tr := range s.Translations {
			extID := elemID(i, tr.ExternalSystem)
			if extID == 0 {
				continue
			}
			if tr.Command != "" {
				cid := elemID(i, tr.Command)
				if cid > 0 {
					b.WriteString(edgeCell(allocID(), edgeStyle, cid, extID))
				}
			}
			if tr.Event != nil && tr.Event.Name != "" {
				eid := elemID(i, tr.Event.Name)
				if eid > 0 {
					b.WriteString(edgeCell(allocID(), edgeStyle, extID, eid))
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
