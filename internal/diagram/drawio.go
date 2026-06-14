// Package diagram renders AST models as diagrams.
package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// Style represents the layout strategy for diagram generation.
type Style int

const (
	// StyleAuto auto-detects the layout based on the context mode.
	StyleAuto Style = iota
	// StyleProjected uses the traditional aggregate-projected layout.
	StyleProjected
	// StyleDCB uses the direct-context-bound (DCB) slice layout.
	StyleDCB
)

// ParseStyle parses a style string into a Style value.
// Valid values: "auto", "projected", "dcb".
func ParseStyle(s string) (Style, error) {
	switch strings.ToLower(s) {
	case "auto":
		return StyleAuto, nil
	case "projected":
		return StyleProjected, nil
	case "dcb":
		return StyleDCB, nil
	default:
		return StyleAuto, fmt.Errorf("unsupported style %q: valid values are auto, projected, dcb", s)
	}
}

// String returns the string representation of the style.
func (s Style) String() string {
	switch s {
	case StyleProjected:
		return "projected"
	case StyleDCB:
		return "dcb"
	default:
		return "auto"
	}
}

// Layout constants.
const (
	marginX    = 40
	marginY    = 60
	sliceWidth = 280
	boxWidth   = 240
	boxHeight  = 55
	sliceGap   = 40
	contextGap = 70
	laneHeight = 190
	laneGap    = 30

	waypointMargin = 40
)

// Color constants for element types.
const (
	fillEvent       = "#ffe6cc"
	strokeEvent     = "#d79b00"
	fillCommand     = "#dae8fc"
	strokeCommand   = "#6c8ebf"
	fillView        = "#d5e8d4"
	strokeView      = "#82b366"
	fillTrigger     = "#ffffff"
	strokeTrigger   = "#333333"
	fillExternal    = "#f5f5f5"
	strokeExternal  = "#666666"
	fillReactor     = "#e1d5e7"
	strokeReactor   = "#9673a6"
	strokePurpleUp  = "#9B59B6"
	strokeGreenUp   = "#82b366"
	strokeStandard  = "#333333"
)

// ExportDrawio converts a parsed AST model into draw.io XML (mxGraph format).
func ExportDrawio(model *ast.Model, _ Style) ([]byte, error) {
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
		sliceIdx   int
		name       string
		id         int
		x, y, w, h int
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
			st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontFamily=Helvetica;",
				fillTrigger, strokeTrigger)
			b.WriteString(vertexCell(id, label, x, triggerCenterY, boxWidth, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: s.Trigger.Name, id: id, x: x, y: triggerCenterY, w: boxWidth, h: boxHeight})
		}

		// --- Commands (middle lane) ---
		totalMid := len(s.Commands) + len(s.Views)
		usableW := sliceWidth - 20
		for ci, cmd := range s.Commands {
			id := allocID()
			itemW, x := itemLayout(usableW, totalMid, ci, sliceX)
			st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontFamily=Helvetica;",
				fillCommand, strokeCommand)
			b.WriteString(vertexCell(id, cmd.Name, x, midCenterY, itemW, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: cmd.Name, id: id, x: x, y: midCenterY, w: itemW, h: boxHeight})
		}

		// --- Views (middle lane) ---
		for vi, view := range s.Views {
			id := allocID()
			idx := len(s.Commands) + vi
			itemW, x := itemLayout(usableW, totalMid, idx, sliceX)
			st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontFamily=Helvetica;",
				fillView, strokeView)
			b.WriteString(vertexCell(id, view.Name, x, midCenterY, itemW, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: view.Name, id: id, x: x, y: midCenterY, w: itemW, h: boxHeight})
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
			st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontFamily=Helvetica;",
				fillEvent, strokeEvent)
			b.WriteString(vertexCell(id, label, x, eventCenterY, itemW, boxHeight, st))
			elems = append(elems, namedElem{sliceIdx: i, name: evt.Name, id: id, x: x, y: eventCenterY, w: itemW, h: boxHeight})
		}
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				id := allocID()
				itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
				ei++
				st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontFamily=Helvetica;",
					fillEvent, strokeEvent)
				b.WriteString(vertexCell(id, tr.Event.Name, x, eventCenterY, itemW, boxHeight, st))
				elems = append(elems, namedElem{sliceIdx: i, name: tr.Event.Name, id: id, x: x, y: eventCenterY, w: itemW, h: boxHeight})
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
			st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontFamily=Helvetica;",
				fillReactor, strokeReactor)
			b.WriteString(vertexCell(id, label, x, y, autoW-boxWidth/8, autoH, st))
			elems = append(elems, namedElem{sliceIdx: i, name: auto.Name, id: id, x: x, y: y, w: autoW - boxWidth/8, h: autoH})
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
			st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;fontFamily=Helvetica;",
				fillReactor, strokeReactor)
			b.WriteString(vertexCell(id, label, x, y, reactorW-boxWidth/8, reactorH, st))
			elems = append(elems, namedElem{sliceIdx: i, name: tr.Name, id: id, x: x, y: y, w: reactorW - boxWidth/8, h: reactorH})
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
			st := fmt.Sprintf("rounded=0;whiteSpace=wrap;html=1;fillColor=%s;strokeColor=%s;dashed=1;fontFamily=Helvetica;",
				fillExternal, strokeExternal)
			b.WriteString(vertexCell(id, tr.ExternalSystem, extX, extY, extW, extH, st))
			elems = append(elems, namedElem{sliceIdx: i, name: tr.ExternalSystem, id: id, x: extX, y: extY, w: extW, h: extH})
		}
	}

	// --- Connections ---
	// Style definitions per guideline.
	standardStyle := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;fontFamily=Helvetica;strokeColor=" + strokeStandard + ";endArrow=classic;"
	purpleUpStyle := "edgeStyle=orthogonalEdgeStyle;html=1;fontFamily=Helvetica;strokeColor=" + strokePurpleUp + ";fontSize=10;endArrow=classic;exitX=1;exitY=0.5;exitDx=0;exitDy=0;curved=1;"
	greenUpStyle := "edgeStyle=orthogonalEdgeStyle;html=1;fontFamily=Helvetica;strokeColor=" + strokeGreenUp + ";fontSize=10;endArrow=classic;exitX=1;exitY=0.5;exitDx=0;exitDy=0;entryX=0;entryY=1;entryDx=0;entryDy=0;curved=1;"
	extStyle := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;fontFamily=Helvetica;strokeColor=" + strokeExternal + ";dashed=1;endArrow=classic;fontSize=10;"

	// Global element lookup across all slices (needed for cross-slice references)
	nameToElem := make(map[string]*namedElem)
	for _, e := range elems {
		nameToElem[e.name] = &e
	}

	for _, entry := range entries {
		s := entry.slice

		// trigger -> command (downward, standard)
		if s.Trigger != nil {
			tid := nameToElem[s.Trigger.Name]
			for _, cmd := range s.Commands {
				c := nameToElem[cmd.Name]
				if tid != nil && c != nil {
					b.WriteString(edgeCell(allocID(), standardStyle, tid.id, c.id))
				}
			}
		}

		// command -> event (downward, standard)
		for _, flow := range s.Flows {
			c := nameToElem[flow.CommandName]
			e := nameToElem[flow.EventName]
			if c != nil && e != nil {
				b.WriteString(edgeCell(allocID(), standardStyle, c.id, e.id))
			}
		}

		// event -> view (upward, green curved with waypoints)
		for _, view := range s.Views {
			v := nameToElem[view.Name]
			if v == nil {
				continue
			}
			for _, sub := range view.Subscribes {
				e := nameToElem[sub]
				if e == nil {
					continue
				}
				rightX := e.x + e.w + waypointMargin
				midY := e.y + e.h/2
				points := [][2]int{
					{rightX, midY},
					{v.x, midY},
				}
				b.WriteString(edgeCellWaypoints(allocID(), greenUpStyle, e.id, v.id, points))
			}
		}

		// event -> automation -> command
		for _, auto := range s.Automations {
			e := nameToElem[auto.TriggerEvent]
			a := nameToElem[auto.Name]
			c := nameToElem[auto.Command]

			// event -> automation (upward, purple curved)
			if e != nil && a != nil {
				rightX := e.x + e.w + waypointMargin
				srcY := e.y + e.h/2
				tgtY := a.y + a.h/2
				points := [][2]int{
					{rightX, srcY},
					{rightX, tgtY},
				}
				b.WriteString(edgeCellWaypoints(allocID(), purpleUpStyle, e.id, a.id, points))
			}

			// automation -> command (downward, standard)
			if a != nil && c != nil {
				b.WriteString(edgeCell(allocID(), standardStyle, a.id, c.id))
			}
		}

		// Translation: ext sys -> reactor -> command/event
		for _, tr := range s.Translations {
			extE := nameToElem[tr.ExternalSystem]
			rE := nameToElem[tr.Name]
			if extE == nil || rE == nil {
				continue
			}

			// reads: view -> external system (downward, standard)
			if tr.Reads != "" {
				v := nameToElem[tr.Reads]
				if v != nil {
					b.WriteString(edgeCell(allocID(), standardStyle, v.id, extE.id))
				}
			}

			// external system -> reactor (upward, dashed gray)
			b.WriteString(edgeCell(allocID(), extStyle, extE.id, rE.id))

			// reactor -> command (downward, standard)
			if tr.Command != "" {
				c := nameToElem[tr.Command]
				if c != nil {
					b.WriteString(edgeCell(allocID(), standardStyle, rE.id, c.id))
				}
			}

			// command -> event (translation implies command emits event)
			if tr.Command != "" && tr.Event != nil && tr.Event.Name != "" {
				c := nameToElem[tr.Command]
				e := nameToElem[tr.Event.Name]
				if c != nil && e != nil {
					b.WriteString(edgeCell(allocID(), standardStyle, c.id, e.id))
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
// It collects from both ctx.Aggregates[].Slices (traditional aggregate mode)
// and ctx.Slices (direct DCB mode), preserving order within each context.
func collectSlices(model *ast.Model) []sliceEntry {
	var entries []sliceEntry
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, s := range agg.Slices {
				entries = append(entries, sliceEntry{slice: s, ctxName: ctx.Name})
			}
		}
		for _, s := range ctx.Slices {
			entries = append(entries, sliceEntry{slice: s, ctxName: ctx.Name})
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

func edgeCellWaypoints(id int, style string, source, target int, points [][2]int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`        <mxCell id="%d" style="%s" edge="1" parent="1" source="%d" target="%d">`+"\n", id, style, source, target))
	sb.WriteString(`          <mxGeometry relative="1" as="geometry">` + "\n")
	sb.WriteString(`            <Array as="points">` + "\n")
	for _, p := range points {
		sb.WriteString(fmt.Sprintf(`              <mxPoint x="%d" y="%d" />`+"\n", p[0], p[1]))
	}
	sb.WriteString(`            </Array>` + "\n")
	sb.WriteString(`          </mxGeometry>` + "\n")
	sb.WriteString(`        </mxCell>` + "\n")
	return sb.String()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
