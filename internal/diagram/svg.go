// Package diagram renders AST models as diagrams.
package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

// ExportSVG generates a self-contained SVG diagram from a parsed AST model.
func ExportSVG(model *ast.Model, _ Style, opts ...Option) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	entries := collectSlices(model)
	copies := NewElementCopies(model)

	columns := sliceColumns(entries, copies)
	ctxBounds := contextBounds(entries, columns)

	triggerLaneY := marginY
	cmdViewLaneY := triggerLaneY + laneHeight + laneGap
	eventLaneY := cmdViewLaneY + laneHeight + laneGap
	extLaneY := eventLaneY + laneHeight + laneGap
	specBandY := specBandTop(entries, extLaneY)

	var cards []specCard
	if resolveOptions(opts).specs {
		cards = specCards(entries, columns, specBandY, "\n")
	}
	bandHeight := specBandHeight(cards)

	diagramW := layoutWidth(columns) + marginX + 120
	diagramH := 2*marginY + 4*laneHeight + 3*laneGap
	if bandHeight > 0 {
		diagramH = specBandY + bandHeight + marginY
	}

	var b strings.Builder
	b.WriteString(svgHeader(diagramW, diagramH))
	b.WriteString(svgDefs())
	if drawsAutomation(entries) {
		b.WriteString(svgGearDefs())
	}

	if len(entries) == 0 {
		b.WriteString("</svg>\n")
		return []byte(b.String()), nil
	}

	laneW := diagramW - 2*marginX
	b.WriteString(svgLane(triggerLaneY, laneW, laneHeight, "Wireframes"))
	b.WriteString(svgLane(cmdViewLaneY, laneW, laneHeight, "Commands / Views"))
	b.WriteString(svgLane(eventLaneY, laneW, laneHeight, "Events"))
	b.WriteString(svgLane(extLaneY, laneW, laneHeight, "External Systems"))

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

	// Badges are filed per slice, in declaration order, rather than by the name
	// they carry: two slices may reject the same invariant, and nameToBox keeps
	// only the last box drawn for a name, so both dashed arrows would end at
	// whichever slice was drawn last.
	badges := make([][]svgBox, len(entries))

	// A copy is filed per slice, by the element it copies: nameToBox keeps the
	// original, which every slice but this one draws its arrows from.
	copyBoxes := make([]map[ElementCopy]svgBox, len(entries))
	sourceBox := func(i int, edge Edge) (svgBox, bool) {
		if copied, found := copies.At(entries[i].slice, edge); found {
			return copyBoxes[i][copied], true
		}
		box, drawn := nameToBox[edge.From]
		return box, drawn
	}
	targetBox := func(i int, edge Edge) (svgBox, bool) {
		if copied, found := copies.At(entries[i].slice, edge); found {
			return copyBoxes[i][copied], true
		}
		box, drawn := nameToBox[edge.To]
		return box, drawn
	}

	// Place elements per slice
	for i, entry := range entries {
		s := entry.slice
		column := columns[i]
		sliceX := column.x

		// --- Trigger (top lane) ---
		if s.Trigger != nil {
			x := sliceX + (column.w-boxWidth)/2
			y := triggerRowY
			label := s.Trigger.Name
			if s.Trigger.Actor != "" {
				label = fmt.Sprintf("%s (%s)", s.Trigger.Name, s.Trigger.Actor)
			}
			b.WriteString(svgTriggerRect(x, y, boxWidth, boxHeight, fillTrigger, strokeTrigger, 5, s.Trigger.Description))
			b.WriteString(svgText(x+boxWidth/2, y+boxHeight/2, label, 12, strokeTrigger))
			nameToBox[s.Trigger.Name] = svgBox{x: x, y: y, w: boxWidth, h: boxHeight}
		}

		// --- Commands (middle lane) ---
		copiedEvents, copiedMiddle := copiesByRow(copies.In(s))
		copyBoxes[i] = make(map[ElementCopy]svgBox)
		totalMid := middleRowItems(s) + len(copiedMiddle)
		usableW := column.w - 20
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

		// --- Copies of views and commands this slice joins from slices far away (middle lane) ---
		for mi, c := range copiedMiddle {
			itemW, x := itemLayout(usableW, totalMid, middleRowItems(s)+mi, sliceX)
			box := svgBox{x: x, y: cmdViewRowY, w: itemW, h: boxHeight}
			b.WriteString(svgCopy(c, box))
			copyBoxes[i][c] = box
		}

		// --- Events (bottom lane, including translation events, rejection badges and copies) ---
		totalEvts := eventRowItems(s) + len(copiedEvents)
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

		// --- Rejection badges (event row) ---
		for _, rejection := range s.Rejections {
			itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
			ei++
			b.WriteString(svgDashedRoundedRect(x, eventRowY, itemW, boxHeight, fillRejection, strokeRejection, 5,
				entry.invariantStatement(rejection.InvariantName)))
			b.WriteString(svgText(x+itemW/2, eventRowY+boxHeight/2, rejection.InvariantName, 11, strokeRejection))
			badges[i] = append(badges[i], svgBox{x: x, y: eventRowY, w: itemW, h: boxHeight})
		}

		// --- Copies of events this slice joins from slices far away (event row) ---
		for _, c := range copiedEvents {
			itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
			ei++
			box := svgBox{x: x, y: eventRowY, w: itemW, h: boxHeight}
			b.WriteString(svgCopy(c, box))
			copyBoxes[i][c] = box
		}

		// --- Automations and translation reactors (middle lane) ---
		for _, reactor := range reactorBoxes(s, cmdViewLaneY, column, "\n") {
			if reactor.isAutomation {
				b.WriteString(svgGear(reactor.x, reactor.y, reactor.w, reactor.description))
				b.WriteString(svgMultilineText(reactor.x+reactor.w/2, reactor.labelCentreY(),
					reactor.label, reactorFontSize, fillAutomation))
				nameToBox[reactor.name] = reactor.gearAndLabelBox()
			} else {
				b.WriteString(svgRoundedRect(reactor.x, reactor.y, reactor.w, reactor.h, fillTranslation, strokeTranslation, 3, reactor.description))
				b.WriteString(svgMultilineText(reactor.x+reactor.w/2, reactor.y+reactor.h/2, reactor.label, reactorFontSize, strokeTranslation))
				nameToBox[reactor.name] = svgBox{x: reactor.x, y: reactor.y, w: reactor.w, h: reactor.h}
			}
		}

		// --- External system boxes (Translations) ---
		for ti, tr := range s.Translations {
			extW := externalBoxWidth
			extH := externalBoxHeight
			extX := sliceX + (column.w-extW)/2
			extY := extRowY - extH/2
			if ti > 0 {
				extY += ti * (extH + externalBoxGap)
			}
			b.WriteString(svgDashedRoundedRect(extX, extY, extW, extH, fillExternal, strokeExternal, 5, tr.Description))
			b.WriteString(svgText(extX+extW/2, extY+extH/2, tr.ExternalSystem, 11, strokeExternal))
			nameToBox[tr.ExternalSystem] = svgBox{x: extX, y: extY, w: extW, h: extH}
		}
	}

	// --- Connections ---

	readsArrow := func(i int, edge Edge, reader svgBox) string {
		view, declared := sourceBox(i, edge)
		if !declared {
			return ""
		}

		return svgArrow(svgRoute(view, reader))
	}

	// A translation's reads arrow is drawn into the external system box
	// fronting its reactor, not the reactor itself.
	reactorExternal := make(map[string]string)
	for _, entry := range entries {
		for _, tr := range entry.slice.Translations {
			reactorExternal[tr.Name] = tr.ExternalSystem
		}
	}

	for i, entry := range entries {
		rejected := 0
		for _, edge := range SliceEdges(entry.slice) {
			switch edge.Kind {
			case EdgeTriggerReads, EdgeAutomationReads:
				if reader, drawn := nameToBox[edge.To]; drawn {
					b.WriteString(readsArrow(i, edge, reader))
				}

			case EdgeTranslationReads:
				if reader, drawn := nameToBox[reactorExternal[edge.To]]; drawn {
					b.WriteString(readsArrow(i, edge, reader))
				}

			case EdgeAutomationCommand, EdgeTranslationCommand:
				from, fromDrawn := nameToBox[edge.From]
				to, toDrawn := targetBox(i, edge)
				if fromDrawn && toDrawn {
					b.WriteString(svgArrow(svgRoute(from, to)))
				}

			case EdgeTranslationFlow:
				from, fromDrawn := sourceBox(i, edge)
				to, toDrawn := nameToBox[edge.To]
				if fromDrawn && toDrawn {
					b.WriteString(svgArrow(svgRoute(from, to)))
				}

			case EdgeAutomationTrigger:
				from, fromDrawn := sourceBox(i, edge)
				to, toDrawn := nameToBox[edge.To]
				if fromDrawn && toDrawn {
					route := svgRoute(from, to)
					b.WriteString(svgArrow(route))
					b.WriteString(svgArrowLabel(route, edge.Label))
				}

			case EdgeRejection:
				// SliceEdges emits rejection edges in declaration order, so the
				// nth one this slice states ends at the nth badge it drew.
				from, fromDrawn := nameToBox[edge.From]
				if fromDrawn && rejected < len(badges[i]) {
					b.WriteString(svgDashedArrow(svgRoute(from, badges[i][rejected])))
				}
				rejected++

			case EdgeSubscription:
				from, fromDrawn := sourceBox(i, edge)
				to, toDrawn := nameToBox[edge.To]
				if fromDrawn && toDrawn {
					b.WriteString(svgArrow(svgRoute(from, to)))
				}

			default:
				from, fromDrawn := nameToBox[edge.From]
				to, toDrawn := nameToBox[edge.To]
				if fromDrawn && toDrawn {
					b.WriteString(svgArrow(svgRoute(from, to)))
				}
			}
		}
	}

	// --- Spec cards (a band of their own, below the lowest lane) ---
	// Written after everything else so a card cannot capture the label of a box
	// drawn before it: a reader binds each text to the last rect it saw.
	if len(cards) > 0 {
		b.WriteString(svgLane(specBandY, laneW, bandHeight, "Specs"))
		for _, card := range cards {
			b.WriteString(svgRoundedRect(card.x, card.y, card.w, card.h, fillSpecCard, strokeSpecCard, 5, ""))
			b.WriteString(svgMultilineText(card.x+card.w/2, card.y+card.h/2, card.label, specCardFontSize, strokeSpecCard))
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
<path d="M 0 0 L 10 5 L 0 10 z" fill="` + arrowStroke + `"/>
</marker>
</defs>
`
}

func drawsAutomation(entries []sliceEntry) bool {
	for _, entry := range entries {
		if len(entry.slice.Automations) > 0 {
			return true
		}
	}
	return false
}

func svgGearDefs() string {
	return fmt.Sprintf(`<defs>
<symbol id="gear" viewBox="%d %d %d %d">
<path d="%s"/>
</symbol>
</defs>
`, -gearExtent, -gearExtent, 2*gearExtent, 2*gearExtent, gearPathData)
}

func svgGear(x, y, size int, description string) string {
	return svgElement("use", fmt.Sprintf(`href="#gear" x="%d" y="%d" width="%d" height="%d" fill="%s"`,
		x, y, size, size, fillAutomation), description)
}

func svgRect(x, y, w, h int, fill, stroke string, rx int, description string) string {
	return svgRectElement(svgRectAttributes(x, y, w, h, fill, stroke, rx), description)
}

func svgRoundedRect(x, y, w, h int, fill, stroke string, rx int, description string) string {
	return svgRect(x, y, w, h, fill, stroke, rx, description)
}

// svgTriggerRect draws a trigger box as a screen: a small header bar inside the
// top edge, then the main rounded rectangle. The header is drawn first so the
// text label that follows attaches to the main rect, not to the framing.
func svgTriggerRect(x, y, w, h int, fill, stroke string, rx int, description string) string {
	const headerMargin = 8
	const headerTop = 6
	const headerHeight = 6

	header := svgRectElement(svgRectAttributes(x+headerMargin, y+headerTop, w-2*headerMargin, headerHeight, stroke, stroke, 0), "")
	box := svgRectElement(svgRectAttributes(x, y, w, h, fill, stroke, rx), description)
	return header + box
}

// svgCopy draws a copy dashed, filled white and outlined in the stroke of what it
// copies, with the copied element's name and the slice that declares it on it.
func svgCopy(c ElementCopy, box svgBox) string {
	stroke := strokeEvent
	switch c.Kind {
	case CopyOfView:
		stroke = strokeView
	case CopyOfCommand:
		stroke = strokeCommand
	}

	return svgDashedRoundedRect(box.x, box.y, box.w, box.h, fillCopy, stroke, 5, "") +
		svgMultilineText(box.x+box.w/2, box.y+box.h/2, strings.Join(c.labelLines(), "\n"), 11, stroke)
}

func svgDashedRoundedRect(x, y, w, h int, fill, stroke string, rx int, description string) string {
	return svgRectElement(svgRectAttributes(x, y, w, h, fill, stroke, rx)+` stroke-dasharray="5,3"`, description)
}

func svgRectAttributes(x, y, w, h int, fill, stroke string, rx int) string {
	return fmt.Sprintf(`x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="%s" rx="%d"`,
		x, y, w, h, fill, stroke, rx)
}

func svgRectElement(attributes, description string) string {
	return svgElement("rect", attributes, description)
}

// A browser shows a description as a tooltip only when the <title> holding it is
// nested inside the shape, which a self-closing element has no room for. A shape
// with nothing to say stays self-closing: opening those too would rewrite the
// diagram of every model that carries no prose.
func svgElement(name, attributes, description string) string {
	if description == "" {
		return fmt.Sprintf("<%s %s/>\n", name, attributes)
	}

	return fmt.Sprintf("<%s %s>\n<title>%s</title>\n</%s>\n", name, attributes, escapeXML(description), name)
}

func svgLane(y, w, h int, label string) string {
	return svgRect(marginX, y, w, h, "#ffffff", "#000000", 5, "") +
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

// lineHeight is how far apart svgMultilineText writes the lines of a text.
func lineHeight(fontSize int) int {
	return fontSize + 4
}

func svgMultilineText(x, y int, text string, fontSize int, color string) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= 1 {
		return svgText(x, y, text, fontSize, color)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="%d" fill="%s">`+"\n",
		x, y, fontSize, color))

	step := lineHeight(fontSize)
	totalH := (len(lines) - 1) * step

	for i, line := range lines {
		dy := step
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

// point is a place in the picture, in pixels.
type point struct {
	x, y int
}

// route is the way an arrow runs from one box to another: out of the first box
// at start, across along its leg at legY, and into the second box at end.
type route struct {
	start, end point
	legY       int
}

// svgRoute leaves from at the top or the bottom, whichever faces to, and enters
// to at the side that faces from, with the leg halfway between the two.
func svgRoute(from, to svgBox) route {
	start, end := point{from.x + from.w/2, from.y}, point{to.x + to.w/2, to.y + to.h}
	if from.y+from.h <= to.y {
		start, end = point{from.x + from.w/2, from.y + from.h}, point{to.x + to.w/2, to.y}
	}

	return route{start: start, end: end, legY: (start.y + end.y) / 2}
}

// points are the points the arrow is drawn through, in order.
func (r route) points() []point {
	if r.start.x == r.end.x {
		return []point{r.start, r.end}
	}

	return []point{r.start, {r.start.x, r.legY}, {r.end.x, r.legY}, r.end}
}

// legMiddle is the middle of the leg across, or of the route itself when it runs
// straight up or down.
func (r route) legMiddle() point {
	return point{(r.start.x + r.end.x) / 2, r.legY}
}

// svgArrowLabel paints an arrow's text at the middle of its leg, rather than at
// either bend, and nothing at all for an arrow carrying none, so the picture of
// a model whose arrows say nothing is byte-identical to what it was.
// The class is what separates this text from a box's own: a reader binding
// every <text> to the last rect it saw would otherwise relabel that box.
func svgArrowLabel(r route, label string) string {
	if label == "" {
		return ""
	}

	at := r.legMiddle()

	return fmt.Sprintf(`<text class="edge-label" x="%d" y="%d" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="%d" fill="%s">%s</text>`+"\n",
		at.x, at.y, edgeLabelFontSize, arrowStroke, escapeXML(label))
}

func svgArrow(r route) string {
	return svgArrowPath(r, "")
}

// svgDashedArrow draws the same route as svgArrow, dashed.
func svgDashedArrow(r route) string {
	return svgArrowPath(r, ` stroke-dasharray="6,4"`)
}

const (
	// arrowStroke paints every arrow, its arrowhead and any text painted on
	// one, so a label and the line it names cannot drift apart.
	arrowStroke = "#666666"
	// edgeLabelFontSize matches the size a reactor box draws its own label at,
	// so a duration on an arrow reads as part of the same picture rather than
	// as an annotation over it.
	edgeLabelFontSize = 10
)

func svgArrowPath(r route, dash string) string {
	var points []string
	for _, p := range r.points() {
		points = append(points, fmt.Sprintf("%d,%d", p.x, p.y))
	}

	return fmt.Sprintf(`<path d="M %s" fill="none" stroke="`+arrowStroke+`" stroke-width="1.5"%s marker-end="url(#arrow)"/>`+"\n",
		strings.Join(points, " L "), dash)
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
