// Package diagram renders AST models as diagrams.
package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// ExportSVG generates a self-contained SVG diagram from a parsed AST model.
func ExportSVG(model *ast.Model, _ Style) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	entries := collectSlices(model)

	sliceXs := sliceXPositions(entries)
	ctxBounds := contextBounds(entries, sliceXs)

	diagramW := layoutWidth(sliceXs) + marginX + 120
	diagramH := 2*marginY + 4*laneHeight + 3*laneGap

	var b strings.Builder
	b.WriteString(svgHeader(diagramW, diagramH))
	b.WriteString(svgDefs())

	if len(entries) == 0 {
		b.WriteString("</svg>\n")
		return []byte(b.String()), nil
	}

	triggerLaneY := marginY
	cmdViewLaneY := triggerLaneY + laneHeight + laneGap
	eventLaneY := cmdViewLaneY + laneHeight + laneGap
	extLaneY := eventLaneY + laneHeight + laneGap

	laneW := diagramW - 2*marginX
	b.WriteString(svgLane(triggerLaneY, laneW, "Wireframes"))
	b.WriteString(svgLane(cmdViewLaneY, laneW, "Commands / Views"))
	b.WriteString(svgLane(eventLaneY, laneW, "Events"))
	b.WriteString(svgLane(extLaneY, laneW, "External Systems"))

	for _, cb := range ctxBounds {
		b.WriteString(svgRect(cb.x, marginY-30, cb.w-20, 22, fillExternal, strokeExternal, 0, cb.description))
		b.WriteString(svgText(cb.x+(cb.w-20)/2, marginY-19, cb.name, 12, strokeExternal))
	}

	triggerRowY := laneRowY(triggerLaneY)
	cmdViewRowY := laneRowY(cmdViewLaneY)
	eventRowY := laneRowY(eventLaneY)
	extRowY := laneRowY(extLaneY)

	// Where the box drawn for each name ended up. One map across every slice, so
	// a connection can reach a box another slice drew.
	nameToBox := make(map[string]svgBox)

	// Place elements per slice
	for i, entry := range entries {
		s := entry.slice
		sliceX := sliceXs[i]

		// --- Trigger (top lane) ---
		if s.Trigger != nil {
			x := sliceX + (sliceWidth-boxWidth)/2
			y := triggerRowY
			label := s.Trigger.Name
			if s.Trigger.Actor != "" {
				label = fmt.Sprintf("%s (%s)", s.Trigger.Name, s.Trigger.Actor)
			}
			b.WriteString(svgRoundedRect(x, y, boxWidth, boxHeight, fillTrigger, strokeTrigger, 5, s.Trigger.Description))
			b.WriteString(svgText(x+boxWidth/2, y+boxHeight/2, label, 12, strokeTrigger))
			nameToBox[s.Trigger.Name] = svgBox{x: x, y: y, w: boxWidth, h: boxHeight}
		}

		// --- Commands (middle lane) ---
		totalMid := len(s.Commands) + len(s.Views)
		usableW := sliceWidth - 20
		for ci, cmd := range s.Commands {
			itemW, x := itemLayout(usableW, totalMid, ci, sliceX)
			b.WriteString(svgRoundedRect(x, cmdViewRowY, itemW, boxHeight, fillCommand, strokeCommand, 5, cmd.Description))
			b.WriteString(svgText(x+itemW/2, cmdViewRowY+boxHeight/2, cmd.Name, 12, strokeCommand))
			nameToBox[cmd.Name] = svgBox{x: x, y: cmdViewRowY, w: itemW, h: boxHeight}
		}

		// --- Views (middle lane) ---
		for vi, view := range s.Views {
			idx := len(s.Commands) + vi
			itemW, x := itemLayout(usableW, totalMid, idx, sliceX)
			b.WriteString(svgRoundedRect(x, cmdViewRowY, itemW, boxHeight, fillView, strokeView, 5, view.Description))
			b.WriteString(svgText(x+itemW/2, cmdViewRowY+boxHeight/2, view.Name, 12, strokeView))
			nameToBox[view.Name] = svgBox{x: x, y: cmdViewRowY, w: itemW, h: boxHeight}
		}

		// --- Events (bottom lane, including translation events) ---
		totalEvts := len(s.Events)
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				totalEvts++
			}
		}
		ei := 0
		for me, evt := range s.Events {
			itemW, x := itemLayout(usableW, totalEvts, me, sliceX)
			label := evt.Name
			if evt.ExternalName != "" {
				label = fmt.Sprintf("%s\n[%s]", evt.Name, evt.ExternalName)
			}
			ei++
			b.WriteString(svgRoundedRect(x, eventRowY, itemW, boxHeight, fillEvent, strokeEvent, 5, evt.Description))
			b.WriteString(svgMultilineText(x+itemW/2, eventRowY+boxHeight/2, label, 11, strokeEvent))
			nameToBox[evt.Name] = svgBox{x: x, y: eventRowY, w: itemW, h: boxHeight}
		}
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
				ei++
				b.WriteString(svgRoundedRect(x, eventRowY, itemW, boxHeight, fillEvent, strokeEvent, 5, tr.Event.Description))
				b.WriteString(svgText(x+itemW/2, eventRowY+boxHeight/2, tr.Event.Name, 11, strokeEvent))
				nameToBox[tr.Event.Name] = svgBox{x: x, y: eventRowY, w: itemW, h: boxHeight}
			}
		}

		// --- Automations and translation reactors (middle lane) ---
		for _, reactor := range reactorBoxes(s, cmdViewLaneY, sliceX, "\n") {
			b.WriteString(svgRoundedRect(reactor.x, reactor.y, reactor.w, reactor.h, fillReactor, strokeReactor, 3, reactor.description))
			b.WriteString(svgMultilineText(reactor.x+reactor.w/2, reactor.y+reactor.h/2, reactor.label, 10, strokeReactor))
			nameToBox[reactor.name] = svgBox{x: reactor.x, y: reactor.y, w: reactor.w, h: reactor.h}
		}

		// --- External system boxes (Translations) ---
		for ti, tr := range s.Translations {
			extW := 100
			extH := 45
			extX := sliceX + (sliceWidth-extW)/2
			extY := extRowY - extH/2
			if ti > 0 {
				extY += ti * (extH + 8)
			}
			b.WriteString(svgDashedRoundedRect(extX, extY, extW, extH, fillExternal, strokeExternal, 5, tr.Description))
			b.WriteString(svgText(extX+extW/2, extY+extH/2, tr.ExternalSystem, 11, strokeExternal))
			nameToBox[tr.ExternalSystem] = svgBox{x: extX, y: extY, w: extW, h: extH}
		}
	}

	// --- Connections ---

	readsArrow := func(reads string, reader svgBox) string {
		if reads == "" {
			return ""
		}
		view, declared := nameToBox[reads]
		if !declared {
			return ""
		}

		return svgArrowBetween(view, reader)
	}

	for _, entry := range entries {
		s := entry.slice

		// trigger -> command
		if s.Trigger != nil {
			if trigger, drawn := nameToBox[s.Trigger.Name]; drawn {
				b.WriteString(readsArrow(s.Trigger.Reads, trigger))
				for _, cmd := range s.Commands {
					if command, drawn := nameToBox[cmd.Name]; drawn {
						b.WriteString(svgArrowBetween(trigger, command))
					}
				}
			}
		}

		// command -> event (via Flow entries)
		for _, flow := range s.Flows {
			command, commandDrawn := nameToBox[flow.CommandName]
			evt, eventDrawn := nameToBox[flow.EventName]
			if commandDrawn && eventDrawn {
				b.WriteString(svgArrowBetween(command, evt))
			}
		}

		// event -> view (via subscribes) — cross-slice lookup
		for _, view := range s.Views {
			viewBox, drawn := nameToBox[view.Name]
			if !drawn {
				continue
			}
			for _, sub := range view.Subscribes {
				if evt, drawn := nameToBox[sub]; drawn {
					b.WriteString(svgArrowBetween(evt, viewBox))
				}
			}
		}

		// event -> automation -> command — cross-slice lookup
		for _, auto := range s.Automations {
			automation, drawn := nameToBox[auto.Name]
			if !drawn {
				continue
			}
			b.WriteString(readsArrow(auto.Reads, automation))
			if evt, drawn := nameToBox[auto.OnEvent]; drawn {
				b.WriteString(svgArrowBetween(evt, automation))
			}
			if command, drawn := nameToBox[auto.Command]; drawn {
				b.WriteString(svgArrowBetween(automation, command))
			}
		}

		// Translation: ext sys -> reactor -> command/event
		for _, tr := range s.Translations {
			externalSystem, externalDrawn := nameToBox[tr.ExternalSystem]
			reactor, reactorDrawn := nameToBox[tr.Name]
			if !externalDrawn || !reactorDrawn {
				continue
			}

			b.WriteString(readsArrow(tr.Reads, externalSystem))
			b.WriteString(svgArrowBetween(externalSystem, reactor))
			if tr.Command != "" {
				if command, drawn := nameToBox[tr.Command]; drawn {
					b.WriteString(svgArrowBetween(reactor, command))
				}
			}
			// command -> event (translation implies command emits event, unless a
			// flow already declares it and has drawn the arrow above)
			if tr.Command != "" && tr.Event != nil && tr.Event.Name != "" &&
				!declaresFlow(s, tr.Command, tr.Event.Name) {
				command, commandDrawn := nameToBox[tr.Command]
				evt, eventDrawn := nameToBox[tr.Event.Name]
				if commandDrawn && eventDrawn {
					b.WriteString(svgArrowBetween(command, evt))
				}
			}
		}
	}

	b.WriteString("</svg>\n")
	return []byte(b.String()), nil
}

// --- SVG helpers ---

func svgHeader(w, h int) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d">`+"\n", w, h)
}

func svgDefs() string {
	return `<defs>
<marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto">
<path d="M 0 0 L 10 5 L 0 10 z" fill="#666666"/>
</marker>
</defs>
`
}

func svgRect(x, y, w, h int, fill, stroke string, rx int, description string) string {
	return svgRectElement(svgRectAttributes(x, y, w, h, fill, stroke, rx), description)
}

func svgRoundedRect(x, y, w, h int, fill, stroke string, rx int, description string) string {
	return svgRect(x, y, w, h, fill, stroke, rx, description)
}

func svgDashedRoundedRect(x, y, w, h int, fill, stroke string, rx int, description string) string {
	return svgRectElement(svgRectAttributes(x, y, w, h, fill, stroke, rx)+` stroke-dasharray="5,3"`, description)
}

func svgRectAttributes(x, y, w, h int, fill, stroke string, rx int) string {
	return fmt.Sprintf(`x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="%s" rx="%d"`,
		x, y, w, h, fill, stroke, rx)
}

// A browser shows a description as a tooltip only when the <title> holding it is
// nested inside the shape, which a self-closing rect has no room for. A shape
// with nothing to say stays self-closing: opening those too would rewrite the
// diagram of every model that carries no prose.
func svgRectElement(attributes, description string) string {
	if description == "" {
		return fmt.Sprintf("<rect %s/>\n", attributes)
	}

	return fmt.Sprintf("<rect %s>\n<title>%s</title>\n</rect>\n", attributes, escapeXML(description))
}

func svgLane(y, w int, label string) string {
	return svgRect(marginX, y, w, laneHeight, "#ffffff", "#000000", 5, "") +
		svgLaneLabel(marginX+10, y+20, label)
}

func svgLaneLabel(x, y int, text string) string {
	return fmt.Sprintf(`<text x="%d" y="%d" font-family="sans-serif" font-size="14" font-weight="bold" fill="#000000">%s</text>`+"\n",
		x, y, escapeXML(text))
}

func svgText(x, y int, text string, fontSize int, color string) string {
	return fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="%d" fill="%s">%s</text>`+"\n",
		x, y, fontSize, color, escapeXML(text))
}

func svgMultilineText(x, y int, text string, fontSize int, color string) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= 1 {
		return svgText(x, y, text, fontSize, color)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="%d" fill="%s">`+"\n",
		x, y, fontSize, color))

	lineHeight := fontSize + 4
	totalH := (len(lines) - 1) * lineHeight

	for i, line := range lines {
		dy := lineHeight
		if i == 0 {
			dy = -totalH / 2
		}
		b.WriteString(fmt.Sprintf(`<tspan x="%d" dy="%d">%s</tspan>`+"\n", x, dy, escapeXML(line)))
	}
	b.WriteString("</text>\n")
	return b.String()
}

// svgBox is where a shape was drawn: the corner it starts at, and how far it
// runs from there.
type svgBox struct {
	x, y, w, h int
}

func svgArrowBetween(from, to svgBox) string {
	return svgArrowPath(from.x+from.w/2, from.y+from.h/2, to.x+to.w/2, to.y+to.h/2)
}

func svgArrowPath(sx, sy, tx, ty int) string {
	if sx == tx && sy == ty {
		return ""
	}

	if sx == tx {
		// Vertical only
		return fmt.Sprintf(`<path d="M %d,%d L %d,%d" fill="none" stroke="#666666" stroke-width="1.5" marker-end="url(#arrow)"/>`+"\n",
			sx, sy, tx, ty)
	}

	if sy == ty {
		// Horizontal only
		return fmt.Sprintf(`<path d="M %d,%d L %d,%d" fill="none" stroke="#666666" stroke-width="1.5" marker-end="url(#arrow)"/>`+"\n",
			sx, sy, tx, ty)
	}

	// Vertical-first orthogonal path
	midY := (sy + ty) / 2
	return fmt.Sprintf(`<path d="M %d,%d L %d,%d L %d,%d L %d,%d" fill="none" stroke="#666666" stroke-width="1.5" marker-end="url(#arrow)"/>`+"\n",
		sx, sy, sx, midY, tx, midY, tx, ty)
}
