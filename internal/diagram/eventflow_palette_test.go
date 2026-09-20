//go:build unit

package diagram_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The two event flow formats each write their colours out in full, and the DSL
// reference documents them. Three copies of one palette drift, so each element
// is checked against what the reference says rather than against the other
// format.
func TestEventFlowPalette(t *testing.T) {
	elements := []struct {
		row      string
		selector string
		property string
	}{
		{row: "Event pill", selector: ".pill", property: "fill"},
		{row: "Event pill", selector: ".pill", property: "stroke"},
		{row: "Event nothing reads", selector: ".pill.dead", property: "stroke"},
		{row: "Automation gear", selector: ".gear", property: "fill"},
		{row: "Origin box", selector: ".store", property: "fill"},
		{row: "Origin box", selector: ".store", property: "stroke"},
		{row: "Invariant refusing it", selector: ".reject", property: "fill"},
		{row: "Invariant refusing it", selector: ".reject", property: "stroke"},
		{row: "Arrow", selector: ".hop", property: "stroke"},
		{row: "Arrow to a refusal", selector: ".hop.refused", property: "stroke"},
	}

	t.Run("the svg", func(t *testing.T) {
		t.Run("paints the light theme in the colours the reference documents", func(t *testing.T) {
			documented := documentedEventFlowPalette(t)
			light, _ := eventFlowStyleRules(t, eventFlowOf(t, everyConstructModel(t)))

			for _, element := range elements {
				require.Equal(t, documented[element.row].light(element.property), light[element.selector][element.property],
					"%s %s of %q", element.selector, element.property, element.row)
			}
		})

		t.Run("paints the dark theme in the colours the reference documents", func(t *testing.T) {
			documented := documentedEventFlowPalette(t)
			_, dark := eventFlowStyleRules(t, eventFlowOf(t, everyConstructModel(t)))

			for _, element := range elements {
				require.Equal(t, documented[element.row].dark(element.property), dark[element.selector][element.property],
					"%s %s of %q in dark mode", element.selector, element.property, element.row)
			}
		})
	})

	t.Run("the mermaid", func(t *testing.T) {
		t.Run("paints each class in the light colours the reference documents", func(t *testing.T) {
			documented := documentedEventFlowPalette(t)
			classes := mermaidClassDefs(t, mermaidFlowOf(t, everyConstructModel(t)))

			for class, row := range map[string]string{
				"event":  "Event pill",
				"dead":   "Event nothing reads",
				"origin": "Origin box",
				"reject": "Invariant refusing it",
			} {
				require.Equal(t, documented[row].lightFill, classes[class]["fill"], "fill of class %s", class)
				require.Equal(t, documented[row].lightStroke, classes[class]["stroke"], "stroke of class %s", class)
			}

			require.Equal(t, documented["Automation gear"].lightFill, classes["gear"]["color"],
				"a gear Mermaid draws as text takes the colour the drawn gear is filled with")
		})

		t.Run("restyles a refusal's arrow in the colour the reference documents", func(t *testing.T) {
			documented := documentedEventFlowPalette(t)

			require.Contains(t, mermaidFlowOf(t, everyConstructModel(t)),
				"stroke:"+documented["Arrow to a refusal"].lightStroke)
		})
	})
}

type eventFlowColours struct {
	lightFill, lightStroke, darkFill, darkStroke string
}

func (c eventFlowColours) light(property string) string {
	if property == "fill" {
		return c.lightFill
	}

	return c.lightStroke
}

func (c eventFlowColours) dark(property string) string {
	if property == "fill" {
		return c.darkFill
	}

	return c.darkStroke
}

var documentedPaletteRow = regexp.MustCompile(`\|\s*([A-Za-z ]+?)\s*\|\s*(#[0-9A-Fa-f]{6}|—)\s*\|\s*(#[0-9A-Fa-f]{6}|—)\s*\|\s*(#[0-9A-Fa-f]{6}|—)\s*\|\s*(#[0-9A-Fa-f]{6}|—)\s*\|`)

// documentedEventFlowPalette reads the event flow palette out of the DSL
// reference, which is where the values are documented for a reader.
func documentedEventFlowPalette(t *testing.T) map[string]eventFlowColours {
	t.Helper()

	raw, err := os.ReadFile("../../docs/dsl-reference.md")
	require.NoError(t, err)

	content := string(raw)
	start := strings.Index(content, "### Event Flow Palette")
	require.NotEqual(t, -1, start, "docs/dsl-reference.md must document the event flow palette")

	rows := documentedPaletteRow.FindAllStringSubmatch(content[start:], -1)
	require.Len(t, rows, 7, "the palette must document every element the two formats paint")

	palette := make(map[string]eventFlowColours, len(rows))
	for _, row := range rows {
		palette[row[1]] = eventFlowColours{
			lightFill:   documentedColour(row[2]),
			lightStroke: documentedColour(row[3]),
			darkFill:    documentedColour(row[4]),
			darkStroke:  documentedColour(row[5]),
		}
	}

	return palette
}

// documentedColour reads the dash the table writes where an element paints
// nothing as the absence the style block states by leaving the property out.
func documentedColour(cell string) string {
	if cell == "—" {
		return ""
	}

	return cell
}

var (
	styleRule        = regexp.MustCompile(`(?m)^([.\w-]+(?:\.[\w-]+)?) \{ (.*) \}$`)
	mermaidClassRule = regexp.MustCompile(`(?m)^\s*classDef (\w+) (.*)$`)
)

// eventFlowStyleRules returns each theme's rules as selector, property, value.
func eventFlowStyleRules(t *testing.T, output string) (light, dark map[string]map[string]string) {
	t.Helper()

	style := output[strings.Index(output, "<style>"):strings.Index(output, "</style>")]
	media := strings.Index(style, "@media")
	require.Positive(t, media, "the picture must carry a dark theme")

	return declarationsOf(style[:media]), declarationsOf(style[media:])
}

func declarationsOf(block string) map[string]map[string]string {
	rules := make(map[string]map[string]string)
	for _, rule := range styleRule.FindAllStringSubmatch(block, -1) {
		properties := make(map[string]string)
		for _, declaration := range strings.Split(rule[2], ";") {
			name, value, found := strings.Cut(declaration, ":")
			if !found {
				continue
			}
			properties[strings.TrimSpace(name)] = strings.TrimSpace(value)
		}
		rules[rule[1]] = properties
	}

	return rules
}

func mermaidClassDefs(t *testing.T, output string) map[string]map[string]string {
	t.Helper()

	classes := make(map[string]map[string]string)
	for _, rule := range mermaidClassRule.FindAllStringSubmatch(output, -1) {
		properties := make(map[string]string)
		for _, declaration := range strings.Split(rule[2], ",") {
			name, value, found := strings.Cut(declaration, ":")
			if !found {
				continue
			}
			properties[strings.TrimSpace(name)] = strings.TrimSpace(value)
		}
		classes[rule[1]] = properties
	}
	require.NotEmpty(t, classes, "the file must define the classes it paints with")

	return classes
}
