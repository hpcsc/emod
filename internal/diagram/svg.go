// Package diagram renders AST models as diagrams.
package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// ExportSVG generates a self-contained SVG diagram from a parsed AST model.
func ExportSVG(model *ast.Model) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	entries := collectSlices(model)

	diagramW := 2*marginX + len(entries)*sliceWidth + (len(entries)-1)*sliceGap + 120
	diagramH := 2*marginY + 3*laneHeight + 2*laneGap

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

	// Swimlanes
	laneW := diagramW - 2*marginX
	b.WriteString(svgRect(marginX, triggerLaneY, laneW, laneHeight, "#ffffff", "#000000", 5))
	b.WriteString(svgLaneLabel(marginX+10, triggerLaneY+20, "UI / Triggers"))
	b.WriteString(svgRect(marginX, cmdViewLaneY, laneW, laneHeight, "#ffffff", "#000000", 5))
	b.WriteString(svgLaneLabel(marginX+10, cmdViewLaneY+20, "Commands / Views"))
	b.WriteString(svgRect(marginX, eventLaneY, laneW, laneHeight, "#ffffff", "#000000", 5))
	b.WriteString(svgLaneLabel(marginX+10, eventLaneY+20, "Events"))

	// Center Y within each lane's content area (below 30px label area)
	triggerCenterY := triggerLaneY + 30 + (laneHeight-30-boxHeight)/2
	midCenterY := cmdViewLaneY + 30 + (laneHeight-30-boxHeight)/2
	eventCenterY := eventLaneY + 30 + (laneHeight-30-boxHeight)/2

	type svgElem struct {
		sliceIdx int
		name     string
		x, y, w, h int
	}
	var elems []svgElem

	elemPos := func(sliceIdx int, name string) (int, int, int, int, bool) {
		for _, e := range elems {
			if e.sliceIdx == sliceIdx && e.name == name {
				return e.x, e.y, e.w, e.h, true
			}
		}
		return 0, 0, 0, 0, false
	}

	// Place elements per slice
	for i, entry := range entries {
		s := entry.slice
		sliceX := marginX + i*(sliceWidth+sliceGap)

		// --- Trigger (top lane) ---
		if s.Trigger != nil {
			x := sliceX + (sliceWidth-boxWidth)/2
			y := triggerCenterY
			label := s.Trigger.Name
			if s.Trigger.Actor != "" {
				label = fmt.Sprintf("%s (%s)", s.Trigger.Name, s.Trigger.Actor)
			}
			b.WriteString(svgRoundedRect(x, y, boxWidth, boxHeight, fillTrigger, strokeTrigger, 5))
			b.WriteString(svgText(x+boxWidth/2, y+boxHeight/2, label, 12, strokeTrigger))
			elems = append(elems, svgElem{
				sliceIdx: i, name: s.Trigger.Name,
				x: x, y: y, w: boxWidth, h: boxHeight,
			})
		}

		// --- Commands (middle lane) ---
		totalMid := len(s.Commands) + len(s.Views)
		usableW := sliceWidth - 20
		for ci, cmd := range s.Commands {
			itemW, x := itemLayout(usableW, totalMid, ci, sliceX)
			b.WriteString(svgRoundedRect(x, midCenterY, itemW, boxHeight, fillCommand, strokeCommand, 5))
			b.WriteString(svgText(x+itemW/2, midCenterY+boxHeight/2, cmd.Name, 12, strokeCommand))
			elems = append(elems, svgElem{
				sliceIdx: i, name: cmd.Name,
				x: x, y: midCenterY, w: itemW, h: boxHeight,
			})
		}

		// --- Views (middle lane) ---
		for vi, view := range s.Views {
			idx := len(s.Commands) + vi
			itemW, x := itemLayout(usableW, totalMid, idx, sliceX)
			b.WriteString(svgRoundedRect(x, midCenterY, itemW, boxHeight, fillView, strokeView, 5))
			b.WriteString(svgText(x+itemW/2, midCenterY+boxHeight/2, view.Name, 12, strokeView))
			elems = append(elems, svgElem{
				sliceIdx: i, name: view.Name,
				x: x, y: midCenterY, w: itemW, h: boxHeight,
			})
		}

		// --- Events (bottom lane) ---
		numEvts := len(s.Events)
		for ei, evt := range s.Events {
			itemW, x := itemLayout(usableW, numEvts, ei, sliceX)
			label := evt.Name
			if evt.ExternalName != "" {
				label = fmt.Sprintf("%s\n[%s]", evt.Name, evt.ExternalName)
			}
			b.WriteString(svgRoundedRect(x, eventCenterY, itemW, boxHeight, fillEvent, strokeEvent, 5))
			b.WriteString(svgMultilineText(x+itemW/2, eventCenterY+boxHeight/2, label, 11, strokeEvent))
			elems = append(elems, svgElem{
				sliceIdx: i, name: evt.Name,
				x: x, y: eventCenterY, w: itemW, h: boxHeight,
			})
		}

		// --- Automations (compact boxes with gear indicator) ---
		for ai, auto := range s.Automations {
			autoW := boxWidth / 2
			autoH := boxHeight / 2
			x := sliceX + sliceWidth - autoW - 10
			y := eventLaneY + laneHeight - autoH - 10
			if ai > 0 {
				x = sliceX + sliceWidth - (autoW+5)*(ai+1) - 10
			}
			label := fmt.Sprintf("\u2699 %s", auto.Name)
			b.WriteString(svgRoundedRect(x, y, autoW, autoH, fillEvent, strokeEvent, 3))
			b.WriteString(svgText(x+autoW/2, y+autoH/2, label, 10, strokeEvent))
			elems = append(elems, svgElem{
				sliceIdx: i, name: auto.Name,
				x: x, y: y, w: autoW, h: autoH,
			})
		}

		// --- External system boxes (Translations) ---
		for ti, tr := range s.Translations {
			extW := 100
			extH := 45
			extX := sliceX + sliceWidth + 10
			extY := cmdViewLaneY + 10 + ti*(extH+8)
			if extX+extW > diagramW-marginX {
				extX = sliceX + 10
				extY = eventLaneY + laneHeight + 5
			}
			b.WriteString(svgDashedRoundedRect(extX, extY, extW, extH, fillExternal, strokeExternal, 5))
			b.WriteString(svgText(extX+extW/2, extY+extH/2, tr.ExternalSystem, 11, strokeExternal))
			elems = append(elems, svgElem{
				sliceIdx: i, name: tr.ExternalSystem,
				x: extX, y: extY, w: extW, h: extH,
			})
		}
	}

	// --- Connections ---
	for i, entry := range entries {
		s := entry.slice

		// trigger -> command
		if s.Trigger != nil {
			tx, ty, tw, th, tok := elemPos(i, s.Trigger.Name)
			if tok {
				tcx, tcy := tx+tw/2, ty+th/2
				for _, cmd := range s.Commands {
					cx, cy, cw, ch, cok := elemPos(i, cmd.Name)
					if cok {
						ccx, ccy := cx+cw/2, cy+ch/2
						b.WriteString(svgArrowPath(tcx, tcy, ccx, ccy))
					}
				}
			}
		}

		// command -> event (via Flow entries)
		for _, flow := range s.Flows {
			cx, cy, cw, ch, cok := elemPos(i, flow.CommandName)
			ex, ey, ew, eh, eok := elemPos(i, flow.EventName)
			if cok && eok {
				ccx, ccy := cx+cw/2, cy+ch/2
				ecx, ecy := ex+ew/2, ey+eh/2
				b.WriteString(svgArrowPath(ccx, ccy, ecx, ecy))
			}
		}

		// event -> view (via subscribes)
		for _, evt := range s.Events {
			ex, ey, ew, eh, eok := elemPos(i, evt.Name)
			if !eok {
				continue
			}
			ecx, ecy := ex+ew/2, ey+eh/2
			for _, view := range s.Views {
				for _, sub := range view.Subscribes {
					if sub == evt.Name {
						vx, vy, vw, vh, vok := elemPos(i, view.Name)
						if vok {
							vcx, vcy := vx+vw/2, vy+vh/2
							b.WriteString(svgArrowPath(ecx, ecy, vcx, vcy))
						}
					}
				}
			}
		}

		// event -> automation -> command
		for _, auto := range s.Automations {
			ex, ey, ew, eh, eok := elemPos(i, auto.TriggerEvent)
			ax, ay, aw, ah, aok := elemPos(i, auto.Name)
			cx, cy, cw, ch, cok := elemPos(i, auto.Command)

			if eok && aok {
				ecx, ecy := ex+ew/2, ey+eh/2
				acx, acy := ax+aw/2, ay+ah/2
				b.WriteString(svgArrowPath(ecx, ecy, acx, acy))
			}
			if aok && cok {
				acx, acy := ax+aw/2, ay+ah/2
				ccx, ccy := cx+cw/2, cy+ch/2
				b.WriteString(svgArrowPath(acx, acy, ccx, ccy))
			}
		}

		// Translation: command -> external system -> event
		for _, tr := range s.Translations {
			extX, extY, extW, extH, extOK := elemPos(i, tr.ExternalSystem)
			if !extOK {
				continue
			}
			extCx, extCy := extX+extW/2, extY+extH/2

			if tr.Command != "" {
				cx, cy, cw, ch, cok := elemPos(i, tr.Command)
				if cok {
					ccx, ccy := cx+cw/2, cy+ch/2
					b.WriteString(svgArrowPath(ccx, ccy, extCx, extCy))
				}
			}
			if tr.Event != nil && tr.Event.Name != "" {
				ex, ey, ew, eh, eok := elemPos(i, tr.Event.Name)
				if eok {
					ecx, ecy := ex+ew/2, ey+eh/2
					b.WriteString(svgArrowPath(extCx, extCy, ecx, ecy))
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

func svgRect(x, y, w, h int, fill, stroke string, rx int) string {
	return fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="%s" rx="%d"/>`+"\n",
		x, y, w, h, fill, stroke, rx)
}

func svgRoundedRect(x, y, w, h int, fill, stroke string, rx int) string {
	return svgRect(x, y, w, h, fill, stroke, rx)
}

func svgDashedRoundedRect(x, y, w, h int, fill, stroke string, rx int) string {
	return fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="%s" rx="%d" stroke-dasharray="5,3"/>`+"\n",
		x, y, w, h, fill, stroke, rx)
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
