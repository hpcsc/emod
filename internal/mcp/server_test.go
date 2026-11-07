//go:build unit

package mcp_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/examples"
	"github.com/hpcsc/emod/internal/cue"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/mcp"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/version"
)

const cleanSource = `emod = 1

model "Ordering" {
}

context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
      command "PlaceOrder" {
        fields {
          orderId    = required(string)
          customerId = required(string)
        }
      }

      event "OrderPlaced" {
        fields {
          orderId    = required(string)
          customerId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    PlaceOrder -> OrderPlaced
      FLOW
    }
  }
}
`

const warningSource = `emod = 1

model "Test" {
}

context "Orders" {
  aggregate "Order" {
    slice "Update Order" {
      command "PlaceOrder" {
        fields {
          orderId = required(string)
          reason  = required(string)
        }
      }

      event "OrderUpdated" {
        fields {
          orderId = required(string)
          reason  = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    PlaceOrder -> OrderUpdated
      FLOW
    }
  }
}
`

const brokenSource = `emod = 1

model "Broken" {
}

context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
      command "PlaceOrder" {
        fields {
          orderId = required(string)
        }
      }
    }
  }
}

foobar "Stray" {
}
`

const slicesSource = `emod = 1

model "Ordering" {
}

context "Orders" {
  aggregate "Order" {
    slice "Place Order" {
      trigger "Checkout" {
      }

      command "PlaceOrder" {
        fields {
          orderId = required(string)
        }
      }

      event "OrderPlaced" {
        fields {
          orderId = required(string)
        }
      }

      flow = <<-FLOW
        command -> event:    PlaceOrder -> OrderPlaced
      FLOW
    }

    slice "Show Orders" {
      view "OrdersView" {
        fields {
          orderId = required(string)
        }
      }
    }
  }
}

context "Stock" {
  mode = dcb

  slice "Reserve Stock" {
    command "ReserveStock" {
      fields {
        orderId = required(string)
      }
    }

    automation "StockReserver" {
      on      = OrderPlaced
      command = ReserveStock
    }
  }
}
`

func connect(t *testing.T) *sdk.ClientSession {
	t.Helper()

	return connectAt(t, t.TempDir())
}

func connectAt(t *testing.T, dir string) *sdk.ClientSession {
	t.Helper()

	root, err := mcp.NewRoot(dir)
	require.NoError(t, err)
	clientReads, serverWrites := io.Pipe()
	serverReads, clientWrites := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() {
		served <- mcp.NewServer(serverReads, serverWrites, root).Run(ctx)
	}()

	client := sdk.NewClient(&sdk.Implementation{Name: "test-host", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk.IOTransport{Reader: clientReads, Writer: clientWrites}, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		<-served
	})
	return session
}

func callCheck(t *testing.T, session *sdk.ClientSession, arguments map[string]any) *sdk.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "emod_check", Arguments: arguments})
	require.NoError(t, err)
	return result
}

func recordsOf(t *testing.T, result *sdk.CallToolResult) []diagnostic.Record {
	t.Helper()

	encoded, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var decoded struct {
		Diagnostics []diagnostic.Record `json:"diagnostics"`
	}
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	return decoded.Diagnostics
}

func callTool(t *testing.T, session *sdk.ClientSession, name string, arguments map[string]any) *sdk.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: arguments})
	require.NoError(t, err)
	return result
}

func objectOf(t *testing.T, value any) map[string]any {
	t.Helper()

	object, ok := value.(map[string]any)
	require.True(t, ok, "the value must be an object, but it is %T", value)
	return object
}

func keysOf(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	return keys
}

func jsonOf(t *testing.T, value any) string {
	t.Helper()

	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

func rulesIn(t *testing.T, diagnostics any) []string {
	t.Helper()

	var records []diagnostic.Record
	require.NoError(t, json.Unmarshal([]byte(jsonOf(t, diagnostics)), &records))
	rules := make([]string, 0, len(records))
	for _, record := range records {
		rules = append(rules, record.Rule)
	}
	return rules
}

func exportedRules(t *testing.T, diagnostics any) []string {
	t.Helper()

	var records []struct {
		RuleName string `json:"rule_name"`
	}
	require.NoError(t, json.Unmarshal([]byte(jsonOf(t, diagnostics)), &records))
	rules := make([]string, 0, len(records))
	for _, record := range records {
		rules = append(rules, record.RuleName)
	}
	return rules
}

func filesUnder(t *testing.T, dir string) map[string]string {
	t.Helper()

	files := map[string]string{}
	require.NoError(t, filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files[path] = info.ModTime().String()
		return nil
	}))
	return files
}

func callSlicesList(t *testing.T, session *sdk.ClientSession, arguments map[string]any) *sdk.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "emod_slices_list", Arguments: arguments})
	require.NoError(t, err)
	return result
}

func callExplainRule(t *testing.T, session *sdk.ClientSession, arguments map[string]any) *sdk.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "emod_explain_rule", Arguments: arguments})
	require.NoError(t, err)
	return result
}

func toolNamed(t *testing.T, session *sdk.ClientSession, name string) *sdk.Tool {
	t.Helper()

	listed, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	for _, tool := range listed.Tools {
		if tool.Name == name {
			return tool
		}
	}
	require.FailNow(t, "the server must list the tool", "tool %s", name)
	return nil
}

func textOf(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()

	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*sdk.TextContent)
	require.True(t, ok, "the content must be a text block, but it is %T", result.Content[0])
	return text.Text
}

func TestServer(t *testing.T) {
	t.Run("initialize", func(t *testing.T) {
		t.Run("names itself emod, with the version that the version package reports", func(t *testing.T) {
			session := connect(t)

			info := session.InitializeResult().ServerInfo

			require.Equal(t, &sdk.Implementation{Name: "emod", Version: version.Current()}, info)
		})

		t.Run("offers tools and resources and no logging, because the server sends no log message", func(t *testing.T) {
			session := connect(t)

			capabilities := session.InitializeResult().Capabilities

			require.NotNil(t, capabilities.Tools)
			require.NotNil(t, capabilities.Resources)
			require.Nil(t, capabilities.Logging)
		})
	})

	t.Run("list tools", func(t *testing.T) {
		t.Run("lists each tool of the server, and no other tool", func(t *testing.T) {
			session := connect(t)

			listed, err := session.ListTools(context.Background(), nil)

			require.NoError(t, err)
			names := make([]string, 0, len(listed.Tools))
			for _, tool := range listed.Tools {
				names = append(names, tool.Name)
			}
			require.ElementsMatch(t, []string{"emod_check", "emod_explain_rule", "emod_export", "emod_glossary", "emod_schema", "emod_slices_list"}, names)
		})

		t.Run("gives emod_glossary the input of emod_check and a format of markdown or json, with markdown as the default", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_glossary")

			schema, ok := tool.InputSchema.(map[string]any)
			require.True(t, ok)
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok)
			require.ElementsMatch(t, []string{"content", "path", "name", "format"}, keysOf(properties))
			format, ok := properties["format"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, []any{"markdown", "json"}, format["enum"])
			require.Equal(t, "markdown", format["default"])
		})

		t.Run("the description of emod_glossary names the command with the same text and the result of a parse error", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_glossary")

			require.Contains(t, tool.Description, "emod glossary --format <format>")
			require.Contains(t, tool.Description, "A source that does not parse gives its parse diagnostics")
		})

		t.Run("gives emod_export the input of emod_check and a format of json, diagram-json or cue, with json as the default", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_export")

			schema, ok := tool.InputSchema.(map[string]any)
			require.True(t, ok)
			require.Nil(t, schema["required"])
			require.Equal(t, false, schema["additionalProperties"])
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok)
			require.ElementsMatch(t, []string{"content", "path", "name", "format"}, keysOf(properties))
			format, ok := properties["format"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "string", format["type"])
			require.Equal(t, []any{"json", "diagram-json", "cue"}, format["enum"])
			require.Equal(t, "json", format["default"])
			require.Contains(t, format["description"], "The default is json")
		})

		t.Run("the description of emod_export names the command with the same output and the rule of cue", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_export")

			require.Contains(t, tool.Description, "emod export --format <format>")
			require.Contains(t, tool.Description, "a source with any diagnostic, also a warning, gives the diagnostics")
		})

		t.Run("gives emod_slices_list content, path and name as optional strings and no other input", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_slices_list")

			schema, ok := tool.InputSchema.(map[string]any)
			require.True(t, ok)
			require.Nil(t, schema["required"])
			require.Equal(t, false, schema["additionalProperties"])
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok)
			require.Len(t, properties, 3)
			require.Equal(t, map[string]any{"type": "string"}, typeOf(properties["content"]))
			require.Equal(t, map[string]any{"type": "string"}, typeOf(properties["path"]))
			require.Equal(t, map[string]any{"type": "string"}, typeOf(properties["name"]))
		})

		t.Run("gives emod_slices_list an output schema of one of two objects: the slices, or the parse diagnostics", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_slices_list")

			schema, ok := tool.OutputSchema.(map[string]any)
			require.True(t, ok, "the tool must give an output schema, but it is %T", tool.OutputSchema)
			require.Equal(t, "object", schema["type"])
			choices, ok := schema["oneOf"].([]any)
			require.True(t, ok)
			var required [][]any
			for _, choice := range choices {
				fields, ok := choice.(map[string]any)
				require.True(t, ok)
				keys, _ := fields["required"].([]any)
				required = append(required, keys)
			}
			require.Equal(t, [][]any{{"slices"}, {"diagnostics"}}, required)
		})

		t.Run("the description of emod_slices_list names the command with the same records and the result of a parse error", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_slices_list")

			require.Contains(t, tool.Description, "emod slices list --format json")
			require.Contains(t, tool.Description, "command, view, automation, translation or unknown")
			require.Contains(t, tool.Description, "A source that does not parse gives its parse diagnostics")
		})

		t.Run("gives emod_schema no input", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_schema")

			schema, ok := tool.InputSchema.(map[string]any)
			require.True(t, ok)
			require.Equal(t, "object", schema["type"])
			require.Empty(t, schema["properties"])
			require.Equal(t, false, schema["additionalProperties"])
		})

		t.Run("the description of emod_schema names the command with the same text and the resource", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_schema")

			require.Contains(t, tool.Description, "emod schema")
			require.Contains(t, tool.Description, "emod://schema")
		})

		t.Run("gives emod_check content, path and name as optional strings and no other input", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_check")

			schema, ok := tool.InputSchema.(map[string]any)
			require.True(t, ok)
			require.Nil(t, schema["required"])
			require.Equal(t, false, schema["additionalProperties"])
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok)
			require.Len(t, properties, 3)
			require.Equal(t, map[string]any{"type": "string"}, typeOf(properties["content"]))
			require.Equal(t, map[string]any{"type": "string"}, typeOf(properties["path"]))
			require.Equal(t, map[string]any{"type": "string"}, typeOf(properties["name"]))
		})

		t.Run("describes each input of emod_check: the source, the path under the root, and the name of the records", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_check")

			schema, ok := tool.InputSchema.(map[string]any)
			require.True(t, ok)
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok)
			require.Contains(t, descriptionOf(properties["content"]), "The .emod source text")
			require.Contains(t, descriptionOf(properties["path"]), "under the root of the server")
			require.Contains(t, descriptionOf(properties["path"]), "only when the call gives no content")
			require.Contains(t, descriptionOf(properties["name"]), "The default is path, else inline.emod")
		})

		t.Run("the description of emod_check says that an empty list passes and that a warning is a finding to fix", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_check")

			require.Contains(t, tool.Description, "An empty list means that the source passes the check")
			require.Contains(t, tool.Description, "A warning is a finding to fix, not a failure of the tool")
			require.Contains(t, tool.Description, "When a call gives both, the tool checks content and does not read path")
		})

		t.Run("gives emod_explain_rule one required string, rule, and no other input", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_explain_rule")

			schema, ok := tool.InputSchema.(map[string]any)
			require.True(t, ok)
			require.Equal(t, []any{"rule"}, schema["required"])
			require.Equal(t, false, schema["additionalProperties"])
			properties, ok := schema["properties"].(map[string]any)
			require.True(t, ok)
			require.Len(t, properties, 1)
			require.Equal(t, map[string]any{"type": "string"}, typeOf(properties["rule"]))
			require.Contains(t, descriptionOf(properties["rule"]), "the rule key of a diagnostic record")
		})

		t.Run("the description of emod_explain_rule names the CLI command with the same text and the resource of all rules", func(t *testing.T) {
			session := connect(t)

			tool := toolNamed(t, session, "emod_explain_rule")

			require.Contains(t, tool.Description, "emod lint --explain")
			require.Contains(t, tool.Description, "emod://rules")
		})
	})

	t.Run("explain rule", func(t *testing.T) {
		t.Run("a rule name gives the rule and its description", func(t *testing.T) {
			session := connect(t)

			result := callExplainRule(t, session, map[string]any{"rule": "state-obsession"})

			require.False(t, result.IsError)
			description, err := linter.RuleDescription("state-obsession")
			require.NoError(t, err)
			require.Equal(t, map[string]any{"rule": "state-obsession", "description": description}, result.StructuredContent)
			require.Contains(t, description, `"OrderShipped" instead of "OrderUpdated"`)
		})

		t.Run("the text block holds the same rule and description, with angle brackets as they are", func(t *testing.T) {
			session := connect(t)

			result := callExplainRule(t, session, map[string]any{"rule": "property-sourcing"})

			description, err := linter.RuleDescription("property-sourcing")
			require.NoError(t, err)
			encoded, err := json.Marshal(map[string]string{"rule": "property-sourcing", "description": description})
			require.NoError(t, err)
			require.JSONEq(t, string(encoded), textOf(t, result))
			require.Contains(t, textOf(t, result), `<AggregateName><Field>Changed`)
		})

		t.Run("an unknown rule gives an error result that names the rule", func(t *testing.T) {
			session := connect(t)

			result := callExplainRule(t, session, map[string]any{"rule": "dcb/nonexistent"})

			require.True(t, result.IsError)
			require.Nil(t, result.StructuredContent)
			require.Equal(t, `unknown rule "dcb/nonexistent"`, textOf(t, result))
		})

		t.Run("an empty rule name gives an error result", func(t *testing.T) {
			session := connect(t)

			result := callExplainRule(t, session, map[string]any{"rule": ""})

			require.True(t, result.IsError)
			require.Equal(t, `unknown rule ""`, textOf(t, result))
		})

		t.Run("a call with no rule gives an error result", func(t *testing.T) {
			session := connect(t)

			result := callExplainRule(t, session, map[string]any{})

			require.True(t, result.IsError)
			require.Contains(t, textOf(t, result), "rule")
		})
	})

	t.Run("list slices", func(t *testing.T) {
		t.Run("gives one record for each slice in source order, for a slice in a dcb context and one under an aggregate", func(t *testing.T) {
			session := connect(t)

			result := callSlicesList(t, session, map[string]any{"content": slicesSource})

			require.False(t, result.IsError)
			require.Equal(t, map[string]any{"slices": []any{
				map[string]any{"name": "Place Order", "pattern": "command", "context": "Orders", "keyElements": "PlaceOrder, OrderPlaced"},
				map[string]any{"name": "Show Orders", "pattern": "view", "context": "Orders", "keyElements": "OrdersView"},
				map[string]any{"name": "Reserve Stock", "pattern": "automation", "context": "Stock", "keyElements": "OrderPlaced, ReserveStock"},
			}}, result.StructuredContent)
		})

		t.Run("the text block holds the same records, with an ampersand as it is", func(t *testing.T) {
			session := connect(t)

			result := callSlicesList(t, session, map[string]any{"content": strings.Replace(slicesSource, "Show Orders", "Show Orders & Returns", 1)})

			require.JSONEq(t, `{"slices":[`+
				`{"name":"Place Order","pattern":"command","context":"Orders","keyElements":"PlaceOrder, OrderPlaced"},`+
				`{"name":"Show Orders & Returns","pattern":"view","context":"Orders","keyElements":"OrdersView"},`+
				`{"name":"Reserve Stock","pattern":"automation","context":"Stock","keyElements":"OrderPlaced, ReserveStock"}]}`,
				textOf(t, result))
			require.Contains(t, textOf(t, result), "Show Orders & Returns")
		})

		t.Run("a model with no slice gives an empty list", func(t *testing.T) {
			session := connect(t)

			result := callSlicesList(t, session, map[string]any{"content": "emod = 1\n\nmodel \"Empty\" {\n}\n"})

			require.False(t, result.IsError)
			require.Equal(t, map[string]any{"slices": []any{}}, result.StructuredContent)
		})

		t.Run("a source that does not parse gives its parse diagnostics and no slices, as a normal result", func(t *testing.T) {
			session := connect(t)

			result := callSlicesList(t, session, map[string]any{"content": "emod = 1\n\nmodel \"Broken\" {\n", "name": "broken.emod"})

			require.False(t, result.IsError)
			structured, ok := result.StructuredContent.(map[string]any)
			require.True(t, ok)
			require.NotContains(t, structured, "slices")
			records, err := json.Marshal(structured["diagnostics"])
			require.NoError(t, err)
			var diagnostics []diagnostic.Record
			require.NoError(t, json.Unmarshal(records, &diagnostics))
			require.NotEmpty(t, diagnostics)
			for _, d := range diagnostics {
				require.Equal(t, "broken.emod", d.File)
				require.Equal(t, "error", d.Severity)
			}
		})

		t.Run("a source that does not parse gives only its parse diagnostics, not the findings of the validator on what the parser recovered", func(t *testing.T) {
			require.Contains(t, rulesOf(oracle.Check(brokenSource, "broken.emod")), "orphan-command",
				"the recovered model must have a validator finding, or the leaf cannot tell the two lists apart")
			session := connect(t)

			result := callSlicesList(t, session, map[string]any{"content": brokenSource, "name": "broken.emod"})

			require.False(t, result.IsError)
			require.Equal(t, map[string]any{"diagnostics": []any{
				map[string]any{
					"file":     "broken.emod",
					"line":     float64(18),
					"severity": "error",
					"message":  `unexpected "foobar" block; expected one of: model, actor, context`,
				},
			}}, result.StructuredContent)
		})

		t.Run("a source with a lint warning still gives its slices, because only a parse error stops the list", func(t *testing.T) {
			session := connect(t)

			result := callSlicesList(t, session, map[string]any{"content": warningSource})

			require.False(t, result.IsError)
			require.Equal(t, map[string]any{"slices": []any{
				map[string]any{"name": "Update Order", "pattern": "unknown", "context": "Orders", "keyElements": ""},
			}}, result.StructuredContent)
		})

		t.Run("a path under the root gives the slices of the file", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "models", "orders.emod"), slicesSource)
			session := connectAt(t, dir)

			result := callSlicesList(t, session, map[string]any{"path": "models/orders.emod"})

			require.False(t, result.IsError)
			structured, ok := result.StructuredContent.(map[string]any)
			require.True(t, ok)
			require.Len(t, structured["slices"], 3)
		})

		t.Run("a path outside the root gives an error result that names the root", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "orders.emod"), slicesSource)
			session := connectAt(t, project)

			result := callSlicesList(t, session, map[string]any{"path": "../orders.emod"})

			require.True(t, result.IsError)
			require.Contains(t, textOf(t, result), "is outside the root "+project)
		})
	})

	t.Run("export", func(t *testing.T) {
		t.Run("a call with no format gives the json document, with the diagnostics in it", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_export", map[string]any{"content": warningSource, "name": "ordering.emod"})

			require.False(t, result.IsError)
			document := objectOf(t, result.StructuredContent)
			require.ElementsMatch(t, []string{"diagnostics", "model"}, keysOf(document))
			require.Equal(t, "Test", objectOf(t, document["model"])["name"])
			require.Equal(t, []string{"state-obsession"}, exportedRules(t, document["diagnostics"]))
			require.JSONEq(t, jsonOf(t, result.StructuredContent), textOf(t, result))
		})

		t.Run("json gives the document also for a source with errors", func(t *testing.T) {
			source, err := os.ReadFile("../../examples/error_diagnostics_test.emod")
			require.NoError(t, err)
			session := connect(t)

			result := callTool(t, session, "emod_export", map[string]any{"content": string(source), "format": "json"})

			require.False(t, result.IsError)
			document := objectOf(t, result.StructuredContent)
			require.Contains(t, document, "model")
			require.Equal(t, []string{"", "orphan-command"}, exportedRules(t, document["diagnostics"]))
		})

		t.Run("diagram-json gives the diagram document, with the diagnostics in it", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_export", map[string]any{"content": warningSource, "format": "diagram-json"})

			require.False(t, result.IsError)
			document := objectOf(t, result.StructuredContent)
			require.ElementsMatch(t, []string{"diagnostics", "diagram"}, keysOf(document))
			require.Equal(t, []string{"state-obsession"}, exportedRules(t, document["diagnostics"]))
			require.Contains(t, jsonOf(t, document["diagram"]), "OrderUpdated")
		})

		t.Run("cue gives the CUE text of a clean source as one text block, with no structured content", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_export", map[string]any{"content": cleanSource, "format": "cue"})

			require.False(t, result.IsError)
			require.Nil(t, result.StructuredContent)
			require.Contains(t, textOf(t, result), `name: "Ordering"`)
			require.Contains(t, textOf(t, result), `name: "PlaceOrder"`)
		})

		t.Run("cue gives the diagnostics and no CUE for a source with only a warning", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_export", map[string]any{"content": warningSource, "format": "cue", "name": "ordering.emod"})

			require.False(t, result.IsError)
			require.Equal(t, []string{"diagnostics"}, keysOf(objectOf(t, result.StructuredContent)))
			require.Equal(t, []string{"state-obsession"}, rulesIn(t, objectOf(t, result.StructuredContent)["diagnostics"]))
			require.NotContains(t, textOf(t, result), `name: "Test"`)
		})

		t.Run("an unknown format gives an error result that names the three formats", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_export", map[string]any{"content": cleanSource, "format": "yaml"})

			require.True(t, result.IsError)
			require.Contains(t, textOf(t, result), "json")
			require.Contains(t, textOf(t, result), "diagram-json")
			require.Contains(t, textOf(t, result), "cue")
		})

		t.Run("a model that the export cannot encode gives an error result that names the value", func(t *testing.T) {
			source := strings.Replace(cleanSource, `      flow = <<-FLOW`, `      spec "places an order for a party" {
        when = PlaceOrder({ orderId = -007 })
        then = [OrderPlaced({ orderId = -007 })]
      }

      flow = <<-FLOW`, 1)
			session := connect(t)

			result := callTool(t, session, "emod_export", map[string]any{"content": source, "format": "json"})

			require.True(t, result.IsError)
			require.Contains(t, textOf(t, result), `"-007"`)
		})

		t.Run("a path outside the root gives an error result that names the root", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "ordering.emod"), cleanSource)
			session := connectAt(t, project)

			result := callTool(t, session, "emod_export", map[string]any{"path": "../ordering.emod"})

			require.True(t, result.IsError)
			require.Contains(t, textOf(t, result), "is outside the root "+project)
		})

		t.Run("a call with a path writes no file under the root, in any format", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "ordering.emod"), cleanSource)
			before := filesUnder(t, dir)
			session := connectAt(t, dir)

			for _, format := range []string{"json", "diagram-json", "cue"} {
				result := callTool(t, session, "emod_export", map[string]any{"path": "ordering.emod", "format": format})
				require.False(t, result.IsError, "format %s", format)
			}

			require.Equal(t, before, filesUnder(t, dir))
		})
	})

	t.Run("glossary", func(t *testing.T) {
		t.Run("a call with no format gives the Markdown glossary as one text block, with no structured content", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_glossary", map[string]any{"content": cleanSource})

			require.False(t, result.IsError)
			require.Nil(t, result.StructuredContent)
			require.True(t, strings.HasPrefix(textOf(t, result), "# Ordering\n"), "the glossary must start with the model: %q", textOf(t, result))
			require.Contains(t, textOf(t, result), "PlaceOrder")
			require.Contains(t, textOf(t, result), "OrderPlaced")
		})

		t.Run("json gives the glossary document as the structured content and the text block", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_glossary", map[string]any{"content": cleanSource, "format": "json"})

			require.False(t, result.IsError)
			document := objectOf(t, result.StructuredContent)
			require.Equal(t, "Ordering", document["name"])
			require.Contains(t, jsonOf(t, document["contexts"]), `"name":"PlaceOrder"`)
			require.JSONEq(t, jsonOf(t, result.StructuredContent), textOf(t, result))
		})

		t.Run("a source that does not parse gives only its parse diagnostics and no glossary", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_glossary", map[string]any{"content": brokenSource, "name": "broken.emod"})

			require.False(t, result.IsError)
			require.Equal(t, map[string]any{"diagnostics": []any{
				map[string]any{
					"file":     "broken.emod",
					"line":     float64(18),
					"severity": "error",
					"message":  `unexpected "foobar" block; expected one of: model, actor, context`,
				},
			}}, result.StructuredContent)
		})

		t.Run("an unknown format gives an error result that names the two formats", func(t *testing.T) {
			session := connect(t)

			result := callTool(t, session, "emod_glossary", map[string]any{"content": cleanSource, "format": "html"})

			require.True(t, result.IsError)
			require.Contains(t, textOf(t, result), "markdown")
			require.Contains(t, textOf(t, result), "json")
		})

		t.Run("a path outside the root gives an error result that names the root", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			writeFile(t, filepath.Join(base, "ordering.emod"), cleanSource)
			session := connectAt(t, project)

			result := callTool(t, session, "emod_glossary", map[string]any{"path": "../ordering.emod"})

			require.True(t, result.IsError)
			require.Contains(t, textOf(t, result), "is outside the root "+project)
		})

		t.Run("a call with a path writes no file under the root, in any format", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "ordering.emod"), cleanSource)
			before := filesUnder(t, dir)
			session := connectAt(t, dir)

			for _, format := range []string{"markdown", "json"} {
				result := callTool(t, session, "emod_glossary", map[string]any{"path": "ordering.emod", "format": format})
				require.False(t, result.IsError, "format %s", format)
			}

			require.Equal(t, before, filesUnder(t, dir))
		})
	})

	t.Run("list resources", func(t *testing.T) {
		t.Run("lists the rules, the schema, the three examples and the DSL reference, and no other resource", func(t *testing.T) {
			session := connect(t)

			listed, err := session.ListResources(context.Background(), nil)

			require.NoError(t, err)
			uris := make([]string, 0, len(listed.Resources))
			for _, resource := range listed.Resources {
				uris = append(uris, resource.URI)
			}
			require.ElementsMatch(t, []string{
				"emod://rules",
				"emod://schema",
				"emod://examples/all_patterns",
				"emod://examples/dcb_model",
				"emod://examples/specs_hotel",
				"emod://docs/dsl-reference",
			}, uris)
		})

		t.Run("lists each example as text, with the description of the example", func(t *testing.T) {
			session := connect(t)

			for _, example := range examples.All() {
				resource := resourceAt(t, session, "emod://examples/"+example.Name)

				require.Equal(t, example.Name+".emod", resource.Name, "example %s", example.Name)
				require.Equal(t, "Example "+example.Name, resource.Title, "example %s", example.Name)
				require.Equal(t, "text/plain", resource.MIMEType, "example %s", example.Name)
				require.Equal(t, example.Description, resource.Description, "example %s", example.Name)
			}
		})

		t.Run("lists the DSL reference as Markdown, with a description", func(t *testing.T) {
			session := connect(t)

			resource := resourceAt(t, session, "emod://docs/dsl-reference")

			require.Equal(t, "dsl-reference.md", resource.Name)
			require.Equal(t, "DSL reference of emod", resource.Title)
			require.Equal(t, "text/markdown", resource.MIMEType)
			require.Contains(t, resource.Description, "slice patterns")
		})

		t.Run("lists emod://rules as JSON, with a description", func(t *testing.T) {
			session := connect(t)

			resource := resourceAt(t, session, "emod://rules")

			require.Equal(t, "rules", resource.Name)
			require.Equal(t, "Lint rules of emod", resource.Title)
			require.Equal(t, "application/json", resource.MIMEType)
			require.Contains(t, resource.Description, "Each lint rule")
		})

		t.Run("lists emod://schema as text, with a description that names the command with the same text", func(t *testing.T) {
			session := connect(t)

			resource := resourceAt(t, session, "emod://schema")

			require.Equal(t, "schema", resource.Name)
			require.Equal(t, "CUE schema of emod", resource.Title)
			require.Equal(t, "text/plain", resource.MIMEType)
			require.Contains(t, resource.Description, "emod schema")
		})
	})

	t.Run("schema", func(t *testing.T) {
		t.Run("gives the CUE schema as one text block, with no structured content", func(t *testing.T) {
			session := connect(t)

			result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "emod_schema"})

			require.NoError(t, err)
			require.False(t, result.IsError)
			require.Nil(t, result.StructuredContent)
			require.Equal(t, cue.Schema, textOf(t, result))
			require.Contains(t, textOf(t, result), "#Command: {")
		})
	})

	t.Run("read schema", func(t *testing.T) {
		t.Run("gives the CUE schema as text", func(t *testing.T) {
			session := connect(t)

			read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://schema"})

			require.NoError(t, err)
			require.Len(t, read.Contents, 1)
			require.Equal(t, "text/plain", read.Contents[0].MIMEType)
			require.Equal(t, cue.Schema, read.Contents[0].Text)
		})
	})

	t.Run("read examples", func(t *testing.T) {
		for _, name := range []string{"all_patterns", "dcb_model", "specs_hotel"} {
			t.Run("gives the text of the file of "+name, func(t *testing.T) {
				source, err := os.ReadFile("../../examples/" + name + ".emod")
				require.NoError(t, err)
				session := connect(t)

				read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://examples/" + name})

				require.NoError(t, err)
				require.Len(t, read.Contents, 1)
				require.Equal(t, "text/plain", read.Contents[0].MIMEType)
				require.Equal(t, string(source), read.Contents[0].Text)
			})

			t.Run("gives a text of "+name+" that passes emod_check", func(t *testing.T) {
				session := connect(t)
				read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://examples/" + name})
				require.NoError(t, err)
				require.Len(t, read.Contents, 1)

				result := callCheck(t, session, map[string]any{"content": read.Contents[0].Text, "name": name + ".emod"})

				require.False(t, result.IsError)
				require.Equal(t, map[string]any{"diagnostics": []any{}}, result.StructuredContent)
			})
		}

		t.Run("does not serve the example that has errors on purpose", func(t *testing.T) {
			session := connect(t)

			_, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://examples/error_diagnostics_test"})

			require.ErrorContains(t, err, "not found")
		})
	})

	t.Run("read DSL reference", func(t *testing.T) {
		t.Run("gives the text of docs/dsl-reference.md as Markdown", func(t *testing.T) {
			reference, err := os.ReadFile("../../docs/dsl-reference.md")
			require.NoError(t, err)
			session := connect(t)

			read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://docs/dsl-reference"})

			require.NoError(t, err)
			require.Len(t, read.Contents, 1)
			require.Equal(t, "text/markdown", read.Contents[0].MIMEType)
			require.Equal(t, string(reference), read.Contents[0].Text)
		})
	})

	t.Run("read rules", func(t *testing.T) {
		t.Run("gives each rule that the linter lists, in the order of the names, with its description", func(t *testing.T) {
			session := connect(t)

			rules := readRules(t, session)

			expected := make([]listedRule, 0, len(rules))
			for _, r := range linter.Rules() {
				expected = append(expected, listedRule{Rule: r.Name, Description: r.Description})
			}
			require.Equal(t, expected, rules)
			names := make([]string, 0, len(rules))
			for _, r := range rules {
				names = append(names, r.Rule)
			}
			require.Subset(t, names, []string{"dcb/orphan-tag-key", "orphan-command", "spec/given-outside-boundary"})
		})

		t.Run("gives one text block for the URI, as JSON", func(t *testing.T) {
			session := connect(t)

			read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://rules"})

			require.NoError(t, err)
			require.Len(t, read.Contents, 1)
			require.Equal(t, "emod://rules", read.Contents[0].URI)
			require.Equal(t, "application/json", read.Contents[0].MIMEType)
			require.True(t, json.Valid([]byte(read.Contents[0].Text)))
		})

		t.Run("keeps angle brackets as they are, so that the text reads as the CLI prints it", func(t *testing.T) {
			session := connect(t)

			read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://rules"})

			require.NoError(t, err)
			require.Len(t, read.Contents, 1)
			require.Contains(t, read.Contents[0].Text, `<AggregateName><Field>Changed`)
		})
	})

	t.Run("check", func(t *testing.T) {
		t.Run("a clean source gives an empty list and no error", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": cleanSource})

			require.False(t, result.IsError)
			require.Equal(t, map[string]any{"diagnostics": []any{}}, result.StructuredContent)
		})

		t.Run("a source with errors gives each error as a record, not as an error result", func(t *testing.T) {
			source, err := os.ReadFile("../../examples/error_diagnostics_test.emod")
			require.NoError(t, err)
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": string(source), "name": "hotel.emod"})

			require.False(t, result.IsError)
			require.Equal(t, []diagnostic.Record{
				{File: "hotel.emod", Line: 43, Severity: "error", Message: `event "GuestCheckedOut" does not exist`},
				{File: "hotel.emod", Line: 64, Rule: "orphan-command", Severity: "error", Message: `command "SendEmail" is orphaned (no flow references it)`},
			}, recordsOf(t, result))
		})

		t.Run("a source with only warnings gives each warning as a record, not as an error result", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": warningSource, "name": "ordering.emod"})

			require.False(t, result.IsError)
			records := recordsOf(t, result)
			require.Len(t, records, 1)
			require.Equal(t, "ordering.emod", records[0].File)
			require.Equal(t, "state-obsession", records[0].Rule)
			require.Equal(t, "warning", records[0].Severity)
			require.Contains(t, records[0].Message, "OrderUpdated")
		})

		t.Run("the text block holds the same records, with no rule key on a record that has no rule", func(t *testing.T) {
			source, err := os.ReadFile("../../examples/error_diagnostics_test.emod")
			require.NoError(t, err)
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": string(source), "name": "hotel.emod"})

			require.JSONEq(t,
				`{"diagnostics":[{"file":"hotel.emod","line":43,"severity":"error","message":"event \"GuestCheckedOut\" does not exist"},`+
					`{"file":"hotel.emod","line":64,"rule":"orphan-command","severity":"error","message":"command \"SendEmail\" is orphaned (no flow references it)"}]}`,
				textOf(t, result))
		})

		t.Run("an empty source is checked like an empty file and gives an empty list", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": ""})

			require.False(t, result.IsError)
			require.Equal(t, map[string]any{"diagnostics": []any{}}, result.StructuredContent)
		})

		t.Run("a call with neither content nor path gives an error result that names both", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"name": "ordering.emod"})

			require.True(t, result.IsError)
			require.Nil(t, result.StructuredContent)
			require.Equal(t, "the call needs content or path", textOf(t, result))
		})

		t.Run("a path under the root gives the records of the file, each with the path as its file", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "models", "ordering.emod"), warningSource)
			session := connectAt(t, dir)

			result := callCheck(t, session, map[string]any{"path": "models/ordering.emod"})

			require.False(t, result.IsError)
			records := recordsOf(t, result)
			require.Len(t, records, 1)
			require.Equal(t, "models/ordering.emod", records[0].File)
			require.Equal(t, "state-obsession", records[0].Rule)
		})

		t.Run("a call with path and name and no content gives its records the path, because name applies only to content", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "models", "ordering.emod"), warningSource)
			session := connectAt(t, dir)

			result := callCheck(t, session, map[string]any{"path": "models/ordering.emod", "name": "draft.emod"})

			require.Equal(t, []string{"models/ordering.emod"}, filesOf(recordsOf(t, result)))
		})

		t.Run("a call with path and content checks content, does not read path, and gives its records the path", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "models", "new.emod"), cleanSource)
			session := connectAt(t, dir)

			result := callCheck(t, session, map[string]any{"content": warningSource, "path": "models/new.emod"})

			require.False(t, result.IsError)
			records := recordsOf(t, result)
			require.Len(t, records, 1)
			require.Equal(t, "models/new.emod", records[0].File)
			require.Equal(t, "state-obsession", records[0].Rule)
		})

		t.Run("a call with content and a path outside the root checks content, and gives its records the path", func(t *testing.T) {
			outside := filepath.Join(t.TempDir(), "drafts", "ordering.emod")
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": warningSource, "path": outside})

			require.False(t, result.IsError)
			require.Equal(t, []string{outside}, filesOf(recordsOf(t, result)))
		})

		t.Run("a call with name, path and content gives the records of content the name", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": warningSource, "path": "models/new.emod", "name": "draft.emod"})

			require.Equal(t, []string{"draft.emod"}, filesOf(recordsOf(t, result)))
		})

		t.Run("a path outside the root gives an error result that names the root", func(t *testing.T) {
			base := t.TempDir()
			project := filepath.Join(base, "project")
			require.NoError(t, os.Mkdir(project, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(base, "secret.emod"), []byte("SECRET-TEXT"), 0o644))
			session := connectAt(t, project)

			result := callCheck(t, session, map[string]any{"path": "../secret.emod"})

			require.True(t, result.IsError)
			require.Nil(t, result.StructuredContent)
			require.Contains(t, textOf(t, result), "is outside the root "+project)
			require.NotContains(t, textOf(t, result), "SECRET-TEXT")
		})

		t.Run("a path that does not exist gives an error result that names the path and the cause", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"path": "missing.emod"})

			require.True(t, result.IsError)
			require.Equal(t, "reading missing.emod: no such file or directory", textOf(t, result))
		})

		t.Run("each record carries inline.emod when the call gives no name", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": warningSource})

			require.Equal(t, []string{"inline.emod"}, filesOf(recordsOf(t, result)))
		})

		t.Run("each record carries inline.emod when the name is empty", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": warningSource, "name": ""})

			require.Equal(t, []string{"inline.emod"}, filesOf(recordsOf(t, result)))
		})

		t.Run("a name with characters outside ASCII comes back unchanged", func(t *testing.T) {
			session := connect(t)

			result := callCheck(t, session, map[string]any{"content": warningSource, "name": "café/commandes.emod"})

			require.Equal(t, []string{"café/commandes.emod"}, filesOf(recordsOf(t, result)))
		})
	})
}

func resourceAt(t *testing.T, session *sdk.ClientSession, uri string) *sdk.Resource {
	t.Helper()

	listed, err := session.ListResources(context.Background(), nil)
	require.NoError(t, err)
	for _, resource := range listed.Resources {
		if resource.URI == uri {
			return resource
		}
	}
	require.FailNow(t, "the server must list the resource", "resource %s", uri)
	return nil
}

type listedRule struct {
	Rule        string `json:"rule"`
	Description string `json:"description"`
}

func readRules(t *testing.T, session *sdk.ClientSession) []listedRule {
	t.Helper()

	read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://rules"})
	require.NoError(t, err)
	require.Len(t, read.Contents, 1)
	var decoded struct {
		Rules []listedRule `json:"rules"`
	}
	require.NoError(t, json.Unmarshal([]byte(read.Contents[0].Text), &decoded))
	return decoded.Rules
}

func rulesOf(entries []*diagnostic.Entry) []string {
	rules := make([]string, 0, len(entries))
	for _, entry := range entries {
		rules = append(rules, entry.RuleName)
	}
	return rules
}

func descriptionOf(property any) string {
	fields, _ := property.(map[string]any)
	description, _ := fields["description"].(string)
	return description
}

func typeOf(property any) map[string]any {
	fields, _ := property.(map[string]any)
	return map[string]any{"type": fields["type"]}
}

func filesOf(records []diagnostic.Record) []string {
	files := make([]string, 0, len(records))
	for _, r := range records {
		files = append(files, r.File)
	}
	return files
}
