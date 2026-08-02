//go:build unit

package diagram_test

import (
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
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
