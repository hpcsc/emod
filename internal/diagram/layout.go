package diagram

import (
	"fmt"
	"sort"
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
	laneGap    = 30

	laneHeaderHeight = 30
	laneRowInset     = 52

	reactorHeight = boxHeight * 3 / 4
	reactorGap    = 6

	reactorFontSize = 10
	laneGearSize    = 28
	gearLabelGap    = 3
	// automationHeight is a gear and the two lines a scheduled automation
	// writes under it, svgMultilineText's line height apart.
	automationHeight = laneGearSize + gearLabelGap + 2*reactorFontSize + 4
	laneBottomInset  = 4

	laneHeight = laneHeaderHeight + laneRowInset + boxHeight + reactorGap + automationHeight + laneBottomInset
)

// Color constants for element types.
const (
	fillEvent         = "#ffe6cc"
	strokeEvent       = "#d79b00"
	fillCommand       = "#dae8fc"
	strokeCommand     = "#6c8ebf"
	fillView          = "#d5e8d4"
	strokeView        = "#82b366"
	fillTrigger       = "#ffffff"
	strokeTrigger     = "#333333"
	fillExternal      = "#f5f5f5"
	strokeExternal    = "#666666"
	fillAutomation    = "#16201b"
	fillTranslation   = "#f5f5f5"
	strokeTranslation = "#666666"

	// A rejection badge is not an element type and holds no palette entry of its
	// own; these name what it paints so it can be repainted apart from the six.
	fillRejection   = "#f8cecc"
	strokeRejection = "#b85450"

	// A spec card is not an element type either, and likewise holds no palette
	// entry: it states scenarios rather than standing for a construct a viewer
	// draws.
	fillSpecCard   = "#fff2cc"
	strokeSpecCard = "#d6b656"

	// A copy of an event takes the event's stroke, so it reads as that event,
	// and a fill of its own, so it does not read as a second declaration.
	fillEventCopy = "#ffffff"
)

// Spec card geometry.
const (
	specCardFontSize   = 10
	specCardLineHeight = specCardFontSize + 4
	specCardPadding    = 10
	// specBandGap separates a card from the edges of the band holding it.
	specBandGap = 12
)

// External system box geometry. A slice's translations stack downwards from the
// row they start on, so a slice declaring several reaches below the lane.
const (
	externalBoxWidth  = 100
	externalBoxHeight = 45
	externalBoxGap    = 8
)

type sliceEntry struct {
	slice          *ast.Slice
	ctxName        string
	ctxDescription string
	fromDCB        bool // true if slice comes from a direct (DCB) context
	// invariants are the ones this slice may name, which is its aggregate's or,
	// for a slice declared directly on a context, that context's — the scope
	// rule the validator resolves a rejection against.
	invariants []*ast.Invariant
	copies     []EventCopy
}

// invariantStatement returns the prose of the named invariant, or "" when the
// slice's own scope declares no such name. Resolving within the scope rather
// than across the model keeps two aggregates free to declare one name each.
func (e sliceEntry) invariantStatement(name string) string {
	for _, inv := range e.invariants {
		if inv != nil && inv.Name == name {
			return inv.Statement
		}
	}
	return ""
}

// collectSlices flattens all slices from the model into a list, in source
// order within each context.
func collectSlices(model *ast.Model) []sliceEntry {
	copies := EventCopies(model)

	var entries []sliceEntry
	for _, ref := range model.SliceRefs() {
		invariants := ref.Context.Invariants
		if ref.Aggregate != nil {
			invariants = ref.Aggregate.Invariants
		}
		entries = append(entries, sliceEntry{
			slice:          ref.Slice,
			ctxName:        ref.Context.Name,
			ctxDescription: ref.Context.Description,
			fromDCB:        ref.Aggregate == nil,
			invariants:     invariants,
			copies:         copies[ref.Slice],
		})
	}
	return entries
}

// eventRowItems counts the boxes a slice draws in its events row other than its
// copies: its events, the events its translations nest, and its rejection badges.
func eventRowItems(s *ast.Slice) int {
	return len(declaredEvents(s)) + len(s.Rejections)
}

// copyMeasureFontSize is the size a copy's label is measured at. It is 1 px
// above the 11 px the SVG draws an event's label at, so the box is measured a
// little wider than its text.
const copyMeasureFontSize = 12

// copyLabelPadding keeps the longest line of a copy clear of its outline.
const copyLabelPadding = 16

// columnWidth is how wide a slice is drawn. A slice that draws copies is widened
// until every box in its events row is as wide as the longest line of a copy.
func columnWidth(entry sliceEntry) int {
	if len(entry.copies) == 0 {
		return sliceWidth
	}

	var longest int
	for _, c := range entry.copies {
		for _, line := range c.labelLines() {
			longest = max(longest, estimatedTextWidth(line, copyMeasureFontSize))
		}
	}
	items := eventRowItems(entry.slice) + len(entry.copies)
	itemW := min(longest+copyLabelPadding, boxWidth)

	return max(sliceWidth, 20+items*itemW+(items-1)*8)
}

// sliceColumn is the band of the layout a slice is drawn in: the x it starts at,
// and how wide it runs from there.
type sliceColumn struct {
	x, w int
}

// sliceColumns returns the column each slice is drawn in, left to right, with a
// wider gap wherever one context gives way to the next.
func sliceColumns(entries []sliceEntry) []sliceColumn {
	columns := make([]sliceColumn, len(entries))
	x := marginX

	for i, entry := range entries {
		if i > 0 {
			if entry.ctxName == entries[i-1].ctxName {
				x += sliceGap
			} else {
				x += contextGap
			}
		}
		columns[i] = sliceColumn{x: x, w: columnWidth(entry)}
		x += columns[i].w
	}

	return columns
}

// contextBound is the horizontal band a context occupies, spanning the slices
// declared in it.
type contextBound struct {
	name        string
	description string
	x           int
	w           int
}

// contextBounds returns one band per context, laid over the slices in the given
// columns. A band stops short of the gap separating it from the next context;
// only the last one runs to the right edge of the layout.
func contextBounds(entries []sliceEntry, columns []sliceColumn) []contextBound {
	var bounds []contextBound

	for i, entry := range entries {
		if i > 0 && entry.ctxName == entries[i-1].ctxName {
			continue
		}
		if len(bounds) > 0 {
			bounds[len(bounds)-1].w = columns[i-1].x + columns[i-1].w - contextGap - bounds[len(bounds)-1].x
		}
		bounds = append(bounds, contextBound{name: entry.ctxName, description: entry.ctxDescription, x: columns[i].x})
	}
	if len(bounds) > 0 {
		bounds[len(bounds)-1].w = layoutWidth(columns) - bounds[len(bounds)-1].x
	}

	return bounds
}

// layoutWidth returns the x the rightmost slice ends at.
func layoutWidth(columns []sliceColumn) int {
	if len(columns) == 0 {
		return marginX
	}
	last := columns[len(columns)-1]
	return last.x + last.w
}

// collectTagKeys returns unique tag keys across all DCB events, sorted alphabetically.
func collectTagKeys(entries []sliceEntry) []string {
	keySet := make(map[string]struct{})
	for _, e := range entries {
		if !e.fromDCB {
			continue
		}
		for _, evt := range e.slice.Events {
			for _, tag := range evt.Tags {
				keySet[tag.Key] = struct{}{}
			}
		}
	}
	if len(keySet) == 0 {
		return nil
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// laneRowY is where a lane's row of boxes starts, below the strip carrying the
// lane's own label.
func laneRowY(laneY int) int {
	return laneY + laneHeaderHeight + laneRowInset
}

// reactorBox is the box drawn for a translation reactor, or the square the gear
// of an automation is drawn in.
type reactorBox struct {
	name         string
	label        string
	description  string
	isAutomation bool
	x, y, w, h   int
}

// reactorBoxes returns the box drawn for each of a slice's automations and then
// for each of its translation reactors: a row of their own under the commands
// and views they wire to, sharing that lane and laid out across the slice the
// way that row is. lineBreak is how the format drawing them starts a new line.
func reactorBoxes(s *ast.Slice, cmdViewLaneY int, column sliceColumn, lineBreak string) []reactorBox {
	boxes := make([]reactorBox, 0, len(s.Automations)+len(s.Translations))
	for _, auto := range s.Automations {
		boxes = append(boxes, automationBox(auto, lineBreak))
	}
	for _, tr := range s.Translations {
		boxes = append(boxes, translationBox(tr))
	}

	rowY := laneRowY(cmdViewLaneY) + boxHeight + reactorGap
	for i := range boxes {
		boxes[i].w, boxes[i].x = itemLayout(column.w-20, len(boxes), i, column.x)
		boxes[i].y, boxes[i].h = rowY, reactorHeight
		if boxes[i].isAutomation {
			boxes[i].placeGear()
		}
	}

	return boxes
}

// placeGear shrinks an automation's box to the square its gear is drawn in, at
// the top of the box and centred across it, with the label under the gear.
func (r *reactorBox) placeGear() {
	r.x += (r.w - laneGearSize) / 2
	r.w, r.h = laneGearSize, laneGearSize
}

// automationBox returns the reactor box for an automation.
func automationBox(auto *ast.Automation, lineBreak string) reactorBox {
	return reactorBox{
		name:         auto.Name,
		label:        strings.Join(automationLabelLines(auto), lineBreak),
		description:  auto.Description,
		isAutomation: true,
	}
}

// translationBox returns the reactor box for a translation reactor.
func translationBox(tr *ast.Translation) reactorBox {
	return reactorBox{
		name:        tr.Name,
		label:       reactorLabel(tr.Name),
		description: tr.Description,
	}
}

// specCard is the card drawn for one slice's scenarios.
type specCard struct {
	label      string
	x, y, w, h int
}

// specCards returns the card drawn under each slice that states at least one
// scenario, in that slice's own column, all of them starting at the same y so
// the band reads as one row. A slice stating none contributes no card, which is
// what keeps a model without specs drawing exactly what it drew before.
// lineBreak is how the format drawing them starts a new line.
func specCards(entries []sliceEntry, columns []sliceColumn, bandY int, lineBreak string) []specCard {
	cards := make([]specCard, 0, len(entries))
	for i, entry := range entries {
		lines := wrapCardLines(specCardLines(entry.slice.Specs))
		if len(lines) == 0 {
			continue
		}
		cards = append(cards, specCard{
			label: strings.Join(lines, lineBreak),
			x:     columns[i].x + 10,
			y:     bandY + laneHeaderHeight + specBandGap,
			w:     columns[i].w - 20,
			h:     len(lines)*specCardLineHeight + 2*specCardPadding,
		})
	}

	return cards
}

// externalBoxesBottom is the lowest edge the external system boxes reach. A
// slice declaring more than two translations stacks them past the bottom of the
// lane they start in, so this is not the lane's own edge.
func externalBoxesBottom(entries []sliceEntry, extRowY int) int {
	var most int
	for _, entry := range entries {
		most = max(most, len(entry.slice.Translations))
	}
	if most == 0 {
		return 0
	}

	return extRowY - externalBoxHeight/2 + (most-1)*(externalBoxHeight+externalBoxGap) + externalBoxHeight
}

// specBandTop is where the band holding the cards starts: a lane gap below
// whichever reaches lower, the lowest lane or a box that overflowed it. Taking
// the lane's edge alone would draw the band, which is opaque and written last,
// over any external system box that stacked past it.
func specBandTop(entries []sliceEntry, extLaneY int) int {
	return max(extLaneY+laneHeight, externalBoxesBottom(entries, laneRowY(extLaneY))) + laneGap
}

// specBandHeight is how tall the band holding the cards has to be: the tallest
// card, the strip carrying the band's own label, and a gap above and below.
// A card stating several scenarios will not fit the fixed laneHeight the lanes
// above take, so the band is sized to its contents rather than to that constant.
func specBandHeight(cards []specCard) int {
	var tallest int
	for _, card := range cards {
		tallest = max(tallest, card.h)
	}
	if tallest == 0 {
		return 0
	}

	return laneHeaderHeight + tallest + 2*specBandGap
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
