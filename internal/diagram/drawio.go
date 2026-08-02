// Package diagram renders AST models as diagrams.
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
	fillEvent        = "#ffe6cc"
	strokeEvent      = "#d79b00"
	fillCommand      = "#dae8fc"
	strokeCommand    = "#6c8ebf"
	fillView         = "#d5e8d4"
	strokeView       = "#82b366"
	fillTrigger      = "#ffffff"
	strokeTrigger    = "#333333"
	fillExternal     = "#f5f5f5"
	strokeExternal   = "#666666"
	fillAutomation   = "#e1d5e7"
	strokeAutomation = "#9673a6"
	fillTranslation  = "#f5f5f5"
	strokeTranslation= "#666666"
	strokePurpleUp   = "#9B59B6"
	strokeGreenUp    = "#82b366"
	strokeStandard   = "#333333"
)

// mxGraph styles for the box drawn per element type.
const (
	boxBase = "rounded=0;whiteSpace=wrap;html=1;"
	boxFont = "fontFamily=Helvetica;"

	styleContextLabel   = boxBase + "fillColor=" + fillExternal + ";strokeColor=" + strokeExternal + ";fontStyle=1;"
	styleTrigger        = boxBase + "fillColor=" + fillTrigger + ";strokeColor=" + strokeTrigger + ";" + boxFont
	styleCommand        = boxBase + "fillColor=" + fillCommand + ";strokeColor=" + strokeCommand + ";" + boxFont
	styleView           = boxBase + "fillColor=" + fillView + ";strokeColor=" + strokeView + ";" + boxFont
	styleEvent          = boxBase + "fillColor=" + fillEvent + ";strokeColor=" + strokeEvent + ";" + boxFont
	styleAutomation     = boxBase + "fillColor=" + fillAutomation + ";strokeColor=" + strokeAutomation + ";" + boxFont
	styleTranslation    = boxBase + "fillColor=" + fillTranslation + ";strokeColor=" + strokeTranslation + ";" + boxFont
	styleExternalSystem = boxBase + "fillColor=" + fillExternal + ";strokeColor=" + strokeExternal + ";dashed=1;" + boxFont
)

// ExportDrawio converts a parsed AST model into draw.io XML (mxGraph format).
func ExportDrawio(model *ast.Model, style Style) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	var b strings.Builder

	entries := collectSlices(model)
	if len(entries) == 0 {
		return buildEmptyDiagram(model.Name), nil
	}

	// Determine if projected (tag-based) layout should be used
	hasDCB := false
	hasAggEvents := false
	for _, e := range entries {
		if e.fromDCB {
			hasDCB = true
		} else if len(e.slice.Events) > 0 {
			hasAggEvents = true
		}
	}
	tagKeys := collectTagKeys(entries)
	useProjected := style == StyleProjected && hasDCB && len(tagKeys) > 0
	useDCB := style == StyleDCB && hasDCB

	nextID := 2
	allocID := func() int {
		id := nextID
		nextID++
		return id
	}

	sliceXs := sliceXPositions(entries)
	ctxBounds := contextBounds(entries, sliceXs)
	diagramW := layoutWidth(sliceXs) + marginX

	// --- Calculate swim lane positions ---
	var (
		triggerLaneY  int
		cmdViewLaneY  int
		eventLaneY    int
		extLaneY      int
		tagLaneYs     []int // one Y per tag key in tagKeys
		hasEventsLane bool  // whether an "Events" lane is needed for aggregate/untagged events
	)

	if useDCB {
		// DCB query-lens layout: shared triggers/commands lane,
		// then a single flat Events lane (no tag-based grouping)
		triggerLaneY = marginY
		cmdViewLaneY = triggerLaneY // same lane
		nextY := triggerLaneY + laneHeight + laneGap
		eventLaneY = nextY
		nextY += laneHeight + laneGap
		extLaneY = nextY
		hasEventsLane = true
		tagLaneYs = nil
	} else if useProjected {
		// Projected layout: shared triggers/commands lane,
		// then tag lanes for each unique tag key
		triggerLaneY = marginY
		cmdViewLaneY = triggerLaneY // same lane
		nextY := triggerLaneY + laneHeight + laneGap

		// Check if we need an Events lane for:
		// - aggregate events, or
		// - untagged DCB events, or
		// - translation events (from any slice)
		hasEventsLane = hasAggEvents
		if !hasEventsLane {
			for _, e := range entries {
				if e.fromDCB {
					for _, evt := range e.slice.Events {
						if len(evt.Tags) == 0 {
							hasEventsLane = true
							break
						}
					}
					if !hasEventsLane && hasTranslationEvents(e.slice) {
						hasEventsLane = true
					}
				}
				if hasEventsLane {
					break
				}
			}
		}
		if !hasEventsLane {
			for _, e := range entries {
				if hasTranslationEvents(e.slice) {
					hasEventsLane = true
					break
				}
			}
		}

		if hasEventsLane {
			eventLaneY = nextY
			nextY += laneHeight + laneGap
		} else {
			eventLaneY = 0
		}

		tagLaneYs = make([]int, len(tagKeys))
		for i := range tagKeys {
			tagLaneYs[i] = nextY
			nextY += laneHeight + laneGap
		}
		extLaneY = nextY
	} else {
		// Standard 4-lane layout
		triggerLaneY = marginY
		cmdViewLaneY = triggerLaneY + laneHeight + laneGap
		eventLaneY = cmdViewLaneY + laneHeight + laneGap
		extLaneY = eventLaneY + laneHeight + laneGap
		hasEventsLane = true
		tagLaneYs = nil
	}

	// Write document header
	b.WriteString(xmlProlog(model.Name))
	b.WriteString(rootOpen())

	lane := func(label string, y int) string {
		return swimlaneCell(allocID(), label, marginX, y, diagramW-2*marginX, laneHeight)
	}

	if useDCB {
		b.WriteString(lane("Triggers / Commands", triggerLaneY))
		b.WriteString(lane("Events", eventLaneY))
		b.WriteString(lane("External Systems", extLaneY))
	} else if useProjected {
		b.WriteString(lane("Triggers / Commands", triggerLaneY))
		if hasEventsLane {
			b.WriteString(lane("Events", eventLaneY))
		}
		for ti, key := range tagKeys {
			b.WriteString(lane("Tag: "+key, tagLaneYs[ti]))
		}
		b.WriteString(lane("External Systems", extLaneY))
	} else {
		b.WriteString(lane("Wireframes", triggerLaneY))
		b.WriteString(lane("Commands / Views", cmdViewLaneY))
		b.WriteString(lane("Events", eventLaneY))
		b.WriteString(lane("External Systems", extLaneY))
	}

	// Context labels above the swimlanes
	for _, cb := range ctxBounds {
		cid := allocID()
		label := escapeXML(cb.name)
		b.WriteString(vertexCell(cid, label, cb.description, cb.x, marginY-30, cb.w-20, 22, styleContextLabel))
	}

	triggerRowY := laneRowY(triggerLaneY)
	cmdViewRowY := laneRowY(cmdViewLaneY)
	var eventRowY int
	if hasEventsLane {
		eventRowY = laneRowY(eventLaneY)
	}
	var tagRowYs []int
	for _, ty := range tagLaneYs {
		tagRowYs = append(tagRowYs, laneRowY(ty))
	}
	extRowY := laneRowY(extLaneY)

	type namedElem struct {
		name       string
		id         int
		x, y, w, h int
	}
	var elems []namedElem

	// Multi-tag event tracking for connectors
	type multiTagEntry struct {
		name    string
		cellIDs []int // cell IDs of representations across lanes
	}
	var multiTagEvents []multiTagEntry

	// Place elements per slice
	for i, entry := range entries {
		s := entry.slice
		sliceX := sliceXs[i]

		// --- Trigger (triggers/commands lane in projected, top lane in standard) ---
		if s.Trigger != nil {
			id := allocID()
			framingID := allocID()
			x := sliceX + (sliceWidth-boxWidth)/2
			label := s.Trigger.Name
			if s.Trigger.Actor != "" {
				label = fmt.Sprintf("%s (%s)", s.Trigger.Name, s.Trigger.Actor)
			}
			b.WriteString(vertexCell(id, label, s.Trigger.Description, x, triggerRowY, boxWidth, boxHeight, styleTrigger))
			b.WriteString(triggerFramingCell(framingID, id, x, triggerRowY, boxWidth, boxHeight, strokeTrigger))
			elems = append(elems, namedElem{name: s.Trigger.Name, id: id, x: x, y: triggerRowY, w: boxWidth, h: boxHeight})
		}

		// --- Commands (middle lane) ---
		totalMid := len(s.Commands) + len(s.Views)
		usableW := sliceWidth - 20
		for ci, cmd := range s.Commands {
			id := allocID()
			itemW, x := itemLayout(usableW, totalMid, ci, sliceX)
			label := cmd.Name
			if useDCB && cmd.DecidesOn != nil {
				ann := formatDecidesOnAnnotation(cmd.DecidesOn)
				if ann != "" {
					label = cmd.Name + "\\n" + ann
				}
			}
			b.WriteString(vertexCell(id, label, cmd.Description, x, cmdViewRowY, itemW, boxHeight, styleCommand))
			elems = append(elems, namedElem{name: cmd.Name, id: id, x: x, y: cmdViewRowY, w: itemW, h: boxHeight})
		}

		// --- Views (middle lane) ---
		for vi, view := range s.Views {
			id := allocID()
			idx := len(s.Commands) + vi
			itemW, x := itemLayout(usableW, totalMid, idx, sliceX)
			b.WriteString(vertexCell(id, view.Name, view.Description, x, cmdViewRowY, itemW, boxHeight, styleView))
			elems = append(elems, namedElem{name: view.Name, id: id, x: x, y: cmdViewRowY, w: itemW, h: boxHeight})
		}

		// --- Events ---
		// In projected style, DCB events go to tag lanes; aggregate events go to Events lane.
		// In standard style, all events go to the Events lane.
		usableW = sliceWidth - 20
		totalEvts := len(s.Events)
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				totalEvts++
			}
		}
		ei := 0
		for _, evt := range s.Events {
			label := evt.Name
			if evt.ExternalName != "" {
				label = fmt.Sprintf("%s\\n[%s]", evt.Name, evt.ExternalName)
			}
			// For DCB mode, add tag badges to the event label
			if useDCB && len(evt.Tags) > 0 {
				tagText := formatEventTagBadges(evt.Tags)
				if tagText != "" {
					label = fmt.Sprintf("%s\\n%s", label, tagText)
				}
			}
			if useProjected && entry.fromDCB && len(evt.Tags) > 0 {
				// DCB event with tags: place in each matching tag lane
				// Compute position once, reuse across all matching lanes
				itemW, itemX := itemLayout(usableW, totalEvts, ei, sliceX)
				ei++
				var placedIDs []int
				for ti, key := range tagKeys {
					if !eventHasTag(evt, key) {
						continue
					}
					id := allocID()
					b.WriteString(vertexCell(id, label, evt.Description, itemX, tagRowYs[ti], itemW, boxHeight, styleEvent))
					elems = append(elems, namedElem{name: evt.Name, id: id, x: itemX, y: tagRowYs[ti], w: itemW, h: boxHeight})
					placedIDs = append(placedIDs, id)
				}
				if len(placedIDs) > 1 {
					// Track for multi-tag connector
					multiTagEvents = append(multiTagEvents, multiTagEntry{name: evt.Name, cellIDs: placedIDs})
				}
			} else {
				id := allocID()
				itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
				ei++
				b.WriteString(vertexCell(id, label, evt.Description, x, eventRowY, itemW, boxHeight, styleEvent))
				elems = append(elems, namedElem{name: evt.Name, id: id, x: x, y: eventRowY, w: itemW, h: boxHeight})
			}
		}
		for _, tr := range s.Translations {
			if tr.Event != nil && tr.Event.Name != "" {
				id := allocID()
				itemW, x := itemLayout(usableW, totalEvts, ei, sliceX)
				ei++
				b.WriteString(vertexCell(id, tr.Event.Name, tr.Event.Description, x, eventRowY, itemW, boxHeight, styleEvent))
				elems = append(elems, namedElem{name: tr.Event.Name, id: id, x: x, y: eventRowY, w: itemW, h: boxHeight})
			}
		}

		// --- Automations and translation reactors (middle lane) ---
		for _, reactor := range reactorBoxes(s, cmdViewLaneY, sliceX, "\\n") {
			id := allocID()
			style := styleTranslation
			if reactor.isAutomation {
				style = styleAutomation
			}
			b.WriteString(vertexCell(id, reactor.label, reactor.description, reactor.x, reactor.y, reactor.w, reactor.h, style))
			elems = append(elems, namedElem{name: reactor.name, id: id, x: reactor.x, y: reactor.y, w: reactor.w, h: reactor.h})
		}

		// --- External system boxes (Translations) ---
		for ti, tr := range s.Translations {
			id := allocID()
			extW := 100
			extH := 45
			extX := sliceX + (sliceWidth-extW)/2
			extY := extRowY - extH/2
			if ti > 0 {
				extY += ti * (extH + 8)
			}
			// An external system is only ever named by a translation and holds no
			// prose of its own, so its box shows what that translation says.
			b.WriteString(vertexCell(id, tr.ExternalSystem, tr.Description, extX, extY, extW, extH, styleExternalSystem))
			elems = append(elems, namedElem{name: tr.ExternalSystem, id: id, x: extX, y: extY, w: extW, h: extH})
		}
	}

	// --- Connections ---
	// Style definitions per guideline.
	standardStyle := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;fontFamily=Helvetica;strokeColor=" + strokeStandard + ";endArrow=classic;"
	purpleUpStyle := "edgeStyle=orthogonalEdgeStyle;html=1;fontFamily=Helvetica;strokeColor=" + strokePurpleUp + ";fontSize=10;endArrow=classic;exitX=1;exitY=0.5;exitDx=0;exitDy=0;curved=1;"
	greenUpStyle := "edgeStyle=orthogonalEdgeStyle;html=1;fontFamily=Helvetica;strokeColor=" + strokeGreenUp + ";fontSize=10;endArrow=classic;exitX=1;exitY=0.5;exitDx=0;exitDy=0;entryX=0;entryY=1;entryDx=0;entryDy=0;curved=1;"
	extStyle := "edgeStyle=orthogonalEdgeStyle;rounded=0;orthogonalLoop=1;jettySize=auto;html=1;fontFamily=Helvetica;strokeColor=" + strokeExternal + ";dashed=1;endArrow=classic;fontSize=10;"

	// Global element lookup across all slices (needed for cross-slice references)
	// For multi-tag events, only the first representation is stored in nameToElem
	// so that standard connections (command->event, event->view, etc.) use a single target.
	nameToElem := make(map[string]*namedElem)
	for _, e := range elems {
		if _, exists := nameToElem[e.name]; !exists {
			nameToElem[e.name] = &e
		}
	}

	// The id is allocated only once the edge is certain to be drawn: hoisting it
	// would renumber every later edge of a model naming a view no slice declares.
	readsEdge := func(reads string, reader *namedElem) string {
		if reads == "" || reader == nil {
			return ""
		}
		view := nameToElem[reads]
		if view == nil {
			return ""
		}

		return edgeCell(allocID(), standardStyle, view.id, reader.id)
	}

	// Multi-tag connectors: link representations of the same event across tag lanes
	if len(multiTagEvents) > 0 {
		multiTagStyle := "edgeStyle=orthogonalEdgeStyle;html=1;fontFamily=Helvetica;strokeColor=#9B59B6;dashed=1;fontSize=10;endArrow=none;curved=1;"
		for _, mte := range multiTagEvents {
			for j := 1; j < len(mte.cellIDs); j++ {
				// Find the namedElem for each representation to compute waypoints
				var srcElem, tgtElem *namedElem
				for k := range elems {
					if elems[k].id == mte.cellIDs[0] {
						srcElem = &elems[k]
					}
					if elems[k].id == mte.cellIDs[j] {
						tgtElem = &elems[k]
					}
				}
				if srcElem != nil && tgtElem != nil {
					rightX := srcElem.x + srcElem.w + waypointMargin
					midY := srcElem.y + srcElem.h/2
					tgtMidY := tgtElem.y + tgtElem.h/2
					points := [][2]int{
						{rightX, midY},
						{rightX, tgtMidY},
					}
					b.WriteString(edgeCellWaypoints(allocID(), multiTagStyle, srcElem.id, tgtElem.id, points))
				}
			}
		}
	}

	for _, entry := range entries {
		s := entry.slice

		// trigger -> command (downward, standard)
		if s.Trigger != nil {
			tid := nameToElem[s.Trigger.Name]
			b.WriteString(readsEdge(s.Trigger.Reads, tid))
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
			e := nameToElem[auto.OnEvent]
			a := nameToElem[auto.Name]
			c := nameToElem[auto.Command]

			b.WriteString(readsEdge(auto.Reads, a))

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

			b.WriteString(readsEdge(tr.Reads, extE))

			// external system -> reactor (upward, dashed gray)
			b.WriteString(edgeCell(allocID(), extStyle, extE.id, rE.id))

			// reactor -> command (downward, standard)
			if tr.Command != "" {
				c := nameToElem[tr.Command]
				if c != nil {
					b.WriteString(edgeCell(allocID(), standardStyle, rE.id, c.id))
				}
			}

			// command -> event (translation implies command emits event, unless a
			// flow already declares it and has drawn the arrow above)
			if tr.Command != "" && tr.Event != nil && tr.Event.Name != "" &&
				!declaresFlow(s, tr.Command, tr.Event.Name) {
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
	slice          *ast.Slice
	ctxName        string
	ctxDescription string
	fromDCB        bool // true if slice comes from a direct (DCB) context
}

// collectSlices flattens all slices from the model into a list.
// It collects from both ctx.Aggregates[].Slices (traditional aggregate mode)
// and ctx.Slices (direct DCB mode), preserving order within each context.
func collectSlices(model *ast.Model) []sliceEntry {
	var entries []sliceEntry
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, s := range agg.Slices {
				entries = append(entries, sliceEntry{slice: s, ctxName: ctx.Name, ctxDescription: ctx.Description, fromDCB: false})
			}
		}
		for _, s := range ctx.Slices {
			entries = append(entries, sliceEntry{slice: s, ctxName: ctx.Name, ctxDescription: ctx.Description, fromDCB: true})
		}
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

// draw.io reads a tooltip only off an <object> wrapping the mxCell. A box with
// nothing to say stays a bare mxCell: wrapping those too would rewrite the
// diagram of every model that carries no prose.
func vertexCell(id int, value, tooltip string, x, y, w, h int, style string) string {
	geometry := fmt.Sprintf(`<mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry" />`, x, y, w, h)

	if tooltip == "" {
		return fmt.Sprintf(`        <mxCell id="%d" value="%s" style="%s" vertex="1" parent="1">`+"\n"+
			`          %s`+"\n"+
			`        </mxCell>`+"\n", id, escapeXML(value), style, geometry)
	}

	return fmt.Sprintf(`        <object label="%s" tooltip="%s" id="%d">`+"\n"+
		`          <mxCell style="%s" vertex="1" parent="1">`+"\n"+
		`            %s`+"\n"+
		`          </mxCell>`+"\n"+
		`        </object>`+"\n", escapeXML(value), escapeXML(tooltip), id, style, geometry)
}

// triggerFramingCell draws the screen framing that distinguishes a trigger box
// from a plain rectangle in draw.io: a small header bar inside the top edge of
// the parent trigger cell.
func triggerFramingCell(id, parentID, x, y, w, h int, stroke string) string {
	const headerMargin = 8
	const headerTop = 6
	const headerHeight = 6

	style := "rounded=0;whiteSpace=wrap;html=1;fillColor=" + stroke + ";strokeColor=" + stroke + ";"
	geometry := fmt.Sprintf(`<mxGeometry x="%d" y="%d" width="%d" height="%d" as="geometry" />`,
		x+headerMargin, y+headerTop, w-2*headerMargin, headerHeight)

	return fmt.Sprintf(`        <mxCell id="%d" value="" style="%s" vertex="1" parent="%d">`+"\n"+
		`          %s`+"\n"+
		`        </mxCell>`+"\n", id, style, parentID, geometry)
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

// formatDecidesOnAnnotation formats a command's decides_on clause for display.
func formatDecidesOnAnnotation(d *ast.DecidesOnClause) string {
	if d == nil || len(d.Events) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("decides_on: ")
	sb.WriteString(strings.Join(d.Events, ", "))
	if d.Predicate != nil {
		sb.WriteString(" where ")
		sb.WriteString(formatPredicateExpr(d.Predicate))
	}
	return sb.String()
}

// formatPredicateExpr formats a predicate expression for display.
func formatPredicateExpr(p ast.PredicateExpr) string {
	if p == nil {
		return ""
	}
	switch expr := p.(type) {
	case ast.TagPredicate:
		return fmt.Sprintf("tag(%s %s %s)", expr.Field, expr.Operator, expr.Value)
	case ast.LogicalExpr:
		return fmt.Sprintf("(%s %s %s)", formatPredicateExpr(expr.Left), expr.Operator, formatPredicateExpr(expr.Right))
	case ast.NotExpr:
		return fmt.Sprintf("not(%s)", formatPredicateExpr(expr.Expr))
	case ast.FieldRef:
		return fmt.Sprintf("tag(%s)", expr.Name)
	default:
		return ""
	}
}

// formatEventTagBadges formats event tags as badge indicators for display.
func formatEventTagBadges(tags []ast.TagEntry) string {
	if len(tags) == 0 {
		return ""
	}
	var parts []string
	for _, tag := range tags {
		if tag.FieldRef != "" {
			parts = append(parts, fmt.Sprintf("[%s: %s]", tag.Key, tag.FieldRef))
		} else {
			parts = append(parts, fmt.Sprintf("[%s]", tag.Key))
		}
	}
	return strings.Join(parts, " ")
}
