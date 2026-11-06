//go:build unit

package mcp_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/mcp"
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
		t.Run("lists emod_check and emod_explain_rule, and no other tool", func(t *testing.T) {
			session := connect(t)

			listed, err := session.ListTools(context.Background(), nil)

			require.NoError(t, err)
			names := make([]string, 0, len(listed.Tools))
			for _, tool := range listed.Tools {
				names = append(names, tool.Name)
			}
			require.ElementsMatch(t, []string{"emod_check", "emod_explain_rule"}, names)
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
			require.Contains(t, descriptionOf(properties["content"]), "The .emod source text to check")
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

	t.Run("list resources", func(t *testing.T) {
		t.Run("lists emod://rules as JSON, with a description", func(t *testing.T) {
			session := connect(t)

			listed, err := session.ListResources(context.Background(), nil)

			require.NoError(t, err)
			require.Len(t, listed.Resources, 1)
			resource := listed.Resources[0]
			require.Equal(t, "emod://rules", resource.URI)
			require.Equal(t, "rules", resource.Name)
			require.Equal(t, "Lint rules of emod", resource.Title)
			require.Equal(t, "application/json", resource.MIMEType)
			require.Contains(t, resource.Description, "Each lint rule")
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
			require.Equal(t, "emod_check needs content or path", textOf(t, result))
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
