// Package diagram renders AST models as diagrams.
package diagram

import (
	"fmt"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
)

const lanesPastLastSlice = 120

// ExportSVG generates a self-contained SVG diagram from a parsed AST model.
func ExportSVG(model *ast.Model, _ Style, opts ...Option) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	entries := collectSlices(model)
	copies := NewElementCopies(model)

	columns := sliceColumns(entries, copies)
	ctxBounds := contextBounds(entries, columns)

	// A gap grows to hold the tracks of the legs in it, which moves the lanes and
	// the boxes under it down, so the picture is placed and routed again until
	// every gap holds its legs.
	var lanes lanePositions
	var boxes string
	var arrows []boxArrow
	var routes []route
	for {
		var obstacles []svgBox
		boxes, arrows, obstacles = placeBoxes(entries, columns, copies, lanes, screenDescriptions(model))
		routes = routesFor(arrows, obstacles, columns, lanes)
		grown := lanes.withTracksOf(routes)
		if grown == lanes {
			break
		}
		lanes = grown
	}

	extLaneY := lanes.top(len(laneNames) - 1)
	specBandY := specBandTop(extLaneY)

	var cards []specCard
	if resolveOptions(opts).specs {
		cards = specCards(entries, columns, specBandY, "\n")
	}
	bandHeight := specBandHeight(cards)

	diagramW := layoutWidth(columns) + marginX + lanesPastLastSlice
	diagramH := extLaneY + laneHeight + marginY
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
	for lane, name := range laneNames {
		b.WriteString(svgLane(lanes.top(lane), laneW, laneHeight, name))
	}

	for _, cb := range ctxBounds {
		b.WriteString(svgRect(cb.x, marginY-30, cb.w-20, 22, fillExternal, strokeExternal, 0, cb.description))
		b.WriteString(svgText(cb.x+(cb.w-20)/2, marginY-19, cb.name, 12, strokeExternal))
	}

	b.WriteString(boxes)
	for i, arrow := range arrows {
		if arrow.dashed {
			b.WriteString(svgDashedArrow(routes[i]))
		} else {
			b.WriteString(svgArrow(routes[i]))
		}
		if arrow.label != "" {
			b.WriteString(svgArrowLabel(labelAt(routes, i, arrow.label), arrow.label))
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

// placeBoxes writes the box of every element of every slice, with the lanes
// where lanes says. It returns the arrows between those boxes, in the order the
// picture draws them, and the boxes an arrow must keep clear of when it does not
// join them.
func placeBoxes(entries []sliceEntry, columns []sliceColumn, copies ElementCopies, lanes lanePositions, screens map[string]string) (string, []boxArrow, []svgBox) {
	var b strings.Builder
	var obstacles []svgBox
	placed := func(box svgBox) svgBox {
		obstacles = append(obstacles, box)
		return box
	}

	triggerRowY := laneRowY(lanes.top(wireframesLane))
	cmdViewLaneY := lanes.top(commandsAndViewsLane)
	cmdViewRowY := laneRowY(cmdViewLaneY)
	eventRowY := laneRowY(lanes.top(eventsLane))
	extRowY := laneRowY(lanes.top(externalSystemsLane))

	// Where the box drawn for each name ended up. One map across every slice, so
	// a connection can reach a box another slice drew.
	nameToBox := make(map[string]svgBox)
	triggerBoxes := make(map[*ast.Slice]svgBox)
	appearanceBoxes := make([]map[string]svgBox, len(entries))

	// Badges are filed per slice, in declaration order, rather than by the name
	// they carry: two slices may reject the same invariant, and nameToBox keeps
	// only the last box drawn for a name, so both dashed arrows would end at
	// whichever slice was drawn last.
	badges := make([][]svgBox, len(entries))

	// The box of the external system each translation reads through, by the
	// name of the translation. A translation can share its name with its
	// external system, so these boxes stay out of nameToBox.
	externalBoxes := make(map[string]svgBox)

	// A copy is filed per slice, by the element it copies, and every element by
	// the slice that declares it, so that an arrow joins the copy or the element
	// that ElementCopies names: nameToBox keeps the last box drawn under a name,
	// of any kind.
	copyBoxes := make([]map[DeclaredElement]svgBox, len(entries))
	declaredBoxes := make(map[DeclaredElement]svgBox)
	declare := func(kind ElementKind, name string, s *ast.Slice, box svgBox) svgBox {
		key := DeclaredElement{Kind: kind, Name: name, Slice: s}
		if _, seen := declaredBoxes[key]; !seen {
			declaredBoxes[key] = box
		}
		return box
	}
	joinedBox := func(i int, edge Edge) (svgBox, bool) {
		element, copied, _ := copies.Joined(entries[i].slice, edge)
		if copied {
			return copyBoxes[i][element], true
		}
		box, drawn := declaredBoxes[element]
		return box, drawn
	}

	// Place elements per slice
	for i, entry := range entries {
		s := entry.slice
		column := columns[i]
		sliceX := column.x
		usableW := column.w - 20

		// --- Wireframes (top lane): the trigger and each screen the slice shows ---
		wireframes := len(s.Appearances)
		if s.Trigger != nil {
			wireframes++
		}
		wireframeAt := func(index int) (int, int) {
			if wireframes == 1 {
				return boxWidth, sliceX + (column.w-boxWidth)/2
			}
			return itemLayout(usableW, wireframes, index, sliceX)
		}
		next := 0
		if s.Trigger != nil {
			w, x := wireframeAt(next)
			next++
			b.WriteString(svgTriggerRect(x, triggerRowY, w, boxHeight, fillTrigger, strokeTrigger, 5, s.Trigger.Description))
			b.WriteString(svgText(x+w/2, triggerRowY+boxHeight/2, s.Trigger.Name, 12, strokeTrigger))
			triggerBoxes[s] = placed(svgBox{x: x, y: triggerRowY, w: w, h: boxHeight, lane: wireframesLane})
		}
		appearanceBoxes[i] = make(map[string]svgBox, len(s.Appearances))
		for _, appearance := range s.Appearances {
			w, x := wireframeAt(next)
			next++
			b.WriteString(svgTriggerRect(x, triggerRowY, w, boxHeight, fillTrigger, strokeTrigger, 5, screens[appearance.Screen]))
			b.WriteString(svgText(x+w/2, triggerRowY+boxHeight/2, appearance.Screen, 12, strokeTrigger))
			appearanceBoxes[i][appearance.Screen] = placed(svgBox{x: x, y: triggerRowY, w: w, h: boxHeight, lane: wireframesLane})
		}

		// --- Commands (middle lane) ---
		copiedEvents, copiedMiddle := copiesByRow(copies.In(s))
		copyBoxes[i] = make(map[DeclaredElement]svgBox)
		totalMid := middleRowItems(s) + len(copiedMiddle)
		for ci, cmd := range s.Commands {
			itemW, x := itemLayout(usableW, totalMid, ci, sliceX)
			b.WriteString(svgRoundedRect(x, cmdViewRowY, itemW, boxHeight, fillCommand, strokeCommand, 5, cmd.Description))
			b.WriteString(svgText(x+itemW/2, cmdViewRowY+boxHeight/2, cmd.Name, 12, strokeCommand))
			nameToBox[cmd.Name] = declare(ElementCommand, cmd.Name, s, placed(svgBox{x: x, y: cmdViewRowY, w: itemW, h: boxHeight, lane: commandsAndViewsLane}))
		}

		// --- Views (middle lane) ---
		for vi, view := range s.Views {
			idx := len(s.Commands) + vi
			itemW, x := itemLayout(usableW, totalMid, idx, sliceX)
			b.WriteString(svgRoundedRect(x, cmdViewRowY, itemW, boxHeight, fillView, strokeView, 5, view.Description))
			b.WriteString(svgText(x+itemW/2, cmdViewRowY+boxHeight/2, view.Name, 12, strokeView))
			nameToBox[view.Name] = declare(ElementView, view.Name, s, placed(svgBox{x: x, y: cmdViewRowY, w: itemW, h: boxHeight, lane: commandsAndViewsLane}))
		}

		// --- Copies of views and commands this slice joins from slices far away (middle lane) ---
		for mi, c := range copiedMiddle {
			itemW, x := itemLayout(usableW, totalMid, middleRowItems(s)+mi, sliceX)
			box := placed(svgBox{x: x, y: cmdViewRowY, w: itemW, h: boxHeight, lane: commandsAndViewsLane})
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
			nameToBox[evt.Name] = declare(ElementEvent, evt.Name, s, placed(svgBox{x: x, y: eventRowY, w: itemW, h: boxHeight, lane: eventsLane}))
		}
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
				ei++
				b.WriteString(svgRoundedRect(x, eventRowY, itemW, boxHeight, fillEvent, strokeEvent, 5, tr.Event.Description))
				b.WriteString(svgText(x+itemW/2, eventRowY+boxHeight/2, tr.Event.Name, 11, strokeEvent))
				nameToBox[tr.Event.Name] = declare(ElementEvent, tr.Event.Name, s, placed(svgBox{x: x, y: eventRowY, w: itemW, h: boxHeight, lane: eventsLane}))
			}
		}

		// --- Rejection badges (event row) ---
		for _, rejection := range s.Rejections {
			itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
			ei++
			b.WriteString(svgDashedRoundedRect(x, eventRowY, itemW, boxHeight, fillRejection, strokeRejection, 5,
				entry.invariantStatement(rejection.InvariantName)))
			b.WriteString(svgText(x+itemW/2, eventRowY+boxHeight/2, rejection.InvariantName, 11, strokeRejection))
			badges[i] = append(badges[i], placed(svgBox{x: x, y: eventRowY, w: itemW, h: boxHeight, lane: eventsLane}))
		}

		// --- Copies of events this slice joins from slices far away (event row) ---
		for _, c := range copiedEvents {
			itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
			ei++
			box := placed(svgBox{x: x, y: eventRowY, w: itemW, h: boxHeight, lane: eventsLane})
			b.WriteString(svgCopy(c, box))
			copyBoxes[i][c] = box
		}

		// --- Automations and translation reactors (middle lane) ---
		for _, reactor := range reactorBoxes(s, cmdViewLaneY, column, "\n") {
			if reactor.isAutomation {
				b.WriteString(svgGear(reactor.x, reactor.y, reactor.w, reactor.description))
				b.WriteString(svgMultilineText(reactor.x+reactor.w/2, reactor.labelCentreY(),
					reactor.label, reactorFontSize, fillAutomation))
				nameToBox[reactor.name] = reactor.arrowEndBox(commandsAndViewsLane)
				obstacles = append(obstacles, reactor.obstacleBox(commandsAndViewsLane))
			} else {
				b.WriteString(svgRoundedRect(reactor.x, reactor.y, reactor.w, reactor.h, fillTranslation, strokeTranslation, 3, reactor.description))
				b.WriteString(svgMultilineText(reactor.x+reactor.w/2, reactor.y+reactor.h/2, reactor.label, reactorFontSize, strokeTranslation))
				nameToBox[reactor.name] = placed(svgBox{x: reactor.x, y: reactor.y, w: reactor.w, h: reactor.h, lane: commandsAndViewsLane})
			}
		}

		// --- External system boxes (Translations), side by side ---
		systems := len(s.Translations)
		systemsX := sliceX + (column.w-systems*externalBoxWidth-(systems-1)*externalBoxGap)/2
		for ti, tr := range s.Translations {
			extX := systemsX + ti*(externalBoxWidth+externalBoxGap)
			extY := extRowY - externalBoxHeight/2
			b.WriteString(svgDashedRoundedRect(extX, extY, externalBoxWidth, externalBoxHeight, fillExternal, strokeExternal, 5, tr.Description))
			b.WriteString(svgText(extX+externalBoxWidth/2, extY+externalBoxHeight/2, tr.ExternalSystem, 11, strokeExternal))
			externalBoxes[tr.Name] = placed(svgBox{x: extX, y: extY, w: externalBoxWidth, h: externalBoxHeight, lane: externalSystemsLane})
		}
	}

	// --- Connections ---

	var arrows []boxArrow
	joinReader := func(i int, edge Edge, reader svgBox) {
		if view, drawn := joinedBox(i, edge); drawn {
			arrows = append(arrows, boxArrow{from: view, to: reader})
		}
	}

	for i, entry := range entries {
		rejected := 0
		for _, edge := range SliceEdges(entry.slice) {
			switch edge.Kind {
			case EdgeTriggerReads:
				if reader, drawn := triggerBoxes[entry.slice]; drawn {
					joinReader(i, edge, reader)
				}

			case EdgeAutomationReads:
				if reader, drawn := nameToBox[edge.To]; drawn {
					joinReader(i, edge, reader)
				}

			case EdgeScreenReads:
				if reader, drawn := appearanceBoxes[i][edge.To]; drawn {
					joinReader(i, edge, reader)
				}

			case EdgeScreenCommand:
				to, toDrawn := joinedBox(i, edge)
				if toDrawn {
					arrows = append(arrows, boxArrow{from: appearanceBoxes[i][edge.From], to: to})
				}

			case EdgeTriggerCommand:
				command := DeclaredElement{Kind: ElementCommand, Name: edge.To, Slice: entry.slice}
				arrows = append(arrows, boxArrow{from: triggerBoxes[entry.slice], to: declaredBoxes[command]})

			case EdgeTranslationReads:
				// A translation's reads arrow is drawn into the external system box
				// fronting its reactor, not the reactor itself.
				joinReader(i, edge, externalBoxes[edge.To])

			case EdgeTranslationExternal:
				arrows = append(arrows, boxArrow{from: externalBoxes[edge.To], to: nameToBox[edge.To]})

			case EdgeAutomationCommand, EdgeTranslationCommand:
				from, fromDrawn := nameToBox[edge.From]
				to, toDrawn := joinedBox(i, edge)
				if fromDrawn && toDrawn {
					arrows = append(arrows, boxArrow{from: from, to: to})
				}

			case EdgeTranslationFlow:
				from, fromDrawn := joinedBox(i, edge)
				to, toDrawn := nameToBox[edge.To]
				if fromDrawn && toDrawn {
					arrows = append(arrows, boxArrow{from: from, to: to})
				}

			case EdgeAutomationTrigger:
				from, fromDrawn := joinedBox(i, edge)
				to, toDrawn := nameToBox[edge.To]
				if fromDrawn && toDrawn {
					arrows = append(arrows, boxArrow{from: from, to: to, label: edge.Label})
				}

			case EdgeRejection:
				// SliceEdges emits rejection edges in declaration order, so the
				// nth one this slice states ends at the nth badge it drew.
				from, fromDrawn := nameToBox[edge.From]
				if fromDrawn && rejected < len(badges[i]) {
					arrows = append(arrows, boxArrow{from: from, to: badges[i][rejected], dashed: true})
				}
				rejected++

			case EdgeSubscription:
				from, fromDrawn := joinedBox(i, edge)
				to, toDrawn := nameToBox[edge.To]
				if fromDrawn && toDrawn {
					arrows = append(arrows, boxArrow{from: from, to: to})
				}

			default:
				from, fromDrawn := nameToBox[edge.From]
				to, toDrawn := nameToBox[edge.To]
				if fromDrawn && toDrawn {
					arrows = append(arrows, boxArrow{from: from, to: to})
				}
			}
		}
	}

	return b.String(), arrows, obstacles
}

func screenDescriptions(model *ast.Model) map[string]string {
	descriptions := make(map[string]string, len(model.Screens))
	for _, screen := range model.Screens {
		descriptions[screen.Name] = screen.Description
	}
	return descriptions
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
func svgCopy(c DeclaredElement, box svgBox) string {
	stroke := strokeEvent
	switch c.Kind {
	case ElementView:
		stroke = strokeView
	case ElementCommand:
		stroke = strokeCommand
	}

	return svgDashedRoundedRect(box.x, box.y, box.w, box.h, fillCopy, stroke, 5, "") +
		svgMultilineText(box.x+box.w/2, box.y+box.h/2, strings.Join(c.copyLabelLines(), "\n"), 11, stroke)
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

// svgBox is where a shape was drawn: the corner it starts at, how far it runs
// from there, and the lane it was drawn in.
type svgBox struct {
	x, y, w, h int
	lane       int
}

// across is the x across b at which a stretch can meet b, stretchClearance in
// from each corner. Its best x is for the caller to set.
func (b svgBox) across() room {
	return room{left: b.x + stretchClearance, right: b.x + b.w - stretchClearance}
}

// labelAt is where the text of arrow i of routes goes: on its first leg, as near
// the middle of the leg as it can be with no other arrow running through the
// text. An arrow that carries text always has a leg, since the router never
// runs it straight.
func labelAt(routes []route, i int, label string) point {
	first := routes[i].legs[0]
	left, right := first.span()
	middle := point{(left + right) / 2, first.y}
	half := estimatedTextWidth(label, edgeLabelFontSize) / 2
	for step := 0; step < (right-left)/2; step += labelStep {
		for _, x := range []int{middle.x - step, middle.x + step} {
			if !otherArrowCrosses(routes, i, x-half, x+half, middle.y) {
				return point{x, middle.y}
			}
		}
	}

	return middle
}

// otherArrowCrosses reports whether an arrow of routes other than arrow i runs
// through y between left and right.
func otherArrowCrosses(routes []route, i, left, right, y int) bool {
	for j, other := range routes {
		if j == i {
			continue
		}
		points := other.points()
		for k := 1; k < len(points); k++ {
			a, b := points[k-1], points[k]
			if min(a.x, b.x) <= right && max(a.x, b.x) >= left && min(a.y, b.y) <= y && max(a.y, b.y) >= y {
				return true
			}
		}
	}

	return false
}

// svgArrowLabel paints an arrow's text centred on at.
// The class is what separates this text from a box's own: a reader binding
// every <text> to the last rect it saw would otherwise relabel that box.
func svgArrowLabel(at point, label string) string {
	return fmt.Sprintf(`<text class="edge-label" x="%d" y="%d" text-anchor="middle" dominant-baseline="middle" font-family="sans-serif" font-size="%d" fill="%s">%s</text>`+"\n",
		at.x, at.y, edgeLabelFontSize, arrowStroke, escapeXML(label))
}

func svgArrow(r route) string {
	return svgArrowPath(r, "")
}

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
	// labelStep is how far along its leg the text of an arrow moves at a time to
	// get clear of another arrow.
	labelStep = 4
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
