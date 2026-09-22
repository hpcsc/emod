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
