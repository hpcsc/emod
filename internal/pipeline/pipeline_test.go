//go:build unit

package pipeline_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/pipeline"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

func TestPipeline(t *testing.T) {
	t.Run("extract request", func(t *testing.T) {
		t.Run("returns the source and the file name it came from", func(t *testing.T) {
			req, err := pipeline.ExtractRequest(`{"source": "model MyModel", "filename": "orders.emod"}`)

			require.NoError(t, err)
			require.Equal(t, pipeline.Request{Source: "model MyModel", Filename: "orders.emod"}, req)
		})

		t.Run("names source that arrives with no file name input.emod", func(t *testing.T) {
			req, err := pipeline.ExtractRequest(`{"source": "model MyModel"}`)

			require.NoError(t, err)
			require.Equal(t, pipeline.Request{Source: "model MyModel", Filename: "input.emod"}, req)
		})

		t.Run("malformed JSON returns error", func(t *testing.T) {
			_, err := pipeline.ExtractRequest(`not json`)

			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid JSON")
		})

		t.Run("a missing or empty source field returns error", func(t *testing.T) {
			for _, input := range []string{`{}`, `{"source": ""}`, `{"filename": "orders.emod"}`} {
				_, err := pipeline.ExtractRequest(input)

				require.Error(t, err, "input %s", input)
				require.Contains(t, err.Error(), "missing source field")
			}
		})

		t.Run("both viewer runtimes send the source under the keys the request reads", func(t *testing.T) {
			read := jsonKeysOf(reflect.TypeOf(pipeline.Request{}))
			require.Equal(t, []string{"filename", "source"}, read)

			for _, platform := range []string{
				"../frontend/static/platform.browser.js",
				"../frontend/desktop/platform.desktop.js",
			} {
				require.Equal(t, read, keysSentBy(t, platform, "parseEmod"), platform)
			}
		})
	})

	t.Run("run on source", func(t *testing.T) {
		t.Run("hands the unwrapped source and file name to the entry point and returns its bytes", func(t *testing.T) {
			var seen pipeline.Request
			answer := pipeline.RunOnSource(`{"source": "model MyModel", "filename": "orders.emod"}`, func(source, filename string) ([]byte, error) {
				seen = pipeline.Request{Source: source, Filename: filename}
				return []byte(`{"ok": true}`), nil
			})

			require.Equal(t, pipeline.Request{Source: "model MyModel", Filename: "orders.emod"}, seen)
			require.Equal(t, `{"ok": true}`, answer)
		})

		t.Run("reports a malformed envelope without running the entry point", func(t *testing.T) {
			called := false
			answer := pipeline.RunOnSource("not json", func(string, string) ([]byte, error) {
				called = true
				return nil, nil
			})

			require.False(t, called)
			require.Contains(t, answer, "invalid JSON")
			require.NotContains(t, answer, `"ok"`)
		})

		t.Run("wraps a failure from the entry point in the error envelope", func(t *testing.T) {
			answer := pipeline.RunOnSource(`{"source": "model MyModel"}`, func(string, string) ([]byte, error) {
				return []byte("partial output"), errors.New("pipeline exploded")
			})

			require.Equal(t, pipeline.ErrorJSON("pipeline exploded"), answer)
			require.NotContains(t, answer, "partial output")
		})
	})

	t.Run("export diagram", func(t *testing.T) {
		t.Run("returns the model's nodes and edges with no diagnostics", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram(test.BillingPayments, "billing.emod")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)

			require.Empty(t, envelope.Diagnostics)
			require.Equal(t, "Billing", envelope.Diagram.ModelName)
			require.Equal(t, []string{"TakePayment", "PaymentTaken"},
				labelsOfType(envelope.Diagram.Nodes, "command", "event"))
			require.Equal(t, []edge{{Source: "command-1", Target: "event-1", Type: "flow"}},
				envelope.Diagram.Edges)
		})

		t.Run("reports diagnostics for unparseable source and still returns an envelope", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram("foobar {\n}\n", "broken.emod")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)

			require.NotEmpty(t, envelope.Diagnostics, "an unparseable model must report why")
			require.Contains(t, envelope.Diagnostics[0].Message, "foobar")
		})

		t.Run("empty source yields an envelope with no nodes", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram("", "empty.emod")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)

			require.Empty(t, envelope.Diagram.Nodes)
		})

		t.Run("says the source did not parse when the parser reported on it, while still answering what it recovered", func(t *testing.T) {
			broken := test.BillingPayments + "context \"Refunds\" {\n"

			result, err := pipeline.RunPipelineExportDiagram(broken, "billing.emod")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)
			require.False(t, envelope.Parsed)
			require.Equal(t, []string{"TakePayment", "PaymentTaken"}, labelsOfType(envelope.Diagram.Nodes, "command", "event"))
		})

		t.Run("says the source parsed when it reports only validation and lint findings", func(t *testing.T) {
			missingEvent := strings.Replace(test.BillingPayments,
				"TakePayment -> PaymentTaken",
				"TakePayment -> PaymentRefunded", 1)
			require.NotEqual(t, test.BillingPayments, missingEvent)

			result, err := pipeline.RunPipelineExportDiagram(missingEvent, "billing.emod")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)
			require.NotEmpty(t, envelope.Diagnostics, "the model must report something, or this cannot tell parsed from clean")
			require.True(t, envelope.Parsed)
		})

		t.Run("every key the viewer reads off a parse's answer is one the answer carries", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram(test.BillingPayments, "billing.emod")
			require.NoError(t, err)
			var answer map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(result, &answer))
			written := make([]string, 0, len(answer))
			for key := range answer {
				written = append(written, key)
			}

			read := fieldsReadBy(t, "../frontend/static/viewer.js", "renderPanelSource", "data")

			require.Contains(t, read, "parsed", "viewer.js must read whether the source parsed")
			require.Subset(t, written, read, "viewer.js reads a key of the parse answer the pipeline does not write")
		})

		t.Run("reports each diagnostic under the file name it was given", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram("foobar {\n}\n", "orders.emod")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)

			require.NotEmpty(t, envelope.Diagnostics)
			for _, d := range envelope.Diagnostics {
				require.Equal(t, "orders.emod", d.File, d.Message)
			}
		})

		t.Run("places each node in the file name it was given", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportDiagram(test.BillingPayments, "orders.emod")
			require.NoError(t, err)

			envelope := decodeDiagramEnvelope(t, result)

			placed := map[string]string{}
			for _, n := range envelope.Diagram.Nodes {
				if n.Type == "command" || n.Type == "event" {
					placed[n.Label] = n.Position.Filename
				}
			}
			require.Equal(t, map[string]string{"TakePayment": "orders.emod", "PaymentTaken": "orders.emod"}, placed)
		})

		t.Run("reports for every example what emod validate reports for that file", func(t *testing.T) {
			reported := 0
			for _, path := range examplePaths(t) {
				source, err := os.ReadFile(path)
				require.NoError(t, err)

				answer := pipeline.RunOnSource(requestFor(t, string(source), filepath.Base(path)), pipeline.RunPipelineExportDiagram)
				envelope := decodeDiagramEnvelope(t, []byte(answer))

				viewer := make([]reportedDiagnostic, 0, len(envelope.Diagnostics))
				for _, d := range envelope.Diagnostics {
					viewer = append(viewer, reportedDiagnostic{Line: d.Line, Rule: d.RuleName, Severity: d.Severity, Message: d.Message})
				}
				require.Equal(t, validateReports(t, path), viewer, path)
				reported += len(viewer)
			}
			require.NotZero(t, reported, "at least one example must report something, or the comparison proves nothing")
		})
	})

	t.Run("export model json", func(t *testing.T) {
		t.Run("returns the parsed model with no diagnostics", func(t *testing.T) {
			result, err := pipeline.RunPipelineExportJSON(test.BillingPayments, "billing.emod")
			require.NoError(t, err)

			var envelope struct {
				Diagnostics []diagnostic `json:"diagnostics"`
				Model       struct {
					Name string `json:"name"`
				} `json:"model"`
			}
			require.NoError(t, json.Unmarshal(result, &envelope))

			require.Empty(t, envelope.Diagnostics)
			require.Equal(t, "Billing", envelope.Model.Name)
		})
	})

	t.Run("apply diagram", func(t *testing.T) {
		t.Run("answers the edited source beside what a parse of that source answers", func(t *testing.T) {
			diagram := drawnDiagram(t, test.BillingPayments, "billing.emod", map[string]any{
				"id": "_cmd_1", "type": "command", "label": "RefundPayment", "parentId": "slice-1",
			})

			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, test.BillingPayments, "billing.emod", diagram)))

			edited := strings.Replace(test.BillingPayments, `      event "PaymentTaken" {`, `      command "RefundPayment" {
      }

      event "PaymentTaken" {`, 1)
			require.Equal(t, edited, answer.Emod)
			require.Contains(t, string(answer.Diagram), `"label":"RefundPayment"`)

			parsed := decodeApplyAnswer(t, pipeline.RunOnSource(requestFor(t, edited, "billing.emod"), pipeline.RunPipelineExportDiagram))
			require.JSONEq(t, string(parsed.Diagnostics), string(answer.Diagnostics))
			require.JSONEq(t, string(parsed.Diagram), string(answer.Diagram))
			require.Equal(t, parsed.Parsed, answer.Parsed)
			require.False(t, answer.Reformatted)
			require.Zero(t, answer.DroppedComments)
		})

		t.Run("says a source not in the emod fmt layout was reformatted, and counts the comments that layout drops", func(t *testing.T) {
			source := strings.Replace(test.BillingPayments, "      FLOW\n    }\n", "      FLOW\n      # after the slice's last entry\n    }\n", 1) +
				"# at the end of the file\n"
			require.Contains(t, source, "# after the slice's last entry")

			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, source, "billing.emod", drawnDiagram(t, source, "billing.emod"))))

			require.Equal(t, test.BillingPayments, answer.Emod)
			require.True(t, answer.Reformatted)
			require.Equal(t, 2, answer.DroppedComments)
		})

		t.Run("says a source was reformatted with no comment dropped when the layout keeps every comment", func(t *testing.T) {
			raw, err := os.ReadFile("../../examples/error_diagnostics_test.emod")
			require.NoError(t, err)
			source := string(raw)

			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, source, "errors.emod", drawnDiagram(t, source, "errors.emod"))))

			require.Empty(t, answer.Error)
			require.True(t, answer.Reformatted)
			require.Zero(t, answer.DroppedComments)
		})

		t.Run("counts no comment dropped for the blank lines the layout takes out", func(t *testing.T) {
			source := strings.Replace(test.BillingPayments, "      }\n\n      event", "      }\n\n\n\n      event", 1)
			require.NotEqual(t, test.BillingPayments, source)

			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, source, "billing.emod", drawnDiagram(t, source, "billing.emod"))))

			require.True(t, answer.Reformatted)
			require.Zero(t, answer.DroppedComments)
		})

		t.Run("counts a comment on a line of a flow block, which the emod fmt layout drops", func(t *testing.T) {
			source := strings.Replace(test.BillingPayments, "      flow = <<-FLOW\n", "      flow = <<-FLOW\n        # the card is charged at once\n\n", 1)
			require.NotEqual(t, test.BillingPayments, source)

			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, source, "billing.emod", drawnDiagram(t, source, "billing.emod"))))

			require.Equal(t, test.BillingPayments, answer.Emod)
			require.True(t, answer.Reformatted)
			require.Equal(t, 1, answer.DroppedComments)
		})

		t.Run("does not count a # line of a description written after a flow block", func(t *testing.T) {
			source := `emod = 1

model "Billing" {
}

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
      command "TakePayment" {
        fields {
          amount = required(int)
        }
      }

      flow = <<-FLOW
        command -> event:    TakePayment -> PaymentTaken
      FLOW

      event "PaymentTaken" {
        description = <<-EOT
          # charged in full
        EOT

        fields {
          amount = required(int)
        }
      }
    }
  }
}
`

			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, source, "billing.emod", drawnDiagram(t, source, "billing.emod"))))

			require.Empty(t, answer.Error)
			require.Contains(t, answer.Emod, "# charged in full")
			require.True(t, answer.Reformatted)
			require.Zero(t, answer.DroppedComments)
		})

		t.Run("does not count the comments of a construct the edit deletes as dropped", func(t *testing.T) {
			source := strings.Replace(test.BillingPayments, `      command "TakePayment" {`, "      # Take the money\n      command \"TakePayment\" {", 1)

			diagram := withoutNode(t, drawnDiagram(t, source, "billing.emod"), "command-1")
			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, source, "billing.emod", diagram)))

			require.Empty(t, answer.Error)
			require.Equal(t, `emod = 1

model "Billing" {
}

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
      event "PaymentTaken" {
        fields {
          amount = required(int)
        }
      }
    }
  }
}
`, answer.Emod)
			require.False(t, answer.Reformatted)
			require.Zero(t, answer.DroppedComments)
		})

		t.Run("places the edited source in input.emod when the envelope names no file", func(t *testing.T) {
			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, test.BillingPayments, "", drawnDiagram(t, test.BillingPayments, "input.emod"))))

			require.Contains(t, string(answer.Diagram), `"filename":"input.emod"`)
			require.NotContains(t, string(answer.Diagram), `"filename":""`)
		})

		t.Run("carries non-ASCII text in labels, descriptions and comments through unchanged", func(t *testing.T) {
			source := `emod = 1

# Zahlungen — über alles ✓
model "Café Billing" {
  description = "Größe zählt"
}

context "Payments" {
  slice "Take Payment" {
    command "TakePayment" {
    }
  }
}
`
			diagram := withLabel(t, drawnDiagram(t, source, "café.emod"), "slice-1", "Zahlung annehmen ✓")

			answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(applyRequest(t, source, "café.emod", diagram)))

			require.Equal(t, strings.Replace(source, `slice "Take Payment" {`, `slice "Zahlung annehmen ✓" {`, 1), answer.Emod)
		})

		for _, refusal := range []struct {
			name, request, reason string
		}{
			{"an envelope that is not JSON", `not json`, "invalid JSON"},
			{"an envelope with no source", `{"diagram": {"nodes": [], "edges": []}}`, "missing source field"},
			{"an envelope with an empty source", `{"source": "", "diagram": {"nodes": [], "edges": []}}`, "missing source field"},
			{"an envelope with no diagram", `{"source": "model \"M\" {\n}\n"}`, "missing diagram field"},
			{"an envelope with a null diagram", `{"source": "model \"M\" {\n}\n", "diagram": null}`, "missing diagram field"},
			{"a diagram that is not a document", `{"source": "model \"M\" {\n}\n", "diagram": "a picture"}`, "invalid diagram JSON"},
			{"a source that does not parse", `{"source": "model \"M\" {\n", "filename": "orders.emod", "diagram": {"nodes": [], "edges": []}}`,
				"the diagram edit cannot be applied: orders.emod does not parse"},
		} {
			t.Run("refuses "+refusal.name+" with the reason", func(t *testing.T) {
				answer := decodeApplyAnswer(t, pipeline.ApplyDiagramJSON(refusal.request))

				require.Contains(t, answer.Error, refusal.reason)
				require.Empty(t, answer.Emod)
			})
		}

		t.Run("every key the viewer reads off an applied edit's answer is one the answer carries", func(t *testing.T) {
			answer := pipeline.ApplyDiagramJSON(applyRequest(t, test.BillingPayments, "billing.emod", drawnDiagram(t, test.BillingPayments, "billing.emod")))
			var keys map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(answer), &keys))
			written := make([]string, 0, len(keys))
			for key := range keys {
				written = append(written, key)
			}

			read := fieldsReadBy(t, "../frontend/static/viewer.js", "writeAppliedSource", "answer")

			require.Contains(t, read, "emod", "viewer.js must read the applied source")
			require.Subset(t, written, read, "viewer.js reads a key of the applied edit's answer the pipeline does not write")
		})

		t.Run("both viewer runtimes send the envelope under the keys it reads", func(t *testing.T) {
			for _, platform := range []string{
				"../frontend/static/platform.browser.js",
				"../frontend/desktop/platform.desktop.js",
			} {
				require.Equal(t, []string{"diagram", "filename", "source"}, keysSentBy(t, platform, "applyDiagram"), platform)
			}
		})
	})

	t.Run("documentation", func(t *testing.T) {
		t.Run("the WASM architecture page names every function the WASM module registers", func(t *testing.T) {
			main, err := os.ReadFile("../../cmd/emod-wasm/main.go")
			require.NoError(t, err)
			registered := regexp.MustCompile(`js\.Global\(\)\.Set\("(\w+)"`).FindAllStringSubmatch(string(main), -1)
			require.NotEmpty(t, registered, "cmd/emod-wasm/main.go must register functions on the global object")
			page, err := os.ReadFile("../../docs/wasm-architecture.md")
			require.NoError(t, err)
			build := strings.Split(strings.Split(string(page), "1. **Build**")[1], "2. **Serve**")[0]

			for _, name := range registered {
				require.Contains(t, build, "`"+name[1]+"`", "the Build step must name %s", name[1])
			}
		})
	})

	t.Run("export emod", func(t *testing.T) {
		t.Run("diagram JSON round-trips back to the source it was parsed from", func(t *testing.T) {
			parsed, err := pipeline.RunPipelineExportDiagram(test.BillingPayments, "billing.emod")
			require.NoError(t, err)

			var envelope struct {
				Diagram json.RawMessage `json:"diagram"`
			}
			require.NoError(t, json.Unmarshal(parsed, &envelope))

			result, err := pipeline.ExportEmod(string(envelope.Diagram))
			require.NoError(t, err)

			require.Equal(t, test.BillingPayments, string(result))
		})

		t.Run("malformed diagram JSON returns error", func(t *testing.T) {
			_, err := pipeline.ExportEmod(`not json`)

			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid diagram JSON")
		})
	})

	t.Run("export emod as json", func(t *testing.T) {
		t.Run("wraps the formatted source in an emod field", func(t *testing.T) {
			parsed := decodeEmodEnvelope(t, pipeline.ExportEmodJSON(`{"model_name":"Billing","nodes":[],"edges":[]}`))

			require.Empty(t, parsed.Error)
			require.Equal(t, "emod = 1\n\nmodel \"Billing\" {\n}\n", parsed.Emod)
		})

		t.Run("reports a failure in an error field instead of an emod field", func(t *testing.T) {
			parsed := decodeEmodEnvelope(t, pipeline.ExportEmodJSON(`not json`))

			require.Empty(t, parsed.Emod)
			require.Contains(t, parsed.Error, "invalid diagram JSON")
		})
	})

	t.Run("error json", func(t *testing.T) {
		t.Run("carries the message in an error field", func(t *testing.T) {
			var parsed struct {
				Error string `json:"error"`
			}
			require.NoError(t, json.Unmarshal([]byte(pipeline.ErrorJSON("something went wrong")), &parsed))

			require.Equal(t, "something went wrong", parsed.Error)
		})
	})
}

type diagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	RuleName string `json:"rule_name"`
}

type node struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Label    string `json:"label"`
	Position struct {
		Filename string `json:"filename"`
	} `json:"position"`
}

type edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

type diagramEnvelope struct {
	Parsed      bool         `json:"parsed"`
	Diagnostics []diagnostic `json:"diagnostics"`
	Diagram     struct {
		ModelName string `json:"model_name"`
		Nodes     []node `json:"nodes"`
		Edges     []edge `json:"edges"`
	} `json:"diagram"`
}

func decodeDiagramEnvelope(t *testing.T, raw []byte) diagramEnvelope {
	t.Helper()

	var envelope diagramEnvelope
	require.NoError(t, json.Unmarshal(raw, &envelope), "pipeline must return valid JSON")

	return envelope
}

func decodeEmodEnvelope(t *testing.T, raw string) struct {
	Emod  string `json:"emod"`
	Error string `json:"error"`
} {
	t.Helper()

	var parsed struct {
		Emod  string `json:"emod"`
		Error string `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	return parsed
}

// labelsOfType returns the labels of nodes matching any of the given types, in
// document order.
func labelsOfType(nodes []node, types ...string) []string {
	wanted := make(map[string]bool, len(types))
	for _, t := range types {
		wanted[t] = true
	}

	var labels []string
	for _, n := range nodes {
		if wanted[n.Type] {
			labels = append(labels, n.Label)
		}
	}
	return labels
}

// reportedDiagnostic is what `emod validate --format json` reports of a
// diagnostic apart from the file, which the CLI names by the path it was handed.
type reportedDiagnostic struct {
	Line     int    `json:"line"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func validateReports(t *testing.T, path string) []reportedDiagnostic {
	t.Helper()

	output := captureStdout(t, func() {
		_ = cli.RunValidate(path, "json")
	})

	reports := []reportedDiagnostic{}
	require.NoError(t, json.Unmarshal([]byte(output), &reports), path)

	return reports
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	return buf.String()
}

func examplePaths(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir("../../examples")
	require.NoError(t, err)

	var paths []string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".emod" {
			paths = append(paths, filepath.Join("../../examples", entry.Name()))
		}
	}
	require.NotEmpty(t, paths)

	return paths
}

type applyAnswer struct {
	Error           string          `json:"error"`
	Emod            string          `json:"emod"`
	Diagnostics     json.RawMessage `json:"diagnostics"`
	Diagram         json.RawMessage `json:"diagram"`
	Parsed          bool            `json:"parsed"`
	Reformatted     bool            `json:"reformatted"`
	DroppedComments int             `json:"dropped_comments"`
}

func decodeApplyAnswer(t *testing.T, raw string) applyAnswer {
	t.Helper()

	var answer applyAnswer
	require.NoError(t, json.Unmarshal([]byte(raw), &answer), "the answer must be JSON")

	return answer
}

// drawnDiagram answers the diagram a parse of source draws, with added appended
// to its nodes the way the viewer adds one.
func drawnDiagram(t *testing.T, source, filename string, added ...map[string]any) map[string]any {
	t.Helper()

	var answer struct {
		Diagram map[string]any `json:"diagram"`
	}
	require.NoError(t, json.Unmarshal([]byte(pipeline.RunOnSource(requestFor(t, source, filename), pipeline.RunPipelineExportDiagram)), &answer))
	require.NotNil(t, answer.Diagram)

	for _, node := range added {
		answer.Diagram["nodes"] = append(answer.Diagram["nodes"].([]any), node)
	}

	return answer.Diagram
}

// withoutNode takes the node with id out of diagram, and every edge that
// touches it, as the viewer's delete does.
func withoutNode(t *testing.T, diagram map[string]any, id string) map[string]any {
	t.Helper()

	var nodes, edges []any
	for _, n := range diagram["nodes"].([]any) {
		if n.(map[string]any)["id"] != id {
			nodes = append(nodes, n)
		}
	}
	require.Less(t, len(nodes), len(diagram["nodes"].([]any)), "the diagram has to hold a node %s", id)
	for _, e := range diagram["edges"].([]any) {
		if edge := e.(map[string]any); edge["source"] != id && edge["target"] != id {
			edges = append(edges, e)
		}
	}
	diagram["nodes"], diagram["edges"] = nodes, edges

	return diagram
}

func withLabel(t *testing.T, diagram map[string]any, id, label string) map[string]any {
	t.Helper()

	for _, n := range diagram["nodes"].([]any) {
		if node := n.(map[string]any); node["id"] == id {
			node["label"] = label
			return diagram
		}
	}
	require.Failf(t, "no node", "the diagram has to hold a node %s", id)

	return diagram
}

func applyRequest(t *testing.T, source, filename string, diagram map[string]any) string {
	t.Helper()

	envelope := map[string]any{"source": source, "diagram": diagram}
	if filename != "" {
		envelope["filename"] = filename
	}
	raw, err := json.Marshal(envelope)
	require.NoError(t, err)

	return string(raw)
}

func requestFor(t *testing.T, source, filename string) string {
	t.Helper()

	raw, err := json.Marshal(pipeline.Request{Source: source, Filename: filename})
	require.NoError(t, err)

	return string(raw)
}

func jsonKeysOf(typ reflect.Type) []string {
	var keys []string
	for i := 0; i < typ.NumField(); i++ {
		keys = append(keys, strings.Split(typ.Field(i).Tag.Get("json"), ",")[0])
	}
	sort.Strings(keys)

	return keys
}

// keysSentBy returns the keys of the object literal the named function hands to
// JSON.stringify, read from the function's own body so a key sent anywhere else
// in the file cannot stand in for it.
func keysSentBy(t *testing.T, path, function string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	body := functionBody(t, string(raw), function, path)
	literal := regexp.MustCompile(`JSON\.stringify\(\{([^}]*)\}\)`).FindStringSubmatch(body)
	require.Len(t, literal, 2, path+"'s "+function+" must hand JSON.stringify an object literal")

	// Each entry's key leads it, written `key: value` or as the shorthand `key`.
	var keys []string
	for _, entry := range strings.Split(literal[1], ",") {
		if key := regexp.MustCompile(`^\s*(\w+)`).FindStringSubmatch(entry); key != nil {
			keys = append(keys, key[1])
		}
	}
	sort.Strings(keys)

	return keys
}

func fieldsReadBy(t *testing.T, path, function, receiver string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var fields []string
	seen := map[string]bool{}
	for _, match := range regexp.MustCompile(`\b`+receiver+`\.(\w+)`).FindAllStringSubmatch(functionBody(t, string(raw), function, path), -1) {
		if !seen[match[1]] {
			seen[match[1]] = true
			fields = append(fields, match[1])
		}
	}

	return fields
}

func functionBody(t *testing.T, source, function, path string) string {
	t.Helper()

	start := regexp.MustCompile(`function ` + function + `\([^)]*\)\s*\{`).FindStringIndex(source)
	require.NotNil(t, start, path+" must declare function "+function)

	depth := 1
	for i := start[1]; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[start[1]:i]
			}
		}
	}
	require.Fail(t, path+"'s "+function+" never closes")

	return ""
}
