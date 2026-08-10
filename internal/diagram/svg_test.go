//go:build unit

package diagram_test

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestExportSVG(t *testing.T) {
	t.Run("empty model returns valid SVG with svg root and no diagram content", func(t *testing.T) {
		model := &ast.Model{Name: "Empty"}
		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, `<svg xmlns="http://www.w3.org/2000/svg"`)
		requireValidXML(t, output)
		require.NotContains(t, output, "Wireframes")
	})

	t.Run("names the lane a person enters through for the wireframes it holds", func(t *testing.T) {
		model := minimalModel("Test", "Slice1")
		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		requireValidXML(t, output)
		require.Equal(t,
			[]string{"Wireframes", "Commands / Views", "Events", "External Systems"},
			svgLaneLabels(t, output),
			"only the lane holding what a person touches is renamed; the lanes below it keep their names")
		require.NotContains(t, output, "UI / Triggers")
	})

	t.Run("renders a trigger with its actor", func(t *testing.T) {
		model := singleSliceModel("Test", "S", &ast.Trigger{Name: "SubmitForm", Actor: "User"})

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "SubmitForm")
		require.Contains(t, output, "(User)")
	})

	t.Run("renders command, event and view labels", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			command("MakeReservation"), event("ReservationMade"), view("AvailableRooms"))

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "MakeReservation")
		require.Contains(t, output, "ReservationMade")
		require.Contains(t, output, "AvailableRooms")
	})

	t.Run("connects trigger to command", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S",
						Trigger:  &ast.Trigger{Name: "Click"},
						Commands: []*ast.Command{{Name: "DoAction"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Click")
		require.Contains(t, output, "DoAction")
		require.Equal(t, 1, arrowCount(output), "trigger to command is one arrow")
	})

	t.Run("connects command to event via flow", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S",
						Commands: []*ast.Command{{Name: "Reserve"}},
						Events:   []*ast.Event{{Name: "Reserved"}},
						Flows:    []*ast.Flow{{CommandName: "Reserve", EventName: "Reserved"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Reserve")
		require.Contains(t, output, "Reserved")
		require.Equal(t, 1, arrowCount(output), "command to event is one arrow")
	})

	t.Run("connects event to view via subscribes", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:   "S",
						Events: []*ast.Event{{Name: "OrderPlaced"}},
						Views:  []*ast.View{{Name: "OrderList", Subscribes: []string{"OrderPlaced"}}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "OrderList")
		require.Equal(t, 1, arrowCount(output), "event to view is one arrow")
	})

	t.Run("renders automation with gear icon indicator", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Automations: []*ast.Automation{{
							Name:    "Notifier",
							OnEvent: "OrderPlaced",
							Command: "SendEmail",
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Notifier")
		require.Contains(t, output, "\u2699") // gear character
	})

	t.Run("connects event to automation to command", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:        "S",
						Events:      []*ast.Event{{Name: "OrderPlaced"}},
						Commands:    []*ast.Command{{Name: "SendEmail"}},
						Automations: []*ast.Automation{{Name: "Notifier", OnEvent: "OrderPlaced", Command: "SendEmail"}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "OrderPlaced")
		require.Contains(t, output, "SendEmail")
		require.Contains(t, output, "Notifier")
		require.Equal(t, 2, arrowCount(output), "event to automation to command is two arrows")
	})

	t.Run("renders external system as gray dashed rounded rect", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name: "S",
						Translations: []*ast.Translation{{
							Name:           "Import",
							ExternalSystem: "Stripe",
							Command:        "Charge",
							Event:          &ast.Event{Name: "Charged"},
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Stripe")
		require.Contains(t, output, `stroke-dasharray`)
	})

	t.Run("connects external system through reactor to command and event", func(t *testing.T) {
		model := &ast.Model{
			Name: "Test",
			Contexts: []*ast.Context{{
				Name: "Ctx",
				Aggregates: []*ast.Aggregate{{
					Name: "Agg",
					Slices: []*ast.Slice{{
						Name:     "S",
						Commands: []*ast.Command{{Name: "Charge"}},
						Events:   []*ast.Event{{Name: "Charged"}},
						Translations: []*ast.Translation{{
							Name:           "Payment",
							ExternalSystem: "Stripe",
							Command:        "Charge",
							Event:          &ast.Event{Name: "Charged"},
						}},
					}},
				}},
			}},
		}

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Charge")
		require.Contains(t, output, "Charged")
		require.Contains(t, output, "Stripe")
		// external system -> reactor, reactor -> command, command -> event
		require.Equal(t, 3, arrowCount(output))
	})

	t.Run("event with external source includes source label", func(t *testing.T) {
		model := singleSliceModel("Test", "S",
			eventWithSource("PaymentReceived", "external", "Stripe"))

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "PaymentReceived")
		require.Contains(t, output, "Stripe")
	})

	t.Run("complete model with all element types produces well-formed diagram", func(t *testing.T) {
		model := fullModel()

		raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
		require.NoError(t, err)

		output := string(raw)
		require.Contains(t, output, "Wireframes")
		require.Contains(t, output, "Commands / Views")
		require.Contains(t, output, "Events")
		require.Equal(t, 11, arrowCount(output), "every flow, subscription, automation and translation edge is drawn")
	})

	t.Run("descriptions", func(t *testing.T) {
		t.Run("hovering a shape shows the description of the construct it was drawn for", func(t *testing.T) {
			raw, err := diagram.ExportSVG(describedModel(), diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			requireValidXML(t, output)
			requireEveryDescriptionShown(t, output, svgTooltipOf)
		})

		t.Run("describing a model leaves the picture itself untouched", func(t *testing.T) {
			described, err := diagram.ExportSVG(describedModel(), diagram.StyleAuto)
			require.NoError(t, err)
			plain, err := diagram.ExportSVG(withoutDescriptions(describedModel()), diagram.StyleAuto)
			require.NoError(t, err)

			require.Equal(t, svgPicture(t, string(plain)), svgPicture(t, string(described)),
				"prose must not add, move or repaint a shape, nor disturb the arrows between shapes")
			require.NotContains(t, string(plain), "<title",
				"a model that describes nothing is written exactly as it was before shapes could be titled")
		})

		t.Run("a description written with markup characters reads back as written", func(t *testing.T) {
			prose := `Rooms held < 24h & marked "urgent"`
			model := describedModel()
			model.Contexts[0].Description = prose
			model.Contexts[0].Aggregates[0].Slices[0].Commands[0].Description = prose

			raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
			require.NoError(t, err)

			output := string(raw)
			requireValidXML(t, output)
			require.Equal(t, prose, svgTooltipOf(t, output, "Bookings"))
			require.Equal(t, prose, svgTooltipOf(t, output, "HoldRoom"))
		})
	})

	t.Run("rejection badges", func(t *testing.T) {
		t.Run("draws one badge per rejection edge, labelled with the invariant it names", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))

			var badged []string
			for _, box := range svgBoxes(t, output) {
				if strings.Contains(box.appearance, fillRejectionHex) {
					badged = append(badged, box.label)
				}
			}

			var expected []string
			for _, edge := range test.RejectionLibraryLendingRejections {
				expected = append(expected, edge.InvariantName)
			}
			require.Equal(t, expected, badged)
		})

		t.Run("a badge carries the invariant's statement as the title a browser shows on hover", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))

			require.Equal(t, "A desk seats at most one reader at any moment",
				svgTooltipOf(t, output, "OneReaderPerDesk"))
		})

		t.Run("a statement written with markup characters reads back as written", func(t *testing.T) {
			prose := `Held < 24h & marked "urgent"`
			model := test.RejectionLibraryLendingModel(t)
			model.Contexts[1].Invariants[0].Statement = prose

			output := svgOf(t, model)

			requireValidXML(t, output)
			require.Equal(t, prose, svgTooltipOf(t, output, "OneReaderPerDesk"))
		})

		t.Run("a dashed arrow runs from the rejected command to that badge, painted unlike a flow arrow", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))

			connections := svgConnections(t, output)
			var rejection, flow diagramConnection
			for _, c := range connections {
				if c.source == "ClaimDesk" && c.target == "OneReaderPerDesk" {
					rejection = c
				}
				if c.source == "ClaimDesk" && c.target == "DeskClaimed" {
					flow = c
				}
			}

			require.NotEmpty(t, flow.paint, "the flow arrow this is compared against must be in the same render")
			require.NotEmpty(t, rejection.paint, "no dashed arrow reaches the badge")
			require.NotEqual(t, flow.paint, rejection.paint,
				"a rejection must not be drawn like the flow beside it")
			require.Contains(t, rejection.paint, "stroke-dasharray")
		})

		t.Run("two slices rejecting one invariant each get their own badge and their own arrow", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))

			var badgeCentres [][2]int
			for _, box := range svgBoxes(t, output) {
				if box.label == "OneCopyPerLoan" {
					badgeCentres = append(badgeCentres, box.rect.centre())
				}
			}
			require.Len(t, badgeCentres, 2, "each of the two slices draws its own badge")

			// Comparing endpoints rather than target labels is the whole point:
			// both arrows resolve to the label "OneCopyPerLoan" even when they
			// end at the same box, so a label comparison cannot see the collapse.
			var reached [][2]int
			for _, end := range dashedArrowEndpoints(t, output) {
				if slices.Contains(badgeCentres, end) {
					reached = append(reached, end)
				}
			}

			require.ElementsMatch(t, badgeCentres, reached,
				"each slice's dashed arrow ends at the badge in its own column; filing a badge under the invariant's name alone points both at whichever slice was drawn last")

			require.Equal(t, []string{"BorrowCopy", "ReturnCopy"}, sourcesReaching(t, output, "OneCopyPerLoan"))
		})

		t.Run("a badge adds one shape to its slice's event row and moves no label the twin drew", func(t *testing.T) {
			stated := test.RejectionLibraryLendingModel(t)
			unstated := test.WithoutRejections(stated)
			require.Empty(t, test.DeclaredRejections(unstated))
			require.Equal(t, test.RejectionLibraryLendingRejections, test.DeclaredRejections(stated))

			statedShapes := svgShapes(t, svgOf(t, stated))
			unstatedShapes := svgShapes(t, svgOf(t, unstated))

			require.Len(t, statedShapes, len(unstatedShapes)+len(test.RejectionLibraryLendingRejections),
				"a badge is one rect followed by exactly one text, so it adds one shape and no more")

			var statedLabels, unstatedLabels []string
			for _, shape := range statedShapes {
				if !strings.Contains(shape.attributes, fillRejectionHex) {
					statedLabels = append(statedLabels, shape.label)
				}
			}
			for _, shape := range unstatedShapes {
				unstatedLabels = append(unstatedLabels, shape.label)
			}
			require.Equal(t, unstatedLabels, statedLabels,
				"a badge emitting a stray text element overwrites the label of the box before it")
		})

		t.Run("badges overlap nothing and sit in the lane their slice's events are drawn in", func(t *testing.T) {
			output := svgOf(t, test.RejectionLibraryLendingModel(t))
			boxes := svgBoxes(t, output)

			rects := drawnElementRects(t, output)
			require.Empty(t, boxesDrawnOver(rects, slices.Sorted(maps.Keys(rects))))

			eventLane := boxLabelled(t, boxes, "Events").rect
			within := labelsWithin(boxes, eventLane)
			for _, edge := range test.RejectionLibraryLendingRejections {
				require.Contains(t, within, edge.InvariantName)
			}
		})

		t.Run("a badge is narrower than a lane, so the diagram still names four lanes and keeps its viewBox", func(t *testing.T) {
			stated := test.RejectionLibraryLendingModel(t)
			statedOut := svgOf(t, stated)
			unstatedOut := svgOf(t, test.WithoutRejections(stated))

			requireValidXML(t, statedOut)
			require.Equal(t, svgLaneLabels(t, unstatedOut), svgLaneLabels(t, statedOut))
			require.Equal(t, svgViewBox(t, unstatedOut), svgViewBox(t, statedOut),
				"a badge takes a place in a row that already exists rather than growing the canvas")
		})
	})
}

// fillRejectionHex is the badge's fill as svgAttributes renders it. Stating it
// here rather than importing the constant keeps the test reading the bytes a
// browser would.
const fillRejectionHex = "fill=#f8cecc"

func svgOf(t *testing.T, model *ast.Model) string {
	t.Helper()

	raw, err := diagram.ExportSVG(model, diagram.StyleAuto)
	require.NoError(t, err)

	return string(raw)
}

var svgViewBoxAttribute = regexp.MustCompile(`viewBox="([^"]*)"`)

func svgViewBox(t *testing.T, output string) string {
	t.Helper()

	found := svgViewBoxAttribute.FindStringSubmatch(output)
	require.NotNil(t, found, "the diagram states no viewBox")

	return found[1]
}

// dashedArrowEndpoints returns the point each dashed arrow ends at, so a test can
// say which box an arrow reached rather than which label that box happens to
// carry — two boxes may carry the same one.
func dashedArrowEndpoints(t *testing.T, output string) [][2]int {
	t.Helper()

	var ends [][2]int
	for _, arrow := range svgArrows(output) {
		if !strings.Contains(arrow, "stroke-dasharray") {
			continue
		}
		path := svgPathData.FindStringSubmatch(arrow)
		require.NotNil(t, path, "an arrow carries no path: %s", arrow)

		points := svgPathPoint.FindAllStringSubmatch(path[1], -1)
		require.NotEmpty(t, points, "an arrow's path names no point: %s", arrow)

		ends = append(ends, svgPoint(t, points[len(points)-1]))
	}

	return ends
}

func sourcesReaching(t *testing.T, output, label string) []string {
	t.Helper()

	var sources []string
	for _, c := range svgConnections(t, output) {
		if c.target == label {
			sources = append(sources, c.source)
		}
	}

	return sources
}

// drawnElementRects keys every labelled box by its label and the position it was
// drawn at, so two slices drawing a badge for one invariant are two entries
// rather than one overwriting the other. The lanes are left out: a lane is drawn
// around the boxes it holds, so it overlaps all of them by design.
func drawnElementRects(t *testing.T, output string) map[string]boxRect {
	t.Helper()

	lanes := make(map[string]bool)
	for _, label := range svgLaneLabels(t, output) {
		lanes[label] = true
	}

	rects := make(map[string]boxRect)
	for i, box := range svgBoxes(t, output) {
		if box.label == "" || lanes[box.label] {
			continue
		}
		rects[fmt.Sprintf("%s#%d", box.label, i)] = box.rect
	}

	return rects
}

// --- svg helpers ---

func arrowCount(output string) int {
	return len(svgArrows(output))
}

// svgArrows returns the arrows the diagram draws between its boxes.
func svgArrows(output string) []string {
	var arrows []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "marker-end") {
			arrows = append(arrows, line)
		}
	}

	return arrows
}

type svgShape struct {
	attributes string
	rect       boxRect
	label      string
	tooltip    string
}

// svgShapes returns the diagram's boxes in document order, decoded through an
// XML parser so a test sees the text a reader sees rather than its escaped form.
// A browser shows only the title nested inside the box being hovered, so a title
// written anywhere else belongs to no box here either; labels are drawn as text
// siblings, so those attach to the nearest box before them.
func svgShapes(t *testing.T, output string) []svgShape {
	t.Helper()

	var (
		shapes []svgShape
		inRect bool
		text   strings.Builder
	)

	decoder := xml.NewDecoder(strings.NewReader(output))
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err, "output must be well-formed XML")

		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "rect":
				shapes = append(shapes, svgShape{attributes: svgAttributes(element), rect: svgRectOf(t, element)})
				inRect = true
			case "title", "text":
				text.Reset()
			}
		case xml.CharData:
			text.Write(element)
		case xml.EndElement:
			switch element.Name.Local {
			case "rect":
				inRect = false
			case "title":
				if inRect {
					shapes[len(shapes)-1].tooltip = text.String()
				}
			case "text":
				if len(shapes) > 0 {
					shapes[len(shapes)-1].label = text.String()
				}
			}
		}
	}

	return shapes
}

func svgBoxes(t *testing.T, output string) []diagramBox {
	t.Helper()

	var boxes []diagramBox
	for _, shape := range svgShapes(t, output) {
		boxes = append(boxes, diagramBox{label: shape.label, appearance: shape.attributes, rect: shape.rect})
	}

	return boxes
}

// svgLaneLabels names the swimlanes the diagram draws, in the order it draws
// them. A lane is the only shape drawn as a band across the whole picture, so
// the widest shapes are the lanes and nothing else is.
func svgLaneLabels(t *testing.T, output string) []string {
	t.Helper()

	shapes := svgShapes(t, output)

	var widest int
	for _, shape := range shapes {
		widest = max(widest, shape.rect.w)
	}

	var labels []string
	for _, shape := range shapes {
		if shape.rect.w == widest {
			labels = append(labels, shape.label)
		}
	}

	return labels
}

func svgRectOf(t *testing.T, element xml.StartElement) boxRect {
	t.Helper()

	measure := func(name string) int {
		for _, a := range element.Attr {
			if a.Name.Local != name {
				continue
			}
			value, err := strconv.Atoi(a.Value)
			require.NoError(t, err, "a box's %s must be a whole number", name)
			return value
		}
		require.Fail(t, "a box is drawn with no "+name)
		return 0
	}

	return boxRect{x: measure("x"), y: measure("y"), w: measure("width"), h: measure("height")}
}

var (
	svgPathData  = regexp.MustCompile(`\sd="([^"]*)"`)
	svgPathPoint = regexp.MustCompile(`(-?\d+),(-?\d+)`)
)

// svgConnections returns the arrows the diagram draws, each named by the boxes
// its two ends meet and carrying how it is painted, so a test can say which
// boxes an arrow runs between instead of restating its coordinates.
func svgConnections(t *testing.T, output string) []diagramConnection {
	t.Helper()

	labelled := make(map[[2]int]string)
	for _, shape := range svgShapes(t, output) {
		labelled[shape.rect.centre()] = shape.label
	}

	boxAt := func(point [2]int) string {
		label, drawn := labelled[point]
		require.True(t, drawn, "an arrow meets %v, where the diagram draws no box", point)
		return label
	}

	var connections []diagramConnection
	for _, arrow := range svgArrows(output) {
		path := svgPathData.FindStringSubmatch(arrow)
		require.NotNil(t, path, "an arrow carries no path: %s", arrow)

		points := svgPathPoint.FindAllStringSubmatch(path[1], -1)
		require.NotEmpty(t, points, "an arrow's path names no point: %s", arrow)

		connections = append(connections, diagramConnection{
			source: boxAt(svgPoint(t, points[0])),
			target: boxAt(svgPoint(t, points[len(points)-1])),
			paint:  strings.TrimSpace(svgPathData.ReplaceAllString(arrow, "")),
		})
	}

	return connections
}

func svgPoint(t *testing.T, point []string) [2]int {
	t.Helper()

	x, err := strconv.Atoi(point[1])
	require.NoError(t, err, "an arrow's path names %q, which is no point", point[0])
	y, err := strconv.Atoi(point[2])
	require.NoError(t, err, "an arrow's path names %q, which is no point", point[0])

	return [2]int{x, y}
}

func svgAttributes(element xml.StartElement) string {
	pairs := make([]string, 0, len(element.Attr))
	for _, a := range element.Attr {
		pairs = append(pairs, a.Name.Local+"="+a.Value)
	}
	return strings.Join(pairs, " ")
}

func svgTooltipOf(t *testing.T, output, label string) string {
	t.Helper()

	var tooltips []string
	for _, shape := range svgShapes(t, output) {
		if strings.Contains(shape.label, label) {
			tooltips = append(tooltips, shape.tooltip)
		}
	}
	require.Len(t, tooltips, 1, "expected one svg shape labelled %q", label)

	return tooltips[0]
}

// svgPicture returns everything the diagram draws — every box with the text on
// it, how it is painted and where it sits, then the arrows between them —
// leaving out what a box only says when hovered, so a described diagram can be
// compared with the one the same model draws without prose.
func svgPicture(t *testing.T, output string) []string {
	t.Helper()

	var drawn []string
	for _, box := range svgBoxes(t, output) {
		drawn = append(drawn, box.appearance+" "+box.label)
	}

	return append(drawn, svgArrows(output)...)
}
