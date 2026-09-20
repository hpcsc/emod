//go:build unit

package diagram_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestExportEventFlowMermaid(t *testing.T) {
	t.Run("nodes", func(t *testing.T) {
		t.Run("an event is a stadium", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", event("CopyBorrowed"),
				&ast.View{Name: "MemberLoansView", Subscribes: []string{"CopyBorrowed"}})

			require.Contains(t, mermaidFlowOf(t, model), `    CopyBorrowed(["CopyBorrowed"])`)
		})

		t.Run("an event nothing reads takes the class painted for a dead end", func(t *testing.T) {
			model := singleSliceModel("Lending", "Borrow Copy", event("CopyBorrowed"))

			require.Contains(t, mermaidFlowOf(t, model), "    class CopyBorrowed dead")
		})

		t.Run("an automation is its gear above its name", func(t *testing.T) {
			output := mermaidFlowOf(t, everyConstructModel(t))

			require.Contains(t, output, `    RemindOnDueDate["⚙<br/>RemindOnDueDate"]`)
			require.Contains(t, output, "    class RemindOnDueDate,RecallOverdueCopy,ExternalDeskBookingImport gear")
		})

		t.Run("a schedule is a line under the name it wakes", func(t *testing.T) {
			output := mermaidFlowOf(t, everyConstructModel(t))

			require.Contains(t, output, `RecallOverdueCopy["⚙<br/>RecallOverdueCopy<br/>every #quot;15m#quot;"]`,
				"the quotes the DSL spells a cadence with are written as the entity code Mermaid resolves")
		})

		t.Run("an origin says under its name what it is", func(t *testing.T) {
			output := mermaidFlowOf(t, everyConstructModel(t))

			require.Contains(t, output, `    Lending_Desk("Lending Desk<br/>Member")`)
			require.Contains(t, output, `    Room_Booking_API("Room Booking API<br/>external system")`)
			require.Contains(t, output, `    ReturnCopy("ReturnCopy<br/>command")`)
		})

		t.Run("an invariant that refuses a command takes the class painted for a refusal", func(t *testing.T) {
			output := mermaidFlowOf(t, everyConstructModel(t))

			require.Contains(t, output, `    OneCopyPerLoan["OneCopyPerLoan"]`)
			require.Contains(t, output, "    class OneCopyPerLoan,OneReaderPerDesk reject")
		})

		t.Run("two nodes whose names clean to one identifier stay two nodes", func(t *testing.T) {
			model := test.EveryConstructLibraryLendingModel(t)
			renamed := model.Contexts[1]
			renamed.Invariants[0].Name = "OneCopyPerLoan"
			for _, ref := range renamed.SliceRefs() {
				for _, rejection := range ref.Slice.Rejections {
					rejection.InvariantName = "OneCopyPerLoan"
				}
			}

			output := mermaidFlowOf(t, model)

			require.Contains(t, output, `    OneCopyPerLoan["OneCopyPerLoan"]`)
			require.Contains(t, output, `    OneCopyPerLoan_2["OneCopyPerLoan"]`)
		})

		t.Run("a name Mermaid reads as its own keyword is not used as an identifier", func(t *testing.T) {
			model := singleSliceModel("Lending", "Close", command("End"), event("Ended"))
			model.Contexts[0].Aggregates[0].Slices[0].Flows = []*ast.Flow{{CommandName: "End", EventName: "Ended"}}

			output := mermaidFlowOf(t, model)

			require.NotRegexp(t, regexp.MustCompile(`(?m)^    End\(`), output)
			require.Contains(t, output, `    nEnd("End<br/>command")`)
		})
	})

	t.Run("arrows", func(t *testing.T) {
		t.Run("every hop the flow states is one arrow", func(t *testing.T) {
			model := everyConstructModel(t)

			require.Len(t, mermaidFlowArrows(mermaidFlowOf(t, model)), len(diagram.CollapseEventFlow(model).Hops))
		})

		t.Run("the delay an automation waits out is written on the arrow that wakes it", func(t *testing.T) {
			output := mermaidFlowOf(t, everyConstructModel(t))

			require.Contains(t, output, `    CopyBorrowed -->|after #quot;72h#quot;| RemindOnDueDate`)
		})

		t.Run("an arrow to a refusal is dotted and restyled by the index it was written at", func(t *testing.T) {
			output := mermaidFlowOf(t, everyConstructModel(t))

			arrows := mermaidFlowArrows(output)
			var dotted []string
			for i, arrow := range arrows {
				if strings.Contains(arrow, "-.->") {
					dotted = append(dotted, strconv.Itoa(i))
				}
			}
			require.Len(t, dotted, 2, "one command refused in each context")
			require.Contains(t, output, "    linkStyle "+strings.Join(dotted, ",")+" stroke:")
		})
	})

	t.Run("parsing", func(t *testing.T) {
		t.Run("the diagram type follows the frontmatter with no comment between them", func(t *testing.T) {
			lines := strings.Split(mermaidFlowOf(t, everyConstructModel(t)), "\n")

			frontmatter := 0
			for i, line := range lines[1:] {
				if line == "---" {
					frontmatter = i + 1
					break
				}
			}
			require.Positive(t, frontmatter, "the file must open with frontmatter")
			require.Equal(t, "flowchart TB", lines[frontmatter+1],
				"a comment between the frontmatter and the diagram type fails to parse")
		})

		t.Run("no comment line is left bare, which Mermaid draws as a node", func(t *testing.T) {
			for _, line := range strings.Split(mermaidFlowOf(t, everyConstructModel(t)), "\n") {
				require.NotEqual(t, "%%", strings.TrimSpace(line))
			}
		})

		t.Run("no label carries a quote that would end it", func(t *testing.T) {
			for _, line := range mermaidFlowLabels(mermaidFlowOf(t, everyConstructModel(t))) {
				require.NotContains(t, line, `"`, "a quote inside a label ends it")
			}
		})

		t.Run("the file asks for the layout that keeps a flow readable", func(t *testing.T) {
			require.Contains(t, mermaidFlowOf(t, everyConstructModel(t)), "layout: elk")
		})

		t.Run("an empty model writes a diagram with no node in it", func(t *testing.T) {
			output := mermaidFlowOf(t, &ast.Model{Name: "Empty"})

			require.Contains(t, output, "flowchart TB")
			require.Empty(t, mermaidFlowArrows(output))
			require.NotContains(t, output, "classDef")
		})

		t.Run("a nil model writes the same rather than failing", func(t *testing.T) {
			raw, err := diagram.ExportEventFlowMermaid(nil, diagram.StyleAuto)

			require.NoError(t, err)
			require.Contains(t, string(raw), "flowchart TB")
		})
	})
}

func mermaidFlowOf(t *testing.T, model *ast.Model) string {
	t.Helper()

	raw, err := diagram.ExportEventFlowMermaid(model, diagram.StyleAuto)
	require.NoError(t, err)

	return string(raw)
}

var mermaidFlowArrow = regexp.MustCompile(`(?m)^ {4}\w+ -\.?->`)

func mermaidFlowArrows(output string) []string {
	var arrows []string
	for _, line := range strings.Split(output, "\n") {
		if mermaidFlowArrow.MatchString(line) {
			arrows = append(arrows, line)
		}
	}

	return arrows
}

var mermaidFlowLabelText = regexp.MustCompile(`"([^"]*)"`)

// mermaidFlowLabels returns the text inside every label the file writes, so a
// character that would end one can be looked for in what it says rather than in
// the punctuation around it.
func mermaidFlowLabels(output string) []string {
	var labels []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "%%") {
			continue
		}
		for _, match := range mermaidFlowLabelText.FindAllStringSubmatch(line, -1) {
			labels = append(labels, match[1])
		}
	}

	return labels
}
