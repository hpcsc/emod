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

	diagramW := xPos + marginX + 120
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

	// Swimlanes
	laneW := diagramW - 2*marginX
	b.WriteString(svgRect(marginX, triggerLaneY, laneW, laneHeight, "#ffffff", "#000000", 5))
	b.WriteString(svgLaneLabel(marginX+10, triggerLaneY+20, "UI / Triggers"))
	b.WriteString(svgRect(marginX, cmdViewLaneY, laneW, laneHeight, "#ffffff", "#000000", 5))
	b.WriteString(svgLaneLabel(marginX+10, cmdViewLaneY+20, "Commands / Views"))
	b.WriteString(svgRect(marginX, eventLaneY, laneW, laneHeight, "#ffffff", "#000000", 5))
	b.WriteString(svgLaneLabel(marginX+10, eventLaneY+20, "Events"))
	b.WriteString(svgRect(marginX, extLaneY, laneW, laneHeight, "#ffffff", "#000000", 5))
	b.WriteString(svgLaneLabel(marginX+10, extLaneY+20, "External Systems"))

	for _, cb := range ctxBounds {
		b.WriteString(svgRect(cb.x, marginY-30, cb.w-20, 22, fillExternal, strokeExternal, 0))
		b.WriteString(svgText(cb.x+(cb.w-20)/2, marginY-19, cb.name, 12, strokeExternal))
	}

	// Center Y within each lane's content area (below 30px label area)
	triggerCenterY := triggerLaneY + 30 + (laneHeight-30-boxHeight)/2
	midCenterY := cmdViewLaneY + 30 + (laneHeight-30-boxHeight)/2
	eventCenterY := eventLaneY + 30 + (laneHeight-30-boxHeight)/2
	extCenterY := extLaneY + 30 + (laneHeight-30-boxHeight)/2

	type svgElem struct {
		sliceIdx int
		name     string
		x, y, w, h int
	}
	var elems []svgElem

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
			b.WriteString(svgRoundedRect(x, eventCenterY, itemW, boxHeight, fillEvent, strokeEvent, 5))
			b.WriteString(svgMultilineText(x+itemW/2, eventCenterY+boxHeight/2, label, 11, strokeEvent))
			elems = append(elems, svgElem{
				sliceIdx: i, name: evt.Name,
				x: x, y: eventCenterY, w: itemW, h: boxHeight,
			})
		}
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
				ei++
				b.WriteString(svgRoundedRect(x, eventCenterY, itemW, boxHeight, fillEvent, strokeEvent, 5))
				b.WriteString(svgText(x+itemW/2, eventCenterY+boxHeight/2, tr.Event.Name, 11, strokeEvent))
				elems = append(elems, svgElem{
					sliceIdx: i, name: tr.Event.Name,
					x: x, y: eventCenterY, w: itemW, h: boxHeight,
				})
			}
		}

		// --- Automations (compact boxes with gear indicator) ---
		for ai, auto := range s.Automations {
			autoW := boxWidth
			autoH := boxHeight * 3 / 4
			autoPadX := 10
			autoPadY := 15 + ai*(autoH+5)
			x := sliceX + autoPadX
			y := triggerLaneY + laneHeight - autoH - autoPadY
			label := fmt.Sprintf("\u2699 %s", auto.Name)
			b.WriteString(svgRoundedRect(x, y, autoW-boxWidth/8, autoH, fillReactor, strokeReactor, 3))
			b.WriteString(svgMultilineText(x+(autoW-boxWidth/8)/2, y+autoH/2, label, 10, strokeReactor))
			elems = append(elems, svgElem{
				sliceIdx: i, name: auto.Name,
				x: x, y: y, w: autoW - boxWidth/8, h: autoH,
			})
		}

		// --- Translation reactors (in UI/Triggers lane, below automations) ---
		for ti, tr := range s.Translations {
			reactorW := boxWidth
			reactorH := boxHeight * 3 / 4
			padX := 10
			padY := 15 + (len(s.Automations)+ti)*(reactorH+5)
			x := sliceX + padX
			y := triggerLaneY + laneHeight - reactorH - padY
			label := fmt.Sprintf("\u2699 %s", tr.Name)
			b.WriteString(svgRoundedRect(x, y, reactorW-boxWidth/8, reactorH, fillReactor, strokeReactor, 3))
			b.WriteString(svgMultilineText(x+(reactorW-boxWidth/8)/2, y+reactorH/2, label, 10, strokeReactor))
			elems = append(elems, svgElem{
				sliceIdx: i, name: tr.Name,
				x: x, y: y, w: reactorW - boxWidth/8, h: reactorH,
			})
		}

		// --- External system boxes (Translations) ---
		for ti, tr := range s.Translations {
			extW := 100
			extH := 45
			extX := sliceX + (sliceWidth-extW)/2
			extY := extCenterY - extH/2
			if ti > 0 {
				extY += ti * (extH + 8)
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

	// Global element lookup across all slices (needed for cross-slice references)
	type elemPosInfo struct{ x, y, w, h int }
	nameToPos := make(map[string]elemPosInfo)
	for _, e := range elems {
		nameToPos[e.name] = elemPosInfo{x: e.x, y: e.y, w: e.w, h: e.h}
	}

	for _, entry := range entries {
		s := entry.slice

		// trigger -> command
		if s.Trigger != nil {
			tPos, tok := nameToPos[s.Trigger.Name]
			if tok {
				tcx, tcy := tPos.x+tPos.w/2, tPos.y+tPos.h/2
				for _, cmd := range s.Commands {
					cPos, cok := nameToPos[cmd.Name]
					if cok {
						ccx, ccy := cPos.x+cPos.w/2, cPos.y+cPos.h/2
						b.WriteString(svgArrowPath(tcx, tcy, ccx, ccy))
					}
				}
			}
		}

		// command -> event (via Flow entries)
		for _, flow := range s.Flows {
			cPos, cok := nameToPos[flow.CommandName]
			ePos, eok := nameToPos[flow.EventName]
			if cok && eok {
				ccx, ccy := cPos.x+cPos.w/2, cPos.y+cPos.h/2
				ecx, ecy := ePos.x+ePos.w/2, ePos.y+ePos.h/2
				b.WriteString(svgArrowPath(ccx, ccy, ecx, ecy))
			}
		}

		// event -> view (via subscribes) — cross-slice lookup
		for _, view := range s.Views {
			vPos, vok := nameToPos[view.Name]
			if !vok {
				continue
			}
			vcx, vcy := vPos.x+vPos.w/2, vPos.y+vPos.h/2
			for _, sub := range view.Subscribes {
				ePos, eok := nameToPos[sub]
				if eok {
					ecx, ecy := ePos.x+ePos.w/2, ePos.y+ePos.h/2
					b.WriteString(svgArrowPath(ecx, ecy, vcx, vcy))
				}
			}
		}

		// event -> automation -> command — cross-slice lookup
		for _, auto := range s.Automations {
			ePos, eok := nameToPos[auto.TriggerEvent]
			aPos, aok := nameToPos[auto.Name]
			cPos, cok := nameToPos[auto.Command]
			if eok && aok {
				ecx, ecy := ePos.x+ePos.w/2, ePos.y+ePos.h/2
				acx, acy := aPos.x+aPos.w/2, aPos.y+aPos.h/2
				b.WriteString(svgArrowPath(ecx, ecy, acx, acy))
			}
			if aok && cok {
				acx, acy := aPos.x+aPos.w/2, aPos.y+aPos.h/2
				ccx, ccy := cPos.x+cPos.w/2, cPos.y+cPos.h/2
				b.WriteString(svgArrowPath(acx, acy, ccx, ccy))
			}
		}

		// Translation: ext sys -> reactor -> command/event
		for _, tr := range s.Translations {
			extPos, extOK := nameToPos[tr.ExternalSystem]
			reactorPos, reactorOK := nameToPos[tr.Name]
			if !extOK || !reactorOK {
				continue
			}
			extCx, extCy := extPos.x+extPos.w/2, extPos.y+extPos.h/2
			reactorCx, reactorCy := reactorPos.x+reactorPos.w/2, reactorPos.y+reactorPos.h/2

			// reads: view -> external system
			if tr.Reads != "" {
				vPos, vok := nameToPos[tr.Reads]
				if vok {
					vcx, vcy := vPos.x+vPos.w/2, vPos.y+vPos.h/2
					b.WriteString(svgArrowPath(vcx, vcy, extCx, extCy))
				}
			}
			// external system -> reactor
			b.WriteString(svgArrowPath(extCx, extCy, reactorCx, reactorCy))
			// reactor -> command
			if tr.Command != "" {
				cPos, cok := nameToPos[tr.Command]
				if cok {
					ccx, ccy := cPos.x+cPos.w/2, cPos.y+cPos.h/2
					b.WriteString(svgArrowPath(reactorCx, reactorCy, ccx, ccy))
				}
			}
			// command -> event (translation implies command emits event)
			if tr.Command != "" && tr.Event != nil && tr.Event.Name != "" {
				cPos, cok := nameToPos[tr.Command]
				ePos, eok := nameToPos[tr.Event.Name]
				if cok && eok {
					ccx, ccy := cPos.x+cPos.w/2, cPos.y+cPos.h/2
					ecx, ecy := ePos.x+ePos.w/2, ePos.y+ePos.h/2
					b.WriteString(svgArrowPath(ccx, ccy, ecx, ecy))
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
