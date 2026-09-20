package diagram

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/hpcsc/emod/internal/ast"
)

// Event flow geometry. A gear is drawn from the origin outwards, so its box is
// what the layout reserves around that centre.
const (
	flowMargin          = 48
	flowRowGap          = 80
	flowNodeGap         = 44
	flowNodeHeight      = 44
	flowGearSize        = 44
	flowGearScale       = 1.5
	flowMinNodeWidth    = 108
	flowNodePadding     = 28
	flowCaptionTop      = 14
	flowCaptionLine     = 15
	flowCaptionBottom   = 4
	flowNameFontSize    = 13
	flowCaptionFontSize = 11
	flowStraightSlack   = 6
	flowLabelGap        = 6
)

// ExportEventFlow draws a model as an event flow: orange event pills, bare gear
// automations, and the arrow from each event to whatever reacts to it. The
// commands are collapsed away and the views left out, so the picture states
// what the system appends rather than every construct the model declares.
//
// The drawing carries its own colours and a dark-mode override, which the lane
// diagrams do not, so it reads on a page of either theme.
func ExportEventFlow(model *ast.Model, _ Style) ([]byte, error) {
	flow := CollapseEventFlow(model)
	placed, width, height := placeEventFlow(flow)

	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`+"\n",
		width, height, width, height))
	b.WriteString(eventFlowStyle)
	b.WriteString(eventFlowDefs)
	b.WriteString(fmt.Sprintf(`<rect class="paper" x="0" y="0" width="%d" height="%d"/>`+"\n", width, height))

	if len(placed) == 0 {
		b.WriteString("</svg>\n")
		return []byte(b.String()), nil
	}

	at := make(map[string]placedNode, len(placed))
	for _, node := range placed {
		at[node.node.ID] = node
	}

	// Arrows first, then the shapes, then every piece of text: a halo masks
	// only what is painted before it, and SVG paints in document order.
	var labels strings.Builder
	for _, hop := range flow.Hops {
		from, drawn := at[hop.From]
		to, reached := at[hop.To]
		if !drawn || !reached {
			continue
		}
		b.WriteString(eventFlowArrow(from, to, to.node.Kind == NodeRejection))
		labels.WriteString(eventFlowHopLabel(from, to, hop.Label))
	}

	for _, node := range placed {
		b.WriteString(node.shape())
		labels.WriteString(node.text())
	}

	b.WriteString(labels.String())
	b.WriteString("</svg>\n")

	return []byte(b.String()), nil
}

type placedNode struct {
	node     Node
	x, y     int
	w, h     int
	slot     int
	captions []flowCaption
}

type flowCaption struct {
	text  string
	class string
	size  int
}

func (p placedNode) centreX() int { return p.x + p.w/2 }

func (p placedNode) centreY() int { return p.y + p.h/2 }

// bottom is where an arrow leaving this node starts: below its captions, which
// an arrow from the shape's own edge would otherwise cross.
func (p placedNode) bottom() int {
	if len(p.captions) == 0 {
		return p.y + p.h
	}

	return p.y + p.h + flowCaptionTop + (len(p.captions)-1)*flowCaptionLine + flowCaptionBottom
}

func (p placedNode) shape() string {
	switch p.node.Kind {
	case NodeAutomation:
		return fmt.Sprintf(`<use href="#gear" transform="translate(%d,%d) scale(%g)">%s</use>`+"\n",
			p.centreX(), p.centreY(), flowGearScale, svgTitle(p.node.Description))
	case NodeEvent:
		class := "pill"
		if p.node.DeadEnd {
			class = "pill dead"
		}
		return p.rect(class, p.h/2)
	case NodeRejection:
		return p.rect("reject", 8)
	default:
		return p.rect("store", 8)
	}
}

func (p placedNode) rect(class string, rx int) string {
	return fmt.Sprintf(`<rect class="%s" x="%d" y="%d" width="%d" height="%d" rx="%d">%s</rect>`+"\n",
		class, p.x, p.y, p.w, p.h, rx, svgTitle(p.node.Description))
}

// text writes what the node says: a name inside every shape but the gear, which
// has no box to hold one, and each caption below.
func (p placedNode) text() string {
	var b strings.Builder
	if p.node.Kind != NodeAutomation {
		class := "name"
		if p.node.Kind == NodeRejection {
			class = "name alert"
		}
		b.WriteString(fmt.Sprintf(
			`<text class="%s" x="%d" y="%d" text-anchor="middle" dominant-baseline="middle">%s</text>`+"\n",
			class, p.centreX(), p.centreY(), escapeXML(p.node.Label)))
	}

	y := p.y + p.h + flowCaptionTop
	for _, caption := range p.captions {
		b.WriteString(fmt.Sprintf(`<text class="%s" x="%d" y="%d" text-anchor="middle">%s</text>`+"\n",
			caption.class, p.centreX(), y, escapeXML(caption.text)))
		y += flowCaptionLine
	}

	return b.String()
}

func svgTitle(description string) string {
	if description == "" {
		return ""
	}

	return "<title>" + escapeXML(description) + "</title>"
}

func eventFlowArrow(from, to placedNode, refused bool) string {
	class, head := "hop", "head"
	if refused {
		class, head = "hop refused", "head-refused"
	}

	sx, sy := from.centreX(), from.bottom()
	tx, ty := to.centreX(), to.y
	// Two nodes a row apart are centred within rows of different widths, so
	// their centres miss each other by a pixel or two. An elbow drawn for that
	// reads as a step in the arrow rather than as a route around anything.
	if abs(sx-tx) <= flowStraightSlack {
		return fmt.Sprintf(`<path class="%s" d="M %d,%d L %d,%d" marker-end="url(#%s)"/>`+"\n",
			class, sx, sy, tx, ty, head)
	}

	midY := (sy + ty) / 2
	return fmt.Sprintf(`<path class="%s" d="M %d,%d L %d,%d L %d,%d L %d,%d" marker-end="url(#%s)"/>`+"\n",
		class, sx, sy, sx, midY, tx, midY, tx, ty, head)
}

func eventFlowHopLabel(from, to placedNode, label string) string {
	if label == "" {
		return ""
	}

	sx, tx := from.centreX(), to.centreX()
	midY := (from.bottom() + to.y) / 2
	// Above the leg it names, not on it: the halo that would otherwise mask the
	// line is a browser's doing, and the renderers a diagram is read in
	// elsewhere paint the line straight through the text.
	if abs(sx-tx) <= flowStraightSlack {
		return fmt.Sprintf(`<text class="hop-label" x="%d" y="%d">%s</text>`+"\n",
			sx+flowLabelGap, midY, escapeXML(label))
	}

	return fmt.Sprintf(`<text class="hop-label" x="%d" y="%d" text-anchor="middle">%s</text>`+"\n",
		(sx+tx)/2, midY-flowLabelGap, escapeXML(label))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}

	return n
}

// placeEventFlow lays the flow out in rows, each row holding the nodes whose
// every producer sits in a row above it. Nodes in a cycle reach no such row, so
// they take one of their own below the rest.
func placeEventFlow(flow EventFlow) ([]placedNode, int, int) {
	placed := make([]placedNode, len(flow.Nodes))
	for i, node := range flow.Nodes {
		placed[i] = measureFlowNode(node)
	}

	rows := flowRows(flow)

	widest := 0
	for _, row := range rows {
		if width := rowWidth(placed, row); width > widest {
			widest = width
		}
	}

	canvas := widest + 2*flowMargin
	top := flowMargin
	for _, row := range rows {
		height, depth := 0, 0
		for _, i := range row {
			if placed[i].h > height {
				height = placed[i].h
			}
			if below := placed[i].bottom() - placed[i].h; below > depth {
				depth = below
			}
		}

		x := (canvas - rowWidth(placed, row)) / 2
		for _, i := range row {
			placed[i].x = x + (placed[i].slot-placed[i].w)/2
			placed[i].y = top + (height-placed[i].h)/2
			x += placed[i].slot + flowNodeGap
		}

		top += height + depth + flowRowGap
	}

	if len(rows) == 0 {
		return nil, 2 * flowMargin, 2 * flowMargin
	}

	return placed, canvas, top - flowRowGap + flowMargin
}

func rowWidth(placed []placedNode, row []int) int {
	width := 0
	for _, i := range row {
		width += placed[i].slot
	}

	return width + (len(row)-1)*flowNodeGap
}

func measureFlowNode(node Node) placedNode {
	p := placedNode{node: node, h: flowNodeHeight}

	if node.Kind == NodeAutomation {
		p.w, p.h = flowGearSize, flowGearSize
		p.captions = append(p.captions, flowCaption{text: node.Label, class: "gear-name", size: flowNameFontSize})
	} else {
		p.w = flowTextWidth(node.Label, flowNameFontSize) + flowNodePadding
		if p.w < flowMinNodeWidth {
			p.w = flowMinNodeWidth
		}
	}
	if node.Caption != "" {
		p.captions = append(p.captions, flowCaption{text: node.Caption, class: "caption", size: flowCaptionFontSize})
	}

	p.slot = p.w
	for _, caption := range p.captions {
		if width := flowTextWidth(caption.text, caption.size); width > p.slot {
			p.slot = width
		}
	}

	return p
}

// flowTextWidth estimates how wide a label is drawn, standing in for a text
// measurement no exporter has. The ratio is the one specCardLineBudget was
// calibrated against: about 0.57em for a line of CamelCase construct names.
func flowTextWidth(text string, fontSize int) int {
	return utf8.RuneCountInString(text) * fontSize * 57 / 100
}

// flowRows groups the nodes into rows by how far each sits from an origin: a
// node takes the row below the last of its producers.
func flowRows(flow EventFlow) [][]int {
	index := make(map[string]int, len(flow.Nodes))
	for i, node := range flow.Nodes {
		index[node.ID] = i
	}

	reaches := make([][]int, len(flow.Nodes))
	producers := make([]int, len(flow.Nodes))
	for _, hop := range flow.Hops {
		from, drawn := index[hop.From]
		to, reached := index[hop.To]
		if !drawn || !reached || from == to {
			continue
		}
		reaches[from] = append(reaches[from], to)
		producers[to]++
	}

	rowed := make([]bool, len(flow.Nodes))
	var row []int
	for i := range flow.Nodes {
		if producers[i] == 0 {
			row = append(row, i)
			rowed[i] = true
		}
	}

	var rows [][]int
	for len(row) > 0 {
		rows = append(rows, row)
		var next []int
		for _, i := range row {
			for _, j := range reaches[i] {
				producers[j]--
				if producers[j] == 0 && !rowed[j] {
					next = append(next, j)
					rowed[j] = true
				}
			}
		}
		sort.Ints(next)
		row = next
	}

	var cycled []int
	for i, done := range rowed {
		if !done {
			cycled = append(cycled, i)
		}
	}
	if len(cycled) > 0 {
		rows = append(rows, cycled)
	}

	return rows
}

// Every colour is written out in full, in both themes. A renderer outside a
// browser resolves no custom property, and a fill that fails to resolve is
// painted black, so a var() here would draw the whole picture as silhouettes
// everywhere but a browser. A renderer that ignores the media query shows the
// light theme, which is the right fallback.
const eventFlowStyle = `<style>
text { font-family: ui-sans-serif, system-ui, -apple-system, "Helvetica Neue", Arial, sans-serif; }
.paper { fill: #FFFFFF; }
.pill { fill: #FFE0C0; stroke: #B44E12; stroke-width: 1.6px; }
.pill.dead { stroke: #A33A2A; stroke-dasharray: 6 4; }
.store { fill: #ECEFED; stroke: #5A6861; stroke-width: 1.4px; }
.reject { fill: #FFFFFF; stroke: #A33A2A; stroke-width: 1.4px; stroke-dasharray: 6 4; }
.gear { fill: #16201B; }
.hub { fill: #FFFFFF; }
.hop { fill: none; stroke: #16201B; stroke-width: 1.4px; }
.hop.refused { stroke: #A33A2A; stroke-dasharray: 6 4; }
.head { fill: #16201B; }
.head-refused { fill: #A33A2A; }
.name { fill: #16201B; font-size: 13px; }
.name.alert { fill: #A33A2A; }
.gear-name { fill: #16201B; font-size: 13px; paint-order: stroke; stroke: #FFFFFF; stroke-width: 5px; }
.caption { fill: #5A6861; font-size: 11px; paint-order: stroke; stroke: #FFFFFF; stroke-width: 5px; }
.hop-label { fill: #5A6861; font-size: 11px; paint-order: stroke; stroke: #FFFFFF; stroke-width: 5px; }
@media (prefers-color-scheme: dark) {
.paper { fill: #16201B; }
.pill { fill: #40260F; stroke: #F2A868; }
.pill.dead { stroke: #E88C78; }
.store { fill: #1E2723; stroke: #93A199; }
.reject { fill: #16201B; stroke: #E88C78; }
.gear { fill: #E2E9E5; }
.hub { fill: #16201B; }
.hop { stroke: #E2E9E5; }
.hop.refused { stroke: #E88C78; }
.head { fill: #E2E9E5; }
.head-refused { fill: #E88C78; }
.name { fill: #E2E9E5; }
.name.alert { fill: #E88C78; }
.gear-name { fill: #E2E9E5; stroke: #16201B; }
.caption { fill: #93A199; stroke: #16201B; }
.hop-label { fill: #93A199; stroke: #16201B; }
}
</style>
`

// The gear is eight teeth at 45 degrees around a hub, and the hole in that hub
// is painted the background colour, which is why it carries a class of its own:
// left to the group's fill it becomes a solid dot in either theme.
const eventFlowDefs = `<defs>
<g id="gear" class="gear">
<circle cx="0" cy="0" r="8.4"/>
<rect x="-2.9" y="-14.2" width="5.8" height="6.8" rx="1.3"/>
<rect x="-2.9" y="-14.2" width="5.8" height="6.8" rx="1.3" transform="rotate(45)"/>
<rect x="-2.9" y="-14.2" width="5.8" height="6.8" rx="1.3" transform="rotate(90)"/>
<rect x="-2.9" y="-14.2" width="5.8" height="6.8" rx="1.3" transform="rotate(135)"/>
<rect x="-2.9" y="-14.2" width="5.8" height="6.8" rx="1.3" transform="rotate(180)"/>
<rect x="-2.9" y="-14.2" width="5.8" height="6.8" rx="1.3" transform="rotate(225)"/>
<rect x="-2.9" y="-14.2" width="5.8" height="6.8" rx="1.3" transform="rotate(270)"/>
<rect x="-2.9" y="-14.2" width="5.8" height="6.8" rx="1.3" transform="rotate(315)"/>
<circle cx="0" cy="0" r="3.4" class="hub"/>
</g>
<marker id="head" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto">
<path class="head" d="M 0 0 L 10 5 L 0 10 z"/>
</marker>
<marker id="head-refused" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto">
<path class="head-refused" d="M 0 0 L 10 5 L 0 10 z"/>
</marker>
</defs>
`
