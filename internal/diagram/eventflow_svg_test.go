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
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

// gearBox is how much room a gear is drawn in, measured from the point it is
// placed at. The gear itself is a <use> and carries no width of its own.
const gearBox = 44

func TestExportEventFlow(t *testing.T) {
	t.Run("shapes", func(t *testing.T) {
		t.Run("an event is drawn as a stadium pill", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", event("CopyBorrowed"),
				&ast.View{Name: "MemberLoansView", Subscribes: []string{"CopyBorrowed"}})

			pill := flowShapeLabelled(t, eventFlowOf(t, model), "CopyBorrowed")

			require.Equal(t, "pill", pill.class)
			require.Equal(t, pill.rect.h/2, pill.rx, "a pill's corner radius is half its height")
		})

		t.Run("an event nothing reads is drawn dashed, so it is not read as consumed", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", event("CopyBorrowed"))

			require.Equal(t, "pill dead", flowShapeLabelled(t, eventFlowOf(t, model), "CopyBorrowed").class)
		})

		t.Run("an automation is drawn as a bare gear, with no box around it", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			require.Len(t, flowGears(t, output), 3, "two automations and one translation reactor")
			for _, shape := range flowRects(t, output) {
				require.NotEqual(t, "RemindOnDueDate", flowTextAt(t, output, shape.rect.centre()),
					"an automation's name is captioned below the gear, never inside a box")
			}
		})

		t.Run("an automation's name is written under its gear", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			gear := flowGearNamed(t, output, "RecallOverdueCopy")
			name := flowTextNamed(t, output, "RecallOverdueCopy")
			require.Equal(t, "gear-name", name.class)
			require.Greater(t, name.y, gear.y, "the name sits below the gear it names")
		})

		t.Run("a schedule-woken automation states its cadence under its name", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			cadence := flowTextNamed(t, output, `every "15m"`)
			require.Equal(t, "caption", cadence.class)
			require.Greater(t, cadence.y, flowTextNamed(t, output, "RecallOverdueCopy").y)
		})

		t.Run("a trigger is drawn as a box saying who works it", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			desk := flowShapeLabelled(t, output, "Lending Desk")
			require.Equal(t, "store", desk.class)
			require.Equal(t, "caption", flowTextNamed(t, output, "Member").class)
		})

		t.Run("an invariant that refuses a command is drawn as a dashed box, never a pill", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			refusal := flowShapeLabelled(t, output, "OneCopyPerLoan")
			require.Equal(t, "reject", refusal.class)
			require.NotEqual(t, refusal.rect.h/2, refusal.rx, "a refusal is a box, so a reader does not take it for a fact in the log")
		})

		t.Run("the prose a construct states is carried as a tooltip", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			require.Equal(t, "A copy left the shelf with a member", flowShapeLabelled(t, output, "CopyBorrowed").tooltip)
			require.Equal(t, "Waits out the grace period, then nudges", flowGearNamed(t, output, "RemindOnDueDate").tooltip)
		})

		t.Run("no two shapes are drawn over each other", func(t *testing.T) {
			shapes := flowBoxes(t, eventFlowOf(t, everyConstructModel(t)))

			for i, shape := range shapes {
				for _, other := range shapes[i+1:] {
					require.False(t, shape.overlaps(other), "%v overlaps %v", shape, other)
				}
			}
		})

		t.Run("every shape is drawn inside the viewBox", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			width, height := flowCanvas(t, output)
			for _, shape := range flowBoxes(t, output) {
				require.GreaterOrEqual(t, shape.x, 0)
				require.GreaterOrEqual(t, shape.y, 0)
				require.LessOrEqual(t, shape.x+shape.w, width)
				require.LessOrEqual(t, shape.y+shape.h, height)
			}
		})
	})

	t.Run("arrows", func(t *testing.T) {
		t.Run("one arrow is drawn for each hop the flow states", func(t *testing.T) {
			model := everyConstructModel(t)

			require.Len(t, flowArrows(eventFlowOf(t, model)), len(diagram.CollapseEventFlow(model).Hops))
		})

		t.Run("an arrow to a refusal is dashed and carries its own arrowhead", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			refused := flowArrowsOfClass(output, "hop refused")
			require.Len(t, refused, 2, "one command refused in each context")
			for _, arrow := range refused {
				require.Contains(t, arrow, `marker-end="url(#head-refused)"`)
			}
		})

		t.Run("the delay an automation waits out is written on the arrow that wakes it", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			require.Equal(t, "hop-label", flowTextNamed(t, output, `after "72h"`).class)
		})

		t.Run("every arrow is drawn before the text, so a label can mask the line it sits on", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			require.Less(t, strings.LastIndex(output, "<path class=\"hop"), strings.Index(output, "<text"))
		})
	})

	t.Run("colours", func(t *testing.T) {
		t.Run("no colour is written as a custom property", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			require.NotContains(t, output, "var(",
				"a renderer outside a browser resolves none, and paints every unresolved fill black")
		})

		t.Run("every class the light theme paints has a dark counterpart", func(t *testing.T) {
			output := eventFlowOf(t, everyConstructModel(t))

			light, dark := flowThemes(t, output)
			for _, selector := range light {
				require.Contains(t, dark, selector)
			}
		})
	})

	t.Run("document", func(t *testing.T) {
		t.Run("the output is well-formed XML", func(t *testing.T) {
			requireValidXML(t, eventFlowOf(t, everyConstructModel(t)))
		})

		t.Run("an empty model draws the frame and no shape at all", func(t *testing.T) {
			output := eventFlowOf(t, &ast.Model{Name: "Empty"})

			requireValidXML(t, output)
			require.Contains(t, output, `<svg xmlns="http://www.w3.org/2000/svg"`)
			require.Empty(t, flowGears(t, output))
			require.Len(t, flowRects(t, output), 1, "only the background the picture is painted on")
		})

		t.Run("a nil model draws the same frame rather than failing", func(t *testing.T) {
			raw, err := diagram.ExportEventFlow(nil, diagram.StyleAuto)

			require.NoError(t, err)
			requireValidXML(t, string(raw))
		})
	})
}

func everyConstructModel(t *testing.T) *ast.Model {
	t.Helper()

	return test.EveryConstructLibraryLendingModel(t)
}

func eventFlowOf(t *testing.T, model *ast.Model) string {
	t.Helper()

	raw, err := diagram.ExportEventFlow(model, diagram.StyleAuto)
	require.NoError(t, err)

	return string(raw)
}

type flowShape struct {
	class   string
	rect    boxRect
	rx      int
	tooltip string
}

type flowText struct {
	class   string
	x, y    int
	content string
}

// flowRects returns the boxes the picture draws, the background among them.
func flowRects(t *testing.T, output string) []flowShape {
	t.Helper()

	var (
		shapes []flowShape
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
				if svgAttributeValue(element, "class") == "" {
					continue
				}
				shapes = append(shapes, flowShape{
					class: svgAttributeValue(element, "class"),
					rect:  svgRectOf(t, element),
					rx:    flowNumber(t, element, "rx"),
				})
				inRect = true
			case "title":
				text.Reset()
			}
		case xml.CharData:
			text.Write(element)
		case xml.EndElement:
			switch element.Name.Local {
			case "rect":
				inRect = false
			case "title":
				if inRect && len(shapes) > 0 {
					shapes[len(shapes)-1].tooltip = text.String()
				}
			}
		}
	}

	return shapes
}

func flowTexts(t *testing.T, output string) []flowText {
	t.Helper()

	var (
		texts   []flowText
		inText  bool
		content strings.Builder
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
			if element.Name.Local != "text" {
				continue
			}
			texts = append(texts, flowText{
				class: svgAttributeValue(element, "class"),
				x:     flowNumber(t, element, "x"),
				y:     flowNumber(t, element, "y"),
			})
			inText = true
			content.Reset()
		case xml.CharData:
			if inText {
				content.Write(element)
			}
		case xml.EndElement:
			if element.Name.Local == "text" && inText {
				texts[len(texts)-1].content = content.String()
				inText = false
			}
		}
	}

	return texts
}

type flowGear struct {
	x, y    int
	tooltip string
}

// flowGears returns where each gear was placed, with the prose it carries.
func flowGears(t *testing.T, output string) []flowGear {
	t.Helper()

	var (
		gears  []flowGear
		inUse  bool
		prose  strings.Builder
		placed = regexp.MustCompile(`translate\((\d+),(\d+)\)`)
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
			case "use":
				at := placed.FindStringSubmatch(svgAttributeValue(element, "transform"))
				require.Len(t, at, 3, "a gear must be placed with a translate")
				gears = append(gears, flowGear{x: flowAtoi(t, at[1]), y: flowAtoi(t, at[2])})
				inUse = true
			case "title":
				prose.Reset()
			}
		case xml.CharData:
			prose.Write(element)
		case xml.EndElement:
			switch element.Name.Local {
			case "use":
				inUse = false
			case "title":
				if inUse && len(gears) > 0 {
					gears[len(gears)-1].tooltip = prose.String()
				}
			}
		}
	}

	return gears
}

func flowGearNamed(t *testing.T, output, name string) flowGear {
	t.Helper()

	caption := flowTextNamed(t, output, name)
	for _, gear := range flowGears(t, output) {
		if gear.x == caption.x && gear.y < caption.y {
			return gear
		}
	}
	require.Failf(t, "no gear under that name", "nothing is captioned %q", name)

	return flowGear{}
}

func flowShapeLabelled(t *testing.T, output, label string) flowShape {
	t.Helper()

	text := flowTextNamed(t, output, label)
	for _, shape := range flowRects(t, output) {
		if shape.class == "paper" {
			continue
		}
		if shape.rect.centre() == [2]int{text.x, text.y} {
			return shape
		}
	}
	require.Failf(t, "no shape holds that label", "nothing is labelled %q", label)

	return flowShape{}
}

func flowTextNamed(t *testing.T, output, content string) flowText {
	t.Helper()

	for _, text := range flowTexts(t, output) {
		if text.content == content {
			return text
		}
	}
	require.Failf(t, "text not drawn", "the picture says nothing about %q", content)

	return flowText{}
}

func flowTextAt(t *testing.T, output string, point [2]int) string {
	t.Helper()

	for _, text := range flowTexts(t, output) {
		if text.x == point[0] && text.y == point[1] {
			return text.content
		}
	}

	return ""
}

// flowBoxes returns the room every shape takes up, the background left out and
// each gear measured as the square it is drawn in.
func flowBoxes(t *testing.T, output string) []boxRect {
	t.Helper()

	var boxes []boxRect
	for _, shape := range flowRects(t, output) {
		if shape.class == "paper" {
			continue
		}
		boxes = append(boxes, shape.rect)
	}
	for _, gear := range flowGears(t, output) {
		boxes = append(boxes, boxRect{x: gear.x - gearBox/2, y: gear.y - gearBox/2, w: gearBox, h: gearBox})
	}

	return boxes
}

var (
	flowPathClass = regexp.MustCompile(`<path class="([^"]*)"`)
	flowViewBox   = regexp.MustCompile(`viewBox="0 0 (\d+) (\d+)"`)
	flowRule      = regexp.MustCompile(`(?m)^([.\w-]+(?:\.[\w-]+)?) \{ ([^}]*)\}$`)
)

func flowArrows(output string) []string {
	var arrows []string
	for _, path := range flowPathClass.FindAllStringSubmatch(output, -1) {
		if strings.HasPrefix(path[1], "hop") {
			arrows = append(arrows, path[1])
		}
	}

	return arrows
}

func flowArrowsOfClass(output, class string) []string {
	var arrows []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, `<path class="`+class+`"`) {
			arrows = append(arrows, line)
		}
	}

	return arrows
}

func flowCanvas(t *testing.T, output string) (int, int) {
	t.Helper()

	size := flowViewBox.FindStringSubmatch(output)
	require.Len(t, size, 3, "the picture must state its viewBox")

	return flowAtoi(t, size[1]), flowAtoi(t, size[2])
}

// flowThemes names the classes each theme paints, so the two can be compared
// without reading the colours themselves.
func flowThemes(t *testing.T, output string) (light, dark []string) {
	t.Helper()

	style := output[strings.Index(output, "<style>"):strings.Index(output, "</style>")]
	media := strings.Index(style, "@media")
	require.Positive(t, media, "the picture must carry a dark theme")

	for _, rule := range flowRule.FindAllStringSubmatch(style[:media], -1) {
		if strings.Contains(rule[2], "fill:") || strings.Contains(rule[2], "stroke:") {
			light = append(light, rule[1])
		}
	}
	for _, rule := range flowRule.FindAllStringSubmatch(style[media:], -1) {
		dark = append(dark, rule[1])
	}

	return light, dark
}

func flowNumber(t *testing.T, element xml.StartElement, name string) int {
	t.Helper()

	value := svgAttributeValue(element, name)
	if value == "" {
		return 0
	}

	return flowAtoi(t, value)
}

func flowAtoi(t *testing.T, value string) int {
	t.Helper()

	number, err := strconv.Atoi(value)
	require.NoError(t, err, "%q must be a whole number", value)

	return number
}
