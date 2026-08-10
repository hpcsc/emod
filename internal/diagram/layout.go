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
	laneHeight = 190
	laneGap    = 30

	laneHeaderHeight = 30

	reactorHeight = boxHeight * 3 / 4
	reactorGap    = 6

	waypointMargin = 40
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
	fillAutomation    = "#e1d5e7"
	strokeAutomation  = "#9673a6"
	fillTranslation   = "#f5f5f5"
	strokeTranslation = "#666666"
	strokePurpleUp    = "#9B59B6"
	strokeGreenUp     = "#82b366"
	strokeStandard    = "#333333"

	// A rejection badge is not an element type and holds no palette entry of its
	// own; these name what it paints so it can be repainted apart from the six.
	fillRejection   = "#f8cecc"
	strokeRejection = "#b85450"
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
		})
	}
	return entries
}

// sliceXPositions returns the x each slice is drawn at, left to right, with a
// wider gap wherever one context gives way to the next.
func sliceXPositions(entries []sliceEntry) []int {
	positions := make([]int, len(entries))
	x := marginX

	for i, entry := range entries {
		if i > 0 {
			if entry.ctxName == entries[i-1].ctxName {
				x += sliceGap
			} else {
				x += contextGap
			}
		}
		positions[i] = x
		x += sliceWidth
	}

	return positions
}

// contextBound is the horizontal band a context occupies, spanning the slices
// declared in it.
type contextBound struct {
	name        string
	description string
	x           int
	w           int
}

// contextBounds returns one band per context, laid over the slices at the given
// positions. A band stops short of the gap separating it from the next context;
// only the last one runs to the right edge of the layout.
func contextBounds(entries []sliceEntry, sliceXs []int) []contextBound {
	var bounds []contextBound

	for i, entry := range entries {
		if i > 0 && entry.ctxName == entries[i-1].ctxName {
			continue
		}
		if len(bounds) > 0 {
			bounds[len(bounds)-1].w = sliceXs[i-1] + sliceWidth - contextGap - bounds[len(bounds)-1].x
		}
		bounds = append(bounds, contextBound{name: entry.ctxName, description: entry.ctxDescription, x: sliceXs[i]})
	}
	if len(bounds) > 0 {
		bounds[len(bounds)-1].w = layoutWidth(sliceXs) - bounds[len(bounds)-1].x
	}

	return bounds
}

// layoutWidth returns the x the rightmost slice ends at.
func layoutWidth(sliceXs []int) int {
	if len(sliceXs) == 0 {
		return marginX
	}
	return sliceXs[len(sliceXs)-1] + sliceWidth
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

// hasTranslationEvents reports whether a slice has translation events.
func hasTranslationEvents(s *ast.Slice) bool {
	for _, tr := range s.Translations {
		if tr.Event != nil && tr.Event.Name != "" {
			return true
		}
	}
	return false
}

// laneRowY is where a lane's row of boxes starts: centred in what the lane has
// left below the strip carrying its own label.
func laneRowY(laneY int) int {
	return laneY + laneHeaderHeight + (laneHeight-laneHeaderHeight-boxHeight)/2
}

// reactorBox is the box drawn for an automation or for a translation reactor.
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
func reactorBoxes(s *ast.Slice, cmdViewLaneY, sliceX int, lineBreak string) []reactorBox {
	boxes := make([]reactorBox, 0, len(s.Automations)+len(s.Translations))
	for _, auto := range s.Automations {
		boxes = append(boxes, automationBox(auto, lineBreak))
	}
	for _, tr := range s.Translations {
		boxes = append(boxes, translationBox(tr))
	}

	rowY := laneRowY(cmdViewLaneY) + boxHeight + reactorGap
	for i := range boxes {
		boxes[i].w, boxes[i].x = itemLayout(sliceWidth-20, len(boxes), i, sliceX)
		boxes[i].y, boxes[i].h = rowY, reactorHeight
	}

	return boxes
}

// automationBox returns the reactor box for an automation.
func automationBox(auto *ast.Automation, lineBreak string) reactorBox {
	return reactorBox{
		name:         auto.Name,
		label:        automationLabel(auto, lineBreak),
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
