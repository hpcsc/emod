//go:build unit

package diagram_test

import (
	"encoding/xml"
	"fmt"
	"maps"
	"math"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// Every exporter turns the same AST into a different rendering of it. The
// behaviour they share — nothing is dropped, order is preserved, output is
// well-formed — belongs to all four, so it is asserted once here against each
// of them rather than copied into every format's file.
type exporter struct {
	name string
	// fillOfLabel returns the fill colour of the shape drawn for a label; nil
	// for the text formats, which have no colours.
	fillOfLabel func(t *testing.T, output, label string) string
	// strokeOfLabel returns the stroke colour of the shape drawn for a label; nil
	// for the text formats, which have no colours.
	strokeOfLabel func(t *testing.T, output, label string) string
	// countConnections reports how many connections the output draws; nil for
	// formats that describe elements without drawing arrows between them.
	countConnections func(output string) int
	// boxes returns the boxes the output draws, in document order; nil for the
	// text formats, which draw none.
	boxes func(t *testing.T, output string) []diagramBox
	// connections returns the arrows the output draws between its boxes, in
	// document order; nil for the text formats, which draw none.
	connections       func(t *testing.T, output string) []diagramConnection
	export            func(*ast.Model, diagram.Style) ([]byte, error)
	requireWellFormed func(t *testing.T, output string)
}

type diagramBox struct {
	label string
	// appearance is where the box sits, how big it is and how it is painted, in
	// the format's own spelling.
	appearance string
	rect       boxRect
}

// boxRect is where a box was drawn: the corner it starts at, and how far it runs
// from there.
type boxRect struct {
	x, y, w, h int
}

// centre is the point an arrow drawn to or from a box meets it at.
func (r boxRect) centre() [2]int {
	return [2]int{r.x + r.w/2, r.y + r.h/2}
}

func (r boxRect) overlaps(other boxRect) bool {
	return r.x < other.x+other.w && other.x < r.x+r.w &&
		r.y < other.y+other.h && other.y < r.y+r.h
}

// within reports whether the box is drawn inside container without touching any
// of its edges, which is how a container itself reads back as outside itself.
func (r boxRect) within(container boxRect) bool {
	return r.x > container.x && r.y > container.y &&
		r.x+r.w < container.x+container.w && r.y+r.h < container.y+container.h
}

// diagramConnection is an arrow the output draws, named by the boxes at its two
// ends rather than by where those boxes sit.
type diagramConnection struct {
	source string
	target string
	// paint is how the arrow is drawn, in the format's own spelling.
	paint string
}

func requireValidXML(t *testing.T, output string) {
	t.Helper()
	require.NoError(t, xml.Unmarshal([]byte(output), new(any)), "output must be well-formed XML")
}

func requireNonEmptyText(t *testing.T, output string) {
	t.Helper()
	require.NotEmpty(t, strings.TrimSpace(output), "output must not be blank")
}

func exporters() []exporter {
	return []exporter{
		{
			name:              "drawio",
			fillOfLabel:       drawioFillOfLabel,
			strokeOfLabel:     drawioStrokeOfLabel,
			countConnections:  func(output string) int { return strings.Count(output, `edge="1"`) },
			boxes:             drawioBoxes,
			connections:       drawioEdges,
			export:            diagram.ExportDrawio,
			requireWellFormed: requireValidXML,
		},
		{
			name:              "svg",
			fillOfLabel:       svgFillOfLabel,
			strokeOfLabel:     svgStrokeOfLabel,
			countConnections:  arrowCount,
			boxes:             svgBoxes,
			connections:       svgConnections,
			export:            diagram.ExportSVG,
			requireWellFormed: requireValidXML,
		},
		{
			name:              "mermaid",
			export:            diagram.ExportMermaid,
			requireWellFormed: requireNonEmptyText,
		},
		{
			name:              "ascii",
			countConnections:  func(output string) int { return strings.Count(output, " -> ") },
			export:            diagram.ExportASCII,
			requireWellFormed: requireNonEmptyText,
		},
	}
}

var (
	drawioStyleFill   = regexp.MustCompile(`fillColor=(#[0-9a-fA-F]{6})`)
	drawioStyleStroke = regexp.MustCompile(`strokeColor=(#[0-9a-fA-F]{6})`)
	svgRectFill       = regexp.MustCompile(`<rect[^>]*\bfill="(#[0-9a-fA-F]{6})"`)
	svgRectStroke     = regexp.MustCompile(`<rect[^>]*\bstroke="(#[0-9a-fA-F]{6})"`)
)

// drawioFillOfLabel returns the fill of the shape whose label contains label.
func drawioFillOfLabel(t *testing.T, output, label string) string {
	t.Helper()

	for _, shape := range drawioShapes(t, output) {
		if !strings.Contains(shape.label, label) {
			continue
		}
		if m := drawioStyleFill.FindStringSubmatch(shape.style); m != nil {
			return strings.ToLower(m[1])
		}
	}

	require.Fail(t, fmt.Sprintf("no drawio cell labelled %q in output", label))
	return ""
}

// drawioStrokeOfLabel returns the stroke of the shape whose label contains label.
func drawioStrokeOfLabel(t *testing.T, output, label string) string {
	t.Helper()

	for _, shape := range drawioShapes(t, output) {
		if !strings.Contains(shape.label, label) {
			continue
		}
		if m := drawioStyleStroke.FindStringSubmatch(shape.style); m != nil {
			return strings.ToLower(m[1])
		}
	}

	require.Fail(t, fmt.Sprintf("no drawio cell labelled %q in output", label))
	return ""
}

// svgFillOfLabel returns the fill of the rect drawn immediately before the text
// element carrying label — the shape that label sits inside.
func svgFillOfLabel(t *testing.T, output, label string) string {
	t.Helper()

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "<text") || !strings.Contains(line, label) {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			if m := svgRectFill.FindStringSubmatch(lines[j]); m != nil {
				return strings.ToLower(m[1])
			}
		}
	}

	require.Fail(t, fmt.Sprintf("no svg shape labelled %q in output", label))
	return ""
}

// svgStrokeOfLabel returns the stroke of the rect drawn immediately before the text
// element carrying label — the shape that label sits inside.
func svgStrokeOfLabel(t *testing.T, output, label string) string {
	t.Helper()

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "<text") || !strings.Contains(line, label) {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			if m := svgRectStroke.FindStringSubmatch(lines[j]); m != nil {
				return strings.ToLower(m[1])
			}
		}
	}

	require.Fail(t, fmt.Sprintf("no svg shape labelled %q in output", label))
	return ""
}

func (e exporter) run(t *testing.T, model *ast.Model, style diagram.Style) string {
	t.Helper()

	raw, err := e.export(model, style)
	require.NoError(t, err)

	return string(raw)
}

func (e exporter) boxLabelled(t *testing.T, output, name string) diagramBox {
	t.Helper()

	return boxLabelled(t, e.boxes(t, output), name)
}

func boxLabelled(t *testing.T, boxes []diagramBox, name string) diagramBox {
	t.Helper()

	var matched []diagramBox
	for _, box := range boxes {
		if strings.Contains(box.label, name) {
			matched = append(matched, box)
		}
	}
	require.Len(t, matched, 1, "expected one box labelled %q", name)

	return matched[0]
}

// rectsLabelled returns where the box each name reaches was drawn.
func rectsLabelled(t *testing.T, boxes []diagramBox, names []string) map[string]boxRect {
	t.Helper()

	rects := make(map[string]boxRect, len(names))
	for _, name := range names {
		rects[name] = boxLabelled(t, boxes, name).rect
	}

	return rects
}

// boxesDrawnOver names, for each of the named boxes, every box the picture draws
// it on top of, so a failure says which two boxes collided.
func boxesDrawnOver(rects map[string]boxRect, names []string) []string {
	drawn := slices.Sorted(maps.Keys(rects))

	var collisions []string
	for _, name := range names {
		for _, other := range drawn {
			if other == name || !rects[name].overlaps(rects[other]) {
				continue
			}
			collisions = append(collisions, name+" over "+other)
		}
	}

	return collisions
}

// gearedBoxes returns the boxes drawn for a model's automations and translation
// reactors, which the picture tells apart from every other box by the gear.
func gearedBoxes(boxes []diagramBox) []diagramBox {
	var geared []diagramBox
	for _, box := range boxes {
		if strings.Contains(box.label, gearMarking) {
			geared = append(geared, box)
		}
	}

	return geared
}

func labelsOf(boxes []diagramBox) []string {
	var labels []string
	for _, box := range boxes {
		labels = append(labels, box.label)
	}

	return labels
}

// labelsWithin names the boxes drawn wholly inside container. Decorative boxes
// with no label are skipped, because a framing element is not a box a reader
// names.
func labelsWithin(boxes []diagramBox, container boxRect) []string {
	var labels []string
	for _, box := range boxes {
		if box.label == "" || !box.rect.within(container) {
			continue
		}
		labels = append(labels, box.label)
	}

	return labels
}

// labelsBelow names the boxes drawn no higher than y.
func labelsBelow(boxes []diagramBox, y int) []string {
	var labels []string
	for _, box := range boxes {
		if box.rect.y >= y {
			labels = append(labels, box.label)
		}
	}

	return labels
}

// lowestEdge is the bottom of the lowest of the given boxes.
func lowestEdge(rects map[string]boxRect) int {
	var bottom int
	for _, rect := range rects {
		bottom = max(bottom, rect.y+rect.h)
	}

	return bottom
}

// edgesTouching returns the arrows the picture draws with one of the named boxes
// at either end, as "source -> target".
func edgesTouching(connections []diagramConnection, names []string) []string {
	var touching []string
	for _, connection := range connections {
		if slices.Contains(names, connection.source) || slices.Contains(names, connection.target) {
			touching = append(touching, connection.source+" -> "+connection.target)
		}
	}

	return touching
}

func TestExporterContract(t *testing.T) {
	for _, e := range exporters() {
		t.Run(e.name, func(t *testing.T) {
			t.Run("nil model produces no output and no error", func(t *testing.T) {
				raw, err := e.export(nil, diagram.StyleAuto)

				require.NoError(t, err)
				require.Empty(t, raw)
			})

			t.Run("a model with no contexts still produces well-formed output", func(t *testing.T) {
				output := e.run(t, &ast.Model{Name: "Empty"}, diagram.StyleAuto)

				e.requireWellFormed(t, output)
			})

			t.Run("every named element in the model appears in the output", func(t *testing.T) {
				output := e.run(t, fullModel(), diagram.StyleAuto)

				e.requireWellFormed(t, output)
				for _, label := range fullModelLabels {
					require.Contains(t, output, label)
				}
			})

			t.Run("slices are rendered in model order", func(t *testing.T) {
				model := &ast.Model{
					Name: "Test",
					Contexts: []*ast.Context{{
						Name: "Ctx",
						Aggregates: []*ast.Aggregate{{
							Name: "Agg",
							Slices: []*ast.Slice{
								{Name: "First", Commands: []*ast.Command{{Name: "CmdA"}}},
								{Name: "Second", Commands: []*ast.Command{{Name: "CmdB"}}},
								{Name: "Third", Commands: []*ast.Command{{Name: "CmdC"}}},
							},
						}},
					}},
				}

				output := e.run(t, model, diagram.StyleAuto)

				require.Equal(t, []string{"CmdA", "CmdB", "CmdC"}, appearanceOrder(t, output, "CmdA", "CmdB", "CmdC"))
			})

			t.Run("slices declared directly under a DCB context are rendered", func(t *testing.T) {
				model := &ast.Model{
					Name: "DCBTest",
					Contexts: []*ast.Context{{
						Name:   "DCBCtx",
						Slices: []*ast.Slice{{Name: "DirectSlice", Commands: []*ast.Command{{Name: "DirectCmd"}}}},
					}},
				}

				output := e.run(t, model, diagram.StyleDCB)

				e.requireWellFormed(t, output)
				require.Contains(t, output, "DirectCmd")
			})

			t.Run("an aggregate-only context renders under the DCB style too", func(t *testing.T) {
				output := e.run(t, singleSliceModel("Test", "S", command("Cmd")), diagram.StyleDCB)

				e.requireWellFormed(t, output)
				require.Contains(t, output, "Cmd")
			})

			t.Run("naming fields after keywords leaves the picture untouched", func(t *testing.T) {
				keywordNamed := test.KeywordFieldSearchCatalogModel(t)
				ordinaryNamed := test.WithOrdinaryFieldNames(test.KeywordFieldSearchCatalogModel(t))

				require.NotEqual(t, keywordNamed, ordinaryNamed,
					"the twin has to be named differently, or the comparison below says nothing")
				require.Equal(t, e.run(t, ordinaryNamed, diagram.StyleAuto), e.run(t, keywordNamed, diagram.StyleAuto),
					"a diagram shows elements and arrows, not field lists, so renaming every field cannot move it")
			})

			t.Run("declaring invariants leaves the picture untouched", func(t *testing.T) {
				constrained := test.InvariantLibraryLendingModel(t)
				unconstrained := withoutInvariants(constrained)

				require.NotEqual(t, constrained, unconstrained,
					"the twin has to declare no invariants, or the comparison below says nothing")
				require.Equal(t, libraryLendingInvariantNames, invariantNamesOf(constrained))
				require.Empty(t, invariantNamesOf(unconstrained),
					"the twin has to lose the invariants of both homes, or the comparison below is answered by whichever home it kept")

				require.Equal(t, e.run(t, unconstrained, diagram.StyleAuto), e.run(t, constrained, diagram.StyleAuto),
					"a diagram shows elements and arrows, not the rules they keep, so declaring an invariant cannot move it")
			})

			t.Run("stating specs leaves the picture untouched", func(t *testing.T) {
				stated := test.SpecLibraryLendingModel(t)
				unstated := test.WithoutSpecs(stated)

				require.NotEqual(t, stated, unstated,
					"the twin has to state no spec, or the comparison below says nothing")
				require.Equal(t, test.SpecLibraryLendingSpecNames, test.DeclaredSpecNames(stated))
				require.Empty(t, test.DeclaredSpecNames(unstated),
					"the twin has to lose the specs of both slice homes, or the comparison below is answered by whichever home it kept")

				require.Equal(t, e.run(t, unstated, diagram.StyleAuto), e.run(t, stated, diagram.StyleAuto),
					"a diagram shows elements and arrows, not the scenarios they must satisfy, so stating a spec cannot move it")
			})
		})
	}
}

// TestExporterTranslationEdges covers the arrow a translation implies between
// its command and its nested event. Drawing it unconditionally duplicated any
// flow that already declared the same pair, which put two identical arrows
// between the same two boxes.
func TestExporterTranslationEdges(t *testing.T) {
	for _, e := range exporters() {
		if e.countConnections == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			t.Run("draws the implied command to event arrow when no flow declares it", func(t *testing.T) {
				output := e.run(t, translationModel(false), diagram.StyleAuto)

				// external system -> reactor, reactor -> command, command -> event
				require.Equal(t, 3, e.countConnections(output))
			})

			t.Run("draws it once when a flow declares the same pair", func(t *testing.T) {
				implied := e.run(t, translationModel(false), diagram.StyleAuto)
				declared := e.run(t, translationModel(true), diagram.StyleAuto)

				require.Equal(t, e.countConnections(implied), e.countConnections(declared),
					"spelling the flow out must not add a second arrow for the same pair")
			})

			t.Run("still draws a flow that the translation does not imply", func(t *testing.T) {
				model := translationModel(false)
				slice := model.Contexts[0].Aggregates[0].Slices[0]
				slice.Commands = append(slice.Commands, command("Refund"))
				slice.Events = append(slice.Events, event("Refunded"))
				slice.Flows = []*ast.Flow{{CommandName: "Refund", EventName: "Refunded"}}

				output := e.run(t, model, diagram.StyleAuto)

				require.Equal(t, 4, e.countConnections(output))
			})
		})
	}
}

// translationModel holds a translation whose command emits its nested event.
// With declareFlow set, the model also states that connection as a flow — the
// same fact said twice.
func translationModel(declareFlow bool) *ast.Model {
	model := singleSliceModel("Payments", "Take Payment", command("Charge"), event("Charged"))
	slice := model.Contexts[0].Aggregates[0].Slices[0]
	slice.Translations = []*ast.Translation{{
		Name:           "StripeWebhook",
		ExternalSystem: "Stripe",
		Command:        "Charge",
		Event:          &ast.Event{Name: "Charged"},
	}}
	if declareFlow {
		slice.Flows = []*ast.Flow{{CommandName: "Charge", EventName: "Charged"}}
	}
	return model
}

// readsLibraryLendingViews names every view test.TriggerReadsLibraryLending
// declares, so an arrow into a trigger or an automation can be told apart from
// the one an automation is also drawn from the event that activates it.
var readsLibraryLendingViews = []string{"MemberLoansView", "DeskOccupancyView"}

// triggerReadsLibraryLendingTriggers names every trigger
// test.TriggerReadsLibraryLending declares, both slice homes together and in
// declaration order, so the views read back off the picture line up with
// test.TriggerReadsLibraryLendingTriggerViewNames and the trigger that reads
// nothing contributes nothing.
var triggerReadsLibraryLendingTriggers = []string{
	"Lending Desk",
	"Returns Counter",
	"Overdue Report",
	"Desk Kiosk",
}

// triggerReadsLibraryLendingAutomations does the same for that fixture's
// automations, lining up with
// test.TriggerReadsLibraryLendingAutomationViewNames.
var triggerReadsLibraryLendingAutomations = []string{
	"RemindOnDueDate",
	"RecallOverdueCopy",
	"FreeDeskAtClosing",
	"RemindReaderOfLoans",
}

// TestExporterReadsEdges covers the arrow a trigger and an automation are drawn
// from the view they read. Only the formats that draw boxes have an arrow for a
// reads to become, and the text formats are here to show the picture they draw
// does not move.
func TestExporterReadsEdges(t *testing.T) {
	for _, e := range exporters() {
		t.Run(e.name, func(t *testing.T) {
			if e.connections == nil {
				t.Run("renders a model whose triggers and automations read views exactly as one whose read none", func(t *testing.T) {
					reading := test.TriggerReadsLibraryLendingModel(t)
					unreadTriggers := test.WithoutTriggerReads(reading)
					unreadAutomations := test.WithoutAutomationReads(reading)

					require.Equal(t, test.TriggerReadsLibraryLendingTriggerViewNames, test.DeclaredTriggerReads(reading))
					require.Equal(t, test.TriggerReadsLibraryLendingAutomationViewNames, test.DeclaredAutomationReads(reading))
					require.Empty(t, test.DeclaredTriggerReads(unreadTriggers),
						"the twin has to lose the reads of both slice homes, or the comparison below is answered by whichever home it kept")
					require.Empty(t, test.DeclaredAutomationReads(unreadAutomations),
						"the twin has to lose the reads of both slice homes, or the comparison below is answered by whichever home it kept")

					output := e.run(t, reading, diagram.StyleAuto)

					require.Equal(t, output, e.run(t, unreadTriggers, diagram.StyleAuto),
						"a format that draws no boxes has no arrow for the view a trigger reads to become")
					require.Equal(t, output, e.run(t, unreadAutomations, diagram.StyleAuto),
						"a format that draws no boxes has no arrow for the view an automation reads to become")
				})

				return
			}

			for _, twin := range readsTwins() {
				t.Run("the "+twin.construct+" reading a declared view is drawn an arrow from that view", func(t *testing.T) {
					reading := readsModel()
					unread := twin.strip(reading)

					require.Equal(t, []string{"MemberLoansView"}, twin.declared(reading))
					require.Empty(t, twin.declared(unread),
						"the twin has to read no view, or the differential below says nothing")

					full, stripped := e.run(t, reading, diagram.StyleAuto), e.run(t, unread, diagram.StyleAuto)
					drawn := e.connections(t, full)

					e.requireWellFormed(t, full)
					require.ElementsMatch(t, append(e.connections(t, stripped), diagramConnection{
						source: e.boxLabelled(t, full, "MemberLoansView").label,
						target: e.boxLabelled(t, full, twin.reader).label,
						paint:  paintOfArrow(t, drawn, "MemberLoansView", "LoanRegistry"),
					}), drawn,
						"the picture gains one arrow, from the view to the "+twin.construct+", painted as the arrow to the view the translation reads already was, and keeps every arrow it drew before")
					require.NotContains(t, paintOfArrow(t, drawn, "MemberLoansView", twin.reader), "dash",
						"an arrow to a reader is not drawn with the dashed treatment an external system gets")
				})
			}

			t.Run("a reads naming a view no slice declares, and a reads left out, are drawn no arrow", func(t *testing.T) {
				output := e.run(t, mixedReadsModel(), diagram.StyleAuto)

				e.requireWellFormed(t, output)
				drawn := e.connections(t, output)

				require.Equal(t, map[string][]string{
					"Desk Kiosk":        {"DeskOccupancyView"},
					"FreeDeskAtClosing": {"DeskOccupancyView"},
					"Booking Page":      nil,
					"ExpireReservation": nil,
					"Closing Bell":      nil,
					"SweepIdleDesks":    nil,
				}, sourcesIntoEach(drawn, []string{
					"Desk Kiosk", "FreeDeskAtClosing",
					"Booking Page", "ExpireReservation",
					"Closing Bell", "SweepIdleDesks",
				}))
				require.Len(t, drawn, 2,
					"the picture draws an arrow for the two reads that name a declared view and for nothing else")
			})

			t.Run("draws one arrow per view the fixture's triggers and its automations read", func(t *testing.T) {
				output := e.run(t, test.TriggerReadsLibraryLendingModel(t), diagram.StyleAuto)

				e.requireWellFormed(t, output)
				drawn := e.connections(t, output)

				require.Equal(t, test.TriggerReadsLibraryLendingTriggerViewNames,
					viewsRead(drawn, triggerReadsLibraryLendingTriggers, readsLibraryLendingViews))
				require.Equal(t, test.TriggerReadsLibraryLendingAutomationViewNames,
					viewsRead(drawn, triggerReadsLibraryLendingAutomations, readsLibraryLendingViews),
					"the automation reading a view another context declares is drawn its arrow too")
			})

			for _, twin := range readsTwins() {
				t.Run("clearing every "+twin.construct+"'s reads draws the same boxes with one arrow fewer per view", func(t *testing.T) {
					reading := test.TriggerReadsLibraryLendingModel(t)
					unread := twin.strip(reading)

					require.Equal(t, twin.declaredViews, twin.declared(reading))
					require.Empty(t, twin.declared(unread),
						"the twin has to lose the reads of both slice homes, or the comparison below is answered by whichever home it kept")
					require.Equal(t, twin.keptViews, twin.kept(unread),
						"the twin has to keep what the other construct reads, or the comparison below is answered by the arrows that construct lost instead")

					full, stripped := e.run(t, reading, diagram.StyleAuto), e.run(t, unread, diagram.StyleAuto)

					require.Equal(t, e.boxes(t, stripped), e.boxes(t, full),
						"an arrow to a reader must not add, move, resize or repaint a box")
					require.Len(t, e.connections(t, stripped), len(e.connections(t, full))-len(twin.declaredViews),
						"the twin loses one arrow per view it stopped reading and disturbs no other")
				})
			}

			t.Run("a trigger whose reads names a view no slice declares is drawn no arrow beside an automation whose does not", func(t *testing.T) {
				reading := test.AutomationReadsLibraryLendingModel(t)

				require.Equal(t, []string{"AvailableCopiesView"}, test.DeclaredTriggerReads(reading),
					"its trigger has to name a view no slice declares, or the silence below says nothing")
				require.Equal(t, test.AutomationReadsLibraryLendingViewNames, test.DeclaredAutomationReads(reading))

				output := e.run(t, reading, diagram.StyleAuto)

				e.requireWellFormed(t, output)
				drawn := e.connections(t, output)
				require.Empty(t, sourcesInto(drawn, "Lending Desk"))
				require.ElementsMatch(t, []string{"CopyBorrowed", "MemberLoansView"},
					sourcesInto(drawn, "RecallOverdueCopy"),
					"the automation beside it reads a view the model does declare, so its arrow is drawn")
			})
		})
	}
}

// readsTwin pairs a construct that reads a view with the twin that reads none —
// of readsModel, where the box it labels is the only one to gain an arrow, and
// of test.TriggerReadsLibraryLending, where the other construct goes on reading
// what it read.
type readsTwin struct {
	construct string
	// reader labels the box readsModel draws for this construct.
	reader        string
	strip         func(*ast.Model) *ast.Model
	declared      func(*ast.Model) []string
	kept          func(*ast.Model) []string
	declaredViews []string
	keptViews     []string
}

func readsTwins() []readsTwin {
	return []readsTwin{
		{
			construct:     "trigger",
			reader:        "Lending Desk",
			strip:         test.WithoutTriggerReads,
			declared:      test.DeclaredTriggerReads,
			kept:          test.DeclaredAutomationReads,
			declaredViews: test.TriggerReadsLibraryLendingTriggerViewNames,
			keptViews:     test.TriggerReadsLibraryLendingAutomationViewNames,
		},
		{
			construct:     "automation",
			reader:        "RecallOverdueCopy",
			strip:         test.WithoutAutomationReads,
			declared:      test.DeclaredAutomationReads,
			kept:          test.DeclaredTriggerReads,
			declaredViews: test.TriggerReadsLibraryLendingAutomationViewNames,
			keptViews:     test.TriggerReadsLibraryLendingTriggerViewNames,
		},
	}
}

// readsModel gives a trigger and an automation the view another slice declares,
// beside a translation reading that same view, so the arrow each reader gains
// can be compared against the one the translation was already drawn.
func readsModel() *ast.Model {
	model := singleSliceModel("Lending", "Borrow Copy",
		&ast.Trigger{Name: "Lending Desk", Actor: "Member", Reads: "MemberLoansView"},
		command("BorrowCopy"), event("CopyBorrowed"))
	aggregate := model.Contexts[0].Aggregates[0]
	aggregate.Slices[0].Translations = []*ast.Translation{{
		Name:           "PartnerImport",
		ExternalSystem: "LoanRegistry",
		Reads:          "MemberLoansView",
		Command:        "BorrowCopy",
	}}
	aggregate.Slices = append(aggregate.Slices,
		&ast.Slice{
			Name:  "Review Member Loans",
			Views: []*ast.View{{Name: "MemberLoansView", Subscribes: []string{"CopyBorrowed"}}},
		},
		&ast.Slice{
			Name:     "Chase Overdue Copy",
			Commands: []*ast.Command{command("RecallCopy")},
			Automations: []*ast.Automation{{
				Name:    "RecallOverdueCopy",
				OnEvent: "CopyBorrowed",
				Reads:   "MemberLoansView",
				Command: "RecallCopy",
			}},
		})

	return model
}

// mixedReadsModel puts a trigger and an automation reading a declared view
// beside a trigger and an automation naming a view no slice declares, and a
// trigger and an automation reading nothing, so one picture shows the arrow and
// its absence together. None of the six is drawn any other arrow, so an arrow
// reaching one of them could only be the view it reads.
func mixedReadsModel() *ast.Model {
	model := singleSliceModel("Reading Room", "Browse Desk Occupancy", view("DeskOccupancyView"))
	aggregate := model.Contexts[0].Aggregates[0]
	aggregate.Slices = append(aggregate.Slices,
		&ast.Slice{
			Name:        "Claim Desk",
			Trigger:     &ast.Trigger{Name: "Desk Kiosk", Reads: "DeskOccupancyView"},
			Automations: []*ast.Automation{{Name: "FreeDeskAtClosing", Reads: "DeskOccupancyView"}},
		},
		&ast.Slice{
			Name:        "Reserve Desk",
			Trigger:     &ast.Trigger{Name: "Booking Page", Reads: "DeskWaitlistView"},
			Automations: []*ast.Automation{{Name: "ExpireReservation", Reads: "DeskWaitlistView"}},
		},
		&ast.Slice{
			Name:        "Close Reading Room",
			Trigger:     &ast.Trigger{Name: "Closing Bell"},
			Automations: []*ast.Automation{{Name: "SweepIdleDesks"}},
		})

	return model
}

// sourcesInto names every box the picture draws an arrow from into the box
// labelled name.
func sourcesInto(connections []diagramConnection, name string) []string {
	var sources []string
	for _, connection := range connections {
		if strings.Contains(connection.target, name) {
			sources = append(sources, connection.source)
		}
	}

	return sources
}

func sourcesIntoEach(connections []diagramConnection, names []string) map[string][]string {
	sources := make(map[string][]string, len(names))
	for _, name := range names {
		sources[name] = sourcesInto(connections, name)
	}

	return sources
}

// viewsRead names the views the picture draws an arrow from into each of the
// named boxes, in the order the boxes are named. An automation is also drawn an
// arrow from the event that activates it, so only an arrow out of one of views
// counts as a read.
func viewsRead(connections []diagramConnection, names, views []string) []string {
	var read []string
	for _, name := range names {
		for _, source := range sourcesInto(connections, name) {
			if slices.Contains(views, source) {
				read = append(read, source)
			}
		}
	}

	return read
}

// paintOfArrow returns how the one arrow between the two named boxes is painted.
func paintOfArrow(t *testing.T, connections []diagramConnection, source, target string) string {
	t.Helper()

	var painted []string
	for _, connection := range connections {
		if strings.Contains(connection.source, source) && strings.Contains(connection.target, target) {
			painted = append(painted, connection.paint)
		}
	}
	require.Len(t, painted, 1, "expected one arrow from %q to %q", source, target)

	return painted[0]
}

const (
	gearMarking  = "⚙"
	clockMarking = "⏱"
)

// scheduledLibraryLendingAutomations names the automations of
// test.AutomationScheduleLibraryLending that run on a cadence, both slice homes
// together and in declaration order, so each name lines up with the cadence
// transcribed at the same place in
// test.AutomationScheduleLibraryLendingSchedules.
var scheduledLibraryLendingAutomations = []string{
	"RemindMemberEachMorning",
	"SweepOverdueLoans",
	"CloseDesksAtNight",
	"SweepIdleDesks",
}

// eventActivatedLibraryLendingAutomations names the automations of that same
// fixture that state an activation event instead of a cadence.
var eventActivatedLibraryLendingAutomations = []string{
	"RecallOnSecondReminder",
	"RemindReaderOfLoans",
}

// TestExporterAutomationSchedule covers the cadence a scheduled automation's
// box shows. The text formats draw no box to show one on, so they sit it out.
func TestExporterAutomationSchedule(t *testing.T) {
	for _, e := range exporters() {
		if e.boxes == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			t.Run("a scheduled automation is drawn with its own cadence beside the gear and its name", func(t *testing.T) {
				output := e.run(t, test.AutomationScheduleLibraryLendingModel(t), diagram.StyleAuto)

				e.requireWellFormed(t, output)
				var shown []string
				for _, name := range scheduledLibraryLendingAutomations {
					label := e.boxLabelled(t, output, name).label
					require.Contains(t, label, gearMarking, "a cadence joins the marking the box already carried")
					shown = append(shown, scheduleShown(label))
				}

				require.Equal(t, test.AutomationScheduleLibraryLendingSchedules, shown,
					"each box shows the cadence its own automation runs on")
			})

			t.Run("an automation stating an activation event is drawn with no clock and no cadence", func(t *testing.T) {
				mixed := e.run(t, test.AutomationScheduleLibraryLendingModel(t), diagram.StyleAuto)

				for _, name := range scheduledLibraryLendingAutomations {
					require.Contains(t, e.boxLabelled(t, mixed, name).label, clockMarking,
						"the scheduled automations of this same rendering have to be marked, or the silence below says nothing")
				}

				for _, name := range eventActivatedLibraryLendingAutomations {
					label := e.boxLabelled(t, mixed, name).label

					require.NotContains(t, label, clockMarking)
					for _, cadence := range test.AutomationScheduleLibraryLendingSchedules {
						require.NotContains(t, label, cadence, "the cadence of a neighbouring automation must not leak onto this box")
					}
				}

				unscheduled := e.run(t, test.AutomationReadsLibraryLendingModel(t), diagram.StyleAuto)

				require.NotContains(t, unscheduled, clockMarking,
					"a model whose automations state no cadence is drawn exactly as it was before cadences could be drawn")
			})

			t.Run("adding a schedule to one automation moves, resizes and repaints no box", func(t *testing.T) {
				const sweep = "SweepOverdueCopies"
				plain := e.run(t, sweepingModel(&ast.Automation{Name: sweep, OnEvent: "CopyBorrowed", Command: "RecallCopy"}), diagram.StyleAuto)
				scheduled := e.run(t, sweepingModel(&ast.Automation{Name: sweep, Schedule: "15m", Command: "RecallCopy"}), diagram.StyleAuto)

				require.Equal(t, "15m", scheduleShown(e.boxLabelled(t, scheduled, sweep).label),
					"the twin has to be drawn with the cadence, or the comparison below says nothing")
				require.Empty(t, scheduleShown(e.boxLabelled(t, plain, sweep).label))

				plainBoxes, scheduledBoxes := e.boxes(t, plain), e.boxes(t, scheduled)
				require.Equal(t, appearancesOf(plainBoxes), appearancesOf(scheduledBoxes),
					"a cadence must not move, resize or repaint a box")
				require.Equal(t, labelsExcept(plainBoxes, sweep), labelsExcept(scheduledBoxes, sweep),
					"a cadence belongs to the box of the automation running on it and to no other")
			})
		})
	}
}

// reactiveModelReactors labels the box drawn for each automation and each
// translation reactor reactiveModel declares, both slices together and in the
// order the picture draws them.
var reactiveModelReactors = []string{
	gearMarking + " NotifyWarehouse",
	gearMarking + " ChargeCard",
	gearMarking + " PaymentGateway",
	gearMarking + " BookPickup",
	gearMarking + " ArchiveOrder",
	gearMarking + " LabelFeed",
}

// reactiveModelAutomations labels the boxes of its automations alone — what a
// twin of it declaring no translation still draws.
var reactiveModelAutomations = []string{
	gearMarking + " NotifyWarehouse",
	gearMarking + " ChargeCard",
	gearMarking + " BookPickup",
	gearMarking + " ArchiveOrder",
}

// reactiveModelCommands labels the boxes on the row the reactive boxes have to
// find room beside.
var reactiveModelCommands = []string{
	"PlaceOrder", "ReserveStock", "OrderStatusView",
	"ShipOrder", "PrintLabel", "ShipmentBoardView",
}

// reactiveModelElements labels every box reactiveModel's own elements are drawn,
// leaving out the lanes and context bands they sit in.
var reactiveModelElements = slices.Concat(reactiveModelCommands, reactiveModelReactors, []string{
	"Order Form (Customer)", "OrderPlaced", "PaymentSettled", "Stripe",
	"Dispatch Desk", "OrderShipped", "LabelPrinted", "DHL",
})

// reactiveModelReactorEdges names every arrow reactiveModel's picture draws with
// an automation or a translation reactor at either end: the event that activates
// each automation, the view one of them reads, the command each issues, and the
// external system each translation restates.
var reactiveModelReactorEdges = []string{
	"OrderPlaced -> " + gearMarking + " NotifyWarehouse",
	gearMarking + " NotifyWarehouse -> ReserveStock",
	"OrderPlaced -> " + gearMarking + " ChargeCard",
	"OrderStatusView -> " + gearMarking + " ChargeCard",
	gearMarking + " ChargeCard -> PlaceOrder",
	"Stripe -> " + gearMarking + " PaymentGateway",
	gearMarking + " PaymentGateway -> PlaceOrder",
	"OrderShipped -> " + gearMarking + " BookPickup",
	gearMarking + " BookPickup -> ShipOrder",
	"OrderShipped -> " + gearMarking + " ArchiveOrder",
	gearMarking + " ArchiveOrder -> PrintLabel",
	"DHL -> " + gearMarking + " LabelFeed",
	gearMarking + " LabelFeed -> PrintLabel",
}

// reactiveModel gives two adjacent slices everything the command and view lane
// has to hold at once — a trigger, two commands, a view, an event, two
// automations and a translation each — so the reactive boxes have to find room
// beside the commands of their own slice and stay out of the neighbouring one.
func reactiveModel() *ast.Model {
	return &ast.Model{
		Name: "Orders",
		Contexts: []*ast.Context{{
			Name: "Orders",
			Aggregates: []*ast.Aggregate{{
				Name: "Order",
				Slices: []*ast.Slice{
					{
						Name:     "Place Order",
						Trigger:  &ast.Trigger{Name: "Order Form", Actor: "Customer"},
						Commands: []*ast.Command{command("PlaceOrder"), command("ReserveStock")},
						Events:   []*ast.Event{event("OrderPlaced")},
						Views:    []*ast.View{{Name: "OrderStatusView", Subscribes: []string{"OrderPlaced"}}},
						Flows:    []*ast.Flow{{CommandName: "PlaceOrder", EventName: "OrderPlaced"}},
						Automations: []*ast.Automation{
							{Name: "NotifyWarehouse", OnEvent: "OrderPlaced", Command: "ReserveStock"},
							{Name: "ChargeCard", OnEvent: "OrderPlaced", Reads: "OrderStatusView", Command: "PlaceOrder"},
						},
						Translations: []*ast.Translation{{
							Name:           "PaymentGateway",
							ExternalSystem: "Stripe",
							Command:        "PlaceOrder",
							Event:          &ast.Event{Name: "PaymentSettled"},
						}},
					},
					{
						Name:     "Ship Order",
						Trigger:  &ast.Trigger{Name: "Dispatch Desk"},
						Commands: []*ast.Command{command("ShipOrder"), command("PrintLabel")},
						Events:   []*ast.Event{event("OrderShipped")},
						Views:    []*ast.View{{Name: "ShipmentBoardView", Subscribes: []string{"OrderShipped"}}},
						Flows:    []*ast.Flow{{CommandName: "ShipOrder", EventName: "OrderShipped"}},
						Automations: []*ast.Automation{
							{Name: "BookPickup", OnEvent: "OrderShipped", Command: "ShipOrder"},
							{Name: "ArchiveOrder", OnEvent: "OrderShipped", Command: "PrintLabel"},
						},
						Translations: []*ast.Translation{{
							Name:           "LabelFeed",
							ExternalSystem: "DHL",
							Command:        "PrintLabel",
							Event:          &ast.Event{Name: "LabelPrinted"},
						}},
					},
				},
			}},
		}},
	}
}

// withoutTranslations strips every translation out of a model in place, taking
// the event and the external system each one names with it.
func withoutTranslations(model *ast.Model) *ast.Model {
	forEachSlice(model, func(s *ast.Slice) { s.Translations = nil })
	return model
}

// withoutAutomations strips every automation out of a model in place, leaving
// every other element where it was declared.
func withoutAutomations(model *ast.Model) *ast.Model {
	forEachSlice(model, func(s *ast.Slice) { s.Automations = nil })
	return model
}

// forEachSlice visits both homes a slice has — nested in an aggregate and
// declared directly on a context — so a strip reaching only one of them leaves
// the other home answering the comparison the strip was made for.
func forEachSlice(model *ast.Model, visit func(*ast.Slice)) {
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			for _, s := range agg.Slices {
				visit(s)
			}
		}
		for _, s := range ctx.Slices {
			visit(s)
		}
	}
}

// TestExporterReactorPlacement covers where the box for an automation and for a
// translation reactor is drawn. Both belong beside the commands and views they
// wire to rather than in the lane a person reads as the way into the system, and
// a lane only stays readable while no box is drawn over another. The text
// formats draw no boxes and sit it out.
func TestExporterReactorPlacement(t *testing.T) {
	for _, e := range exporters() {
		if e.boxes == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			t.Run("every automation and translation reactor is drawn in the command and view lane", func(t *testing.T) {
				output := e.run(t, reactiveModel(), diagram.StyleAuto)

				e.requireWellFormed(t, output)
				boxes := e.boxes(t, output)
				geared := gearedBoxes(boxes)
				lane := boxLabelled(t, boxes, "Commands / Views").rect

				require.Equal(t, reactiveModelReactors, labelsOf(geared),
					"every automation and every translation reactor is drawn one box, marked with the gear")
				require.Equal(t, reactiveModelReactors, labelsWithin(geared, lane),
					"each of those boxes lies wholly inside the command and view lane")
				require.Equal(t, reactiveModelReactors,
					labelsBelow(geared, lowestEdge(rectsLabelled(t, boxes, reactiveModelCommands))),
					"and below the commands and views it wires to, clear of the strip carrying the lane's name")
			})

			t.Run("no box is drawn over another", func(t *testing.T) {
				output := e.run(t, reactiveModel(), diagram.StyleAuto)

				e.requireWellFormed(t, output)
				rects := rectsLabelled(t, e.boxes(t, output), reactiveModelElements)

				require.Empty(t, boxesDrawnOver(rects, reactiveModelElements),
					"a box drawn under another is a label nobody can read")
			})

			t.Run("the top lane holds nothing but the triggers", func(t *testing.T) {
				output := e.run(t, reactiveModel(), diagram.StyleAuto)

				e.requireWellFormed(t, output)
				boxes := e.boxes(t, output)

				require.Equal(t, []string{"Order Form (Customer)", "Dispatch Desk"},
					labelsWithin(boxes, boxLabelled(t, boxes, "Wireframes").rect),
					"the lane holding what a person touches holds the two triggers and nothing else")
			})

			t.Run("every arrow into and out of a reactor still joins the same two boxes", func(t *testing.T) {
				output := e.run(t, reactiveModel(), diagram.StyleAuto)

				e.requireWellFormed(t, output)

				require.ElementsMatch(t, reactiveModelReactorEdges,
					edgesTouching(e.connections(t, output), reactiveModelReactors))
			})

			t.Run("declaring an automation leaves every box drawn around it where it was", func(t *testing.T) {
				featured := e.run(t, withoutTranslations(reactiveModel()), diagram.StyleAuto)
				stripped := e.run(t, withoutAutomations(withoutTranslations(reactiveModel())), diagram.StyleAuto)

				e.requireWellFormed(t, featured)
				e.requireWellFormed(t, stripped)
				featuredBoxes, strippedBoxes := e.boxes(t, featured), e.boxes(t, stripped)

				require.Equal(t, reactiveModelAutomations, labelsOf(gearedBoxes(featuredBoxes)),
					"the twin compared from has to draw every automation, or the comparison below is two identical pictures agreeing")
				require.Empty(t, gearedBoxes(strippedBoxes),
					"and the twin compared against has to draw none")

				require.Equal(t, strippedBoxes, boxesExcept(featuredBoxes, reactiveModelAutomations),
					"room for an automation must not be made by moving, resizing or repainting the boxes around it")
			})
		})
	}
}

func sweepingModel(sweep *ast.Automation) *ast.Model {
	return singleSliceModel("Lending", "Chase Overdue Copy",
		command("RemindMember"), command("RecallCopy"), event("CopyBorrowed"),
		&ast.Automation{Name: "RemindOnDueDate", OnEvent: "CopyBorrowed", Command: "RemindMember"},
		sweep)
}

func scheduleShown(label string) string {
	_, cadence, marked := strings.Cut(label, clockMarking)
	if !marked {
		return ""
	}

	return strings.TrimSpace(cadence)
}

func appearancesOf(boxes []diagramBox) []string {
	var appearances []string
	for _, box := range boxes {
		appearances = append(appearances, box.appearance)
	}
	return appearances
}

func labelsExcept(boxes []diagramBox, name string) []string {
	var labels []string
	for _, box := range boxes {
		if strings.Contains(box.label, name) {
			continue
		}
		labels = append(labels, box.label)
	}
	return labels
}

// boxesExcept returns the boxes the picture draws for everything but the named
// ones, in the order it drew them.
func boxesExcept(boxes []diagramBox, names []string) []diagramBox {
	var kept []diagramBox
	for _, box := range boxes {
		if slices.Contains(names, box.label) {
			continue
		}
		kept = append(kept, box)
	}

	return kept
}

// triggerScreenFraming returns the framing rectangle drawn inside a trigger box
// for the two formats that draw shapes, and a boolean that is true only when the
// box labelled label actually carries a screen framing. In SVG the framing is the
// label-less rect inside the main rect; in draw.io it is the child cell whose
// parent is the trigger cell.
func triggerScreenFraming(t *testing.T, e exporter, output, label string) (boxRect, bool) {
	t.Helper()

	switch e.name {
	case "svg":
		shapes := svgShapes(t, output)
		var main svgShape
		for _, shape := range shapes {
			if shape.label == label {
				main = shape
				break
			}
		}
		require.NotEmpty(t, main.label, "expected a trigger shape labelled %q", label)
		for _, shape := range shapes {
			if shape.label == "" && shape.rect.within(main.rect) {
				return shape.rect, true
			}
		}
		return boxRect{}, false

	case "drawio":
		shapes := drawioShapes(t, output)
		var trigger drawioShape
		for _, shape := range shapes {
			if shape.label == label {
				trigger = shape
				break
			}
		}
		require.NotEmpty(t, trigger.id, "expected a drawio cell labelled %q", label)
		for _, shape := range shapes {
			if shape.parentID == trigger.id && shape.rect.within(trigger.rect) {
				return shape.rect, true
			}
		}
		return boxRect{}, false

	default:
		require.Fail(t, "trigger screen framing not supported for %q", e.name)
	}

	return boxRect{}, false
}

// TestExporterTriggerScreen checks that a trigger is drawn as a screen rather
// than as a plain rounded rectangle, so it can be told from a sticky note without
// reading its colour.
func TestExporterTriggerScreen(t *testing.T) {
	for _, e := range exporters() {
		if e.fillOfLabel == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			t.Run("draws a screen framing on the trigger and no other element type", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				triggerRect := e.boxLabelled(t, output, "Form").rect
				framing, ok := triggerScreenFraming(t, e, output, "Form")
				require.True(t, ok, "expected the trigger to carry a screen framing")

				require.True(t, framing.within(triggerRect),
					"the trigger's framing must lie inside the trigger box")
				require.NotEmpty(t, framing.h,
					"the trigger's framing must have a positive height")

				for _, other := range []string{"Cmd", "Evt", "Rmo", "Stripe"} {
					_, ok := triggerScreenFraming(t, e, output, other)
					require.False(t, ok, "element type %q must not carry trigger screen framing", other)
				}
			})

			t.Run("keeps the trigger's box the same position and size", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				trigger := e.boxLabelled(t, output, "Form")
				require.Equal(t, boxRect{x: 60, y: 142, w: 240, h: 55}, trigger.rect,
					"the trigger box must keep the same position and size as it had without the framing")
			})

			t.Run("is still distinguishable when every fill is normalised to one colour", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)
				if e.name == "svg" {
					output = regexp.MustCompile(`fill="#[0-9a-fA-F]{6}"`).ReplaceAllString(output, `fill="#888888"`)
				} else {
					output = regexp.MustCompile(`fillColor=#[0-9a-fA-F]{6}`).ReplaceAllString(output, `fillColor=#888888`)
				}

				// Normalising must not break the structural distinction: the trigger
				// still has a framing and no other element type does.
				_, ok := triggerScreenFraming(t, e, output, "Form")
				require.True(t, ok, "the trigger must still have a framing after colour normalisation")
				for _, other := range []string{"Cmd", "Evt", "Rmo", "Stripe"} {
					_, ok := triggerScreenFraming(t, e, output, other)
					require.False(t, ok, "element type %q must not be mistaken for a trigger by framing", other)
				}
			})

			t.Run("keeps the trigger tooltip and fill on the trigger's own shape", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				if e.name == "svg" {
					require.Equal(t, "Where the order is placed", svgTooltipOf(t, output, "Form"))
				} else {
					require.Equal(t, "Where the order is placed", drawioTooltipOf(t, output, "Form"))
				}
				require.Equal(t, "#ffffff", e.fillOfLabel(t, output, "Form"))
			})
		})
	}
}

// viewerNodePalette parses the node palette declared in the viewer's config.js
// and returns, for each element type, the fill and stroke the viewer paints it
// with. The map keys are the node type names used in the viewer.
func viewerNodePalette(t *testing.T) map[string]struct{ fill, stroke string } {
	t.Helper()

	raw, err := os.ReadFile("../viewer/static/config.js")
	require.NoError(t, err)
	content := string(raw)

	require.Contains(t, content, "nodePalette", "the viewer must declare a nodePalette table")

	start := strings.Index(content, "nodePalette")
	require.NotEqual(t, -1, start, "nodePalette not found in config.js")
	block := content[start:]
	end := strings.Index(block, "};")
	require.NotEqual(t, -1, end, "nodePalette block does not end with }; in config.js")
	block = block[:end+2]

	entryRe := regexp.MustCompile(`(?m)^\s*(\w+):\s*\{\s*fill:\s*['"]?(#[0-9a-fA-F]{6})['"]?\s*,\s*stroke:\s*['"]?(#[0-9a-fA-F]{6})['"]?`)
	matches := entryRe.FindAllStringSubmatch(block, -1)
	require.Len(t, matches, 6, "nodePalette must list exactly six element types, got %d", len(matches))

	palette := make(map[string]struct{ fill, stroke string }, len(matches))
	for _, m := range matches {
		palette[m[1]] = struct{ fill, stroke string }{
			fill:   strings.ToLower(m[2]),
			stroke: strings.ToLower(m[3]),
		}
	}

	return palette
}

// viewerTypeToModelLabel maps the viewer's node type names to the labels
// paletteModel gives each element type.
var viewerTypeToModelLabel = map[string]string{
	"trigger":     "Form",
	"command":     "Cmd",
	"event":       "Evt",
	"view":        "Rmo",
	"automation":  "Auto",
	"translation": "Import",
}

// dslReferencePalette parses the diagram palette table from the DSL reference
// docs and returns a map from element type to the documented fill and stroke.
func dslReferencePalette(t *testing.T) map[string]struct{ fill, stroke string } {
	t.Helper()

	raw, err := os.ReadFile("../../docs/dsl-reference.md")
	require.NoError(t, err)
	content := string(raw)

	start := strings.Index(content, "## 13. Diagram Palette")
	require.NotEqual(t, -1, start, "docs/dsl-reference.md must contain a 'Diagram Palette' section")

	block := content[start:]
	end := strings.Index(block, "## 14.")
	if end == -1 {
		end = len(block)
	} else {
		end = strings.LastIndex(block[:end], "\n|")
		if end == -1 {
			end = len(block)
		}
	}
	block = block[:end]

	rowRe := regexp.MustCompile(`\|\s*(Trigger|Command|Event|View|Automation|Translation)\s*\|\s*(#[0-9a-fA-F]{6})\s*\|\s*(#[0-9a-fA-F]{6})\s*\|`)
	matches := rowRe.FindAllStringSubmatch(block, -1)
	require.Len(t, matches, 6, "palette table must list exactly six element types, got %d", len(matches))

	palette := make(map[string]struct{ fill, stroke string }, len(matches))
	for _, m := range matches {
		palette[strings.ToLower(m[1])] = struct{ fill, stroke string }{
			fill:   strings.ToLower(m[2]),
			stroke: strings.ToLower(m[3]),
		}
	}

	return palette
}

// TestExporterPalettePinsViewer checks that the two Go renderers emit the same
// fill and stroke per element type that the viewer's own palette table names.
func TestExporterPalettePinsViewer(t *testing.T) {
	viewerPalette := viewerNodePalette(t)

	for _, e := range exporters() {
		if e.fillOfLabel == nil || e.strokeOfLabel == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			output := e.run(t, paletteModel(), diagram.StyleAuto)

			for viewerType, modelLabel := range viewerTypeToModelLabel {
				viewerEntry, ok := viewerPalette[viewerType]
				require.True(t, ok, "viewer palette missing entry for %q", viewerType)

				require.Equal(t, viewerEntry.fill, e.fillOfLabel(t, output, modelLabel),
					"fill for %q disagrees between %s and the viewer palette", viewerType, e.name)
				require.Equal(t, viewerEntry.stroke, e.strokeOfLabel(t, output, modelLabel),
					"stroke for %q disagrees between %s and the viewer palette", viewerType, e.name)
			}
		})
	}
}

// TestExporterPaletteMatchesReference checks that the documented palette in the
// DSL reference is the same palette used by the viewer and the Go renderers.
func TestExporterPaletteMatchesReference(t *testing.T) {
	viewerPalette := viewerNodePalette(t)
	referencePalette := dslReferencePalette(t)

	require.Equal(t, viewerPalette, referencePalette,
		"viewer palette must match the palette documented in dsl-reference.md")

	for _, e := range exporters() {
		if e.fillOfLabel == nil || e.strokeOfLabel == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			output := e.run(t, paletteModel(), diagram.StyleAuto)

			for viewerType, modelLabel := range viewerTypeToModelLabel {
				refEntry, ok := referencePalette[viewerType]
				require.True(t, ok, "reference palette missing entry for %q", viewerType)

				require.Equal(t, refEntry.fill, e.fillOfLabel(t, output, modelLabel),
					"fill for %q disagrees between %s and dsl-reference.md", viewerType, e.name)
				require.Equal(t, refEntry.stroke, e.strokeOfLabel(t, output, modelLabel),
					"stroke for %q disagrees between %s and dsl-reference.md", viewerType, e.name)
			}
		})
	}
}

// TestExporterPalette pins the event-modeling sticky-note convention — orange
// events, blue commands, green read models, white triggers, purple automations,
// grey translations — without pinning the exact palette values, which are free to
// change.
func TestExporterPalette(t *testing.T) {
	for _, e := range exporters() {
		if e.fillOfLabel == nil {
			continue
		}

		t.Run(e.name, func(t *testing.T) {
			t.Run("follows the sticky-note colour convention", func(t *testing.T) {
				models := []struct {
					name  string
					model func() *ast.Model
				}{
					{name: "when every element is described", model: paletteModel},
					{name: "when the model describes nothing", model: func() *ast.Model {
						return withoutDescriptions(paletteModel())
					}},
				}

				for _, m := range models {
					t.Run(m.name, func(t *testing.T) {
						output := e.run(t, m.model(), diagram.StyleAuto)

						require.Equal(t, "orange", colorFamily(t, e.fillOfLabel(t, output, "Evt")))
						require.Equal(t, "blue", colorFamily(t, e.fillOfLabel(t, output, "Cmd")))
						require.Equal(t, "green", colorFamily(t, e.fillOfLabel(t, output, "Rmo")))
						require.Equal(t, "white", colorFamily(t, e.fillOfLabel(t, output, "Form")))
						require.Equal(t, "purple", colorFamily(t, e.fillOfLabel(t, output, "Auto")))
						require.Equal(t, "grey", colorFamily(t, e.fillOfLabel(t, output, "Import")))
						require.Equal(t, "grey", colorFamily(t, e.fillOfLabel(t, output, "Stripe")))
					})
				}
			})

			t.Run("gives each element type a distinguishable fill", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				fills := []string{
					e.fillOfLabel(t, output, "Evt"),
					e.fillOfLabel(t, output, "Cmd"),
					e.fillOfLabel(t, output, "Rmo"),
					e.fillOfLabel(t, output, "Form"),
					e.fillOfLabel(t, output, "Auto"),
					e.fillOfLabel(t, output, "Import"),
				}

				require.Len(t, unique(fills), len(fills), "each element type needs its own fill")
			})

			t.Run("gives each element type a distinguishable stroke", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				strokes := []string{
					e.strokeOfLabel(t, output, "Evt"),
					e.strokeOfLabel(t, output, "Cmd"),
					e.strokeOfLabel(t, output, "Rmo"),
					e.strokeOfLabel(t, output, "Form"),
					e.strokeOfLabel(t, output, "Auto"),
					e.strokeOfLabel(t, output, "Import"),
				}

				require.Len(t, unique(strokes), len(strokes), "each element type needs its own stroke")
			})

			t.Run("fills fall in six distinct colour families", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				families := []string{
					colorFamily(t, e.fillOfLabel(t, output, "Evt")),
					colorFamily(t, e.fillOfLabel(t, output, "Cmd")),
					colorFamily(t, e.fillOfLabel(t, output, "Rmo")),
					colorFamily(t, e.fillOfLabel(t, output, "Form")),
					colorFamily(t, e.fillOfLabel(t, output, "Auto")),
					colorFamily(t, e.fillOfLabel(t, output, "Import")),
				}

				require.Len(t, unique(families), len(families), "near-identical hues must not masquerade as distinct fills")
			})

			t.Run("automation and translation reactor fills differ", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				require.NotEqual(t,
					e.fillOfLabel(t, output, "Auto"),
					e.fillOfLabel(t, output, "Import"),
					"the automation and the translation reactor must not share a fill")
			})

			t.Run("draws an external system with a dashed outline while the translation reactor beside it is not", func(t *testing.T) {
				output := e.run(t, paletteModel(), diagram.StyleAuto)

				require.Contains(t, output, "dash", "an external system is drawn dashed")

				importBox := e.boxLabelled(t, output, "Import")
				require.NotContains(t, importBox.appearance, "dash",
					"the translation reactor is not the external system and keeps a solid outline")
			})
		})
	}
}

// paletteModel holds one element of every coloured kind, each with a distinct
// label so a test can ask for its fill. Every element is described, so asking
// for a shape by its label goes through whatever a format does with prose.
func paletteModel() *ast.Model {
	model := singleSliceModel("Palette", "S",
		&ast.Trigger{Name: "Form", Description: "Where the order is placed"},
		&ast.Command{Name: "Cmd", Description: "Asks for the order to be placed"},
		&ast.Event{Name: "Evt", Description: "The order was placed"},
		&ast.View{Name: "Rmo", Description: "Every order placed so far"},
		&ast.Automation{Name: "Auto", Description: "Reacts to the order being placed", OnEvent: "Evt", Command: "Cmd"},
	)
	model.Contexts[0].Aggregates[0].Slices[0].Translations = []*ast.Translation{{
		Name:           "Import",
		Description:    "Restates a payment provider callback in our own language",
		ExternalSystem: "Stripe",
		Command:        "Cmd",
		Event:          &ast.Event{Name: "Evt"},
	}}
	return model
}

// colorFamily classifies a hex colour by hue, so the convention is asserted
// rather than the specific shade.
func colorFamily(t *testing.T, hex string) string {
	t.Helper()

	hue, saturation, value := hsv(t, hex)
	switch {
	case saturation < 0.05 && value >= 0.99:
		return "white"
	case saturation < 0.05:
		return "grey"
	case hue >= 20 && hue < 50:
		return "orange"
	case hue >= 70 && hue < 170:
		return "green"
	case hue >= 170 && hue < 260:
		return "blue"
	case hue >= 260 && hue < 320:
		return "purple"
	default:
		return fmt.Sprintf("unclassified(hue %.0f)", hue)
	}
}

func hsv(t *testing.T, hex string) (hue, saturation, value float64) {
	t.Helper()
	require.Len(t, hex, 7, "expected a #rrggbb colour, got %q", hex)

	channel := func(offset int) float64 {
		v, err := strconv.ParseUint(hex[offset:offset+2], 16, 8)
		require.NoError(t, err, "parsing colour %q", hex)
		return float64(v) / 255
	}
	r, g, b := channel(1), channel(3), channel(5)

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	switch {
	case delta == 0:
		hue = 0
	case max == r:
		hue = 60 * math.Mod((g-b)/delta+6, 6)
	case max == g:
		hue = 60 * ((b-r)/delta + 2)
	default:
		hue = 60 * ((r-g)/delta + 4)
	}

	if max > 0 {
		saturation = delta / max
	}

	return hue, saturation, max
}

// appearanceOrder returns the given substrings sorted by where they first occur
// in output, failing if any is missing.
func appearanceOrder(t *testing.T, output string, substrings ...string) []string {
	t.Helper()

	ordered := make([]string, len(substrings))
	copy(ordered, substrings)

	positions := make(map[string]int, len(substrings))
	for _, s := range substrings {
		idx := strings.Index(output, s)
		require.NotEqual(t, -1, idx, "output does not contain %q", s)
		positions[s] = idx
	}

	sort.Slice(ordered, func(i, j int) bool { return positions[ordered[i]] < positions[ordered[j]] })

	return ordered
}

func unique(values []string) []string {
	seen := make(map[string]bool, len(values))
	var out []string
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// --- model builders shared by every exporter's tests ---

func minimalModel(name, sliceName string) *ast.Model {
	return &ast.Model{
		Name: name,
		Contexts: []*ast.Context{{
			Name: "Ctx",
			Aggregates: []*ast.Aggregate{{
				Name: "Agg",
				Slices: []*ast.Slice{{
					Name: sliceName,
				}},
			}},
		}},
	}
}

func singleSliceModel(modelName, sliceName string, opts ...any) *ast.Model {
	m := minimalModel(modelName, sliceName)
	s := m.Contexts[0].Aggregates[0].Slices[0]
	for _, opt := range opts {
		switch v := opt.(type) {
		case *ast.Command:
			s.Commands = append(s.Commands, v)
		case *ast.Event:
			s.Events = append(s.Events, v)
		case *ast.View:
			s.Views = append(s.Views, v)
		case *ast.Trigger:
			s.Trigger = v
		case *ast.Automation:
			s.Automations = append(s.Automations, v)
		}
	}
	return m
}

func command(name string) *ast.Command {
	return &ast.Command{Name: name}
}

func event(name string) *ast.Event {
	return &ast.Event{Name: name}
}

func eventWithSource(name, source, externalName string) *ast.Event {
	return &ast.Event{Name: name, Source: source, ExternalName: externalName}
}

func view(name string) *ast.View {
	return &ast.View{Name: name}
}

// fullModelLabels lists the element names every format renders. Structural
// names (model, context, slice) and external systems appear in some formats
// only, so those stay in the per-format tests.
var fullModelLabels = []string{
	"PlaceOrderForm", "ShipTimer",
	"CreateOrder", "ValidatePayment", "ShipOrder",
	"OrderCreated", "PaymentValidated", "OrderShipped",
	"OrderSummary",
	"InventoryUpdater",
	"PaymentGW",
}

// describedModel carries a distinct description on every construct that accepts
// one — including the aggregate and the slices, which no format draws a shape
// for — so a test can tell whose prose reached which shape. Its context mixes an
// aggregate with a directly declared slice, so each layout style has work to do.
func describedModel() *ast.Model {
	return &ast.Model{
		Name:        "Described",
		Description: "How the hotel takes and imports room bookings",
		Actors: []*ast.Actor{
			{Name: "Guest", Description: "A person booking a room, not always the one staying in it"},
		},
		Contexts: []*ast.Context{{
			Name:        "Bookings",
			Description: "Everything the hotel knows about a stay before the guest arrives",
			Aggregates: []*ast.Aggregate{{
				Name:        "Booking",
				Description: "One guest holding one room over one date range",
				Slices: []*ast.Slice{{
					Name:        "Hold a room",
					Description: "A guest books a room from the public site",
					Trigger: &ast.Trigger{
						Name:        "BookingForm",
						Actor:       "Guest",
						Description: "The booking form on the public site",
					},
					Commands: []*ast.Command{{
						Name:        "HoldRoom",
						Description: "Ask the hotel to hold a room over a date range",
					}},
					Events: []*ast.Event{{
						Name:        "RoomHeld",
						Description: "A room is held for a guest",
					}},
					Views: []*ast.View{{
						Name:        "StayList",
						Description: "Every booking with the stage it has reached",
						Subscribes:  []string{"RoomHeld"},
					}},
					Flows: []*ast.Flow{{CommandName: "HoldRoom", EventName: "RoomHeld"}},
					Automations: []*ast.Automation{{
						Name:        "AutoConfirm",
						Description: "Confirms every booking the moment it is made",
						OnEvent:     "RoomHeld",
						Command:     "HoldRoom",
					}},
					Translations: []*ast.Translation{{
						Name:           "PartnerWebhook",
						Description:    "Restates a partner webhook in the hotel's own language",
						ExternalSystem: "PartnerAPI",
						Command:        "HoldRoom",
						Event: &ast.Event{
							Name:        "PartnerBookingReceived",
							Description: "A partner site reported a booking",
						},
					}},
				}},
			}},
			Slices: []*ast.Slice{{
				Name:        "Settle the stay",
				Description: "The guest pays on the morning they leave",
				Events: []*ast.Event{{
					Name:        "StaySettled",
					Description: "The guest has paid for the whole stay",
					Tags:        []ast.TagEntry{{Key: "stay", FieldRef: "stayId"}},
				}},
			}},
		}},
	}
}

// describedModelTooltips pairs each shape describedModel draws, named by its
// label, with the prose that shape shows when hovered. An external system holds
// no prose of its own, so its box shows the description of the translation that
// names it, as the reactor box does.
var describedModelTooltips = map[string]string{
	"Bookings":               "Everything the hotel knows about a stay before the guest arrives",
	"BookingForm":            "The booking form on the public site",
	"HoldRoom":               "Ask the hotel to hold a room over a date range",
	"RoomHeld":               "A room is held for a guest",
	"StayList":               "Every booking with the stage it has reached",
	"StaySettled":            "The guest has paid for the whole stay",
	"AutoConfirm":            "Confirms every booking the moment it is made",
	"PartnerBookingReceived": "A partner site reported a booking",
	"PartnerWebhook":         "Restates a partner webhook in the hotel's own language",
	"PartnerAPI":             "Restates a partner webhook in the hotel's own language",
}

// requireEveryDescriptionShown asserts that every shape describedModel draws
// shows the description of the construct it was drawn for, read back through
// the format's own tooltipOf.
func requireEveryDescriptionShown(t *testing.T, output string, tooltipOf func(t *testing.T, output, label string) string) {
	t.Helper()

	shown := make(map[string]string, len(describedModelTooltips))
	for label := range describedModelTooltips {
		shown[label] = tooltipOf(t, output, label)
	}

	require.Equal(t, describedModelTooltips, shown)
}

// withoutDescriptions strips the prose out of a model in place, so a test can
// compare a described model's diagram against the one it draws without prose.
func withoutDescriptions(model *ast.Model) *ast.Model {
	model.Description = ""
	for _, actor := range model.Actors {
		actor.Description = ""
	}
	for _, ctx := range model.Contexts {
		ctx.Description = ""
		for _, agg := range ctx.Aggregates {
			agg.Description = ""
		}
	}
	forEachSlice(model, undescribeSlice)

	return model
}

func undescribeSlice(s *ast.Slice) {
	s.Description = ""
	if s.Trigger != nil {
		s.Trigger.Description = ""
	}
	for _, cmd := range s.Commands {
		cmd.Description = ""
	}
	for _, evt := range s.Events {
		evt.Description = ""
	}
	for _, v := range s.Views {
		v.Description = ""
	}
	for _, auto := range s.Automations {
		auto.Description = ""
	}
	for _, tr := range s.Translations {
		tr.Description = ""
		if tr.Event != nil {
			tr.Event.Description = ""
		}
	}
}

// libraryLendingInvariantNames transcribes every identifier
// test.InvariantLibraryLending declares, both homes together, so a walk that
// reaches only one of them reads back short.
var libraryLendingInvariantNames = []string{
	"OneCopyPerLoan",
	"FiveCopiesPerMember",
	"OneReaderPerDesk",
	"OneDeskPerReader",
	"DeskFreeAtClosing",
}

func invariantNamesOf(model *ast.Model) []string {
	var names []string
	for _, invariant := range declaredInvariants(model) {
		names = append(names, invariant.Name)
	}
	return names
}

func declaredInvariants(model *ast.Model) []*ast.Invariant {
	var invariants []*ast.Invariant
	for _, ctx := range model.Contexts {
		for _, agg := range ctx.Aggregates {
			invariants = append(invariants, agg.Invariants...)
		}
		invariants = append(invariants, ctx.Invariants...)
	}
	return invariants
}

// withoutInvariants returns a copy of model declaring no invariant in either
// home. Stripping in place would leave the caller comparing a model with
// itself.
func withoutInvariants(model *ast.Model) *ast.Model {
	unconstrained := *model
	unconstrained.Contexts = nil
	for _, ctx := range model.Contexts {
		freeContext := *ctx
		freeContext.Invariants = nil
		freeContext.Aggregates = nil
		for _, agg := range ctx.Aggregates {
			freeAggregate := *agg
			freeAggregate.Invariants = nil
			freeContext.Aggregates = append(freeContext.Aggregates, &freeAggregate)
		}
		unconstrained.Contexts = append(unconstrained.Contexts, &freeContext)
	}
	return &unconstrained
}

func fullModel() *ast.Model {
	return &ast.Model{
		Name: "FullModel",
		Contexts: []*ast.Context{{
			Name: "Orders",
			Aggregates: []*ast.Aggregate{{
				Name: "Order",
				Slices: []*ast.Slice{
					{
						Name: "Create Order",
						Trigger: &ast.Trigger{
							Name:  "PlaceOrderForm",
							Actor: "Customer",
						},
						Commands: []*ast.Command{
							{Name: "CreateOrder"},
							{Name: "ValidatePayment"},
						},
						Events: []*ast.Event{
							{Name: "OrderCreated"},
							{Name: "PaymentValidated"},
						},
						Views: []*ast.View{
							{Name: "OrderSummary", Subscribes: []string{"OrderCreated"}},
						},
						Flows: []*ast.Flow{
							{CommandName: "CreateOrder", EventName: "OrderCreated"},
							{CommandName: "ValidatePayment", EventName: "PaymentValidated"},
						},
						Automations: []*ast.Automation{
							{
								Name:    "InventoryUpdater",
								OnEvent: "OrderCreated",
								Command: "CreateOrder",
							},
						},
						Translations: []*ast.Translation{
							{
								Name:           "PaymentGW",
								ExternalSystem: "Stripe",
								Command:        "ValidatePayment",
								Event:          &ast.Event{Name: "PaymentValidated"},
							},
						},
					},
					{
						Name: "Ship Order",
						Trigger: &ast.Trigger{
							Name: "ShipTimer",
						},
						Commands: []*ast.Command{
							{Name: "ShipOrder"},
						},
						Events: []*ast.Event{
							{Name: "OrderShipped"},
						},
						Flows: []*ast.Flow{
							{CommandName: "ShipOrder", EventName: "OrderShipped"},
						},
					},
				},
			}},
		}},
	}
}
