//go:build unit

package lsp_test

import (
	"encoding/json"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

// The expected JSON below is the shape the LSP specification defines, so an
// editor built against the spec can read what this server writes. Only the
// encoding direction is asserted — decoding is exercised for real by
// server_test.go, which drives the server over a JSON-RPC pipe.
func TestProtocolTypes(t *testing.T) {
	t.Run("positions and ranges", func(t *testing.T) {
		t.Run("a position encodes line and character", func(t *testing.T) {
			requireEncodesTo(t, lsp.Position{Line: 10, Character: 5},
				`{"line":10,"character":5}`)
		})

		t.Run("a range encodes its start and end positions", func(t *testing.T) {
			requireEncodesTo(t, lsp.Range{
				Start: lsp.Position{Line: 1, Character: 0},
				End:   lsp.Position{Line: 1, Character: 5},
			}, `{"start":{"line":1,"character":0},"end":{"line":1,"character":5}}`)
		})
	})

	t.Run("diagnostics", func(t *testing.T) {
		t.Run("severities use the spec's numbering", func(t *testing.T) {
			require.Equal(t, lsp.DiagnosticSeverity(1), lsp.SeverityError)
			require.Equal(t, lsp.DiagnosticSeverity(2), lsp.SeverityWarning)
		})

		t.Run("a diagnostic encodes its range, severity, message and source", func(t *testing.T) {
			requireEncodesTo(t, lsp.Diagnostic{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 0, Character: 1},
				},
				Severity: lsp.SeverityError,
				Message:  "unexpected token",
				Source:   "emod",
			}, `{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},`+
				`"severity":1,"message":"unexpected token","source":"emod"}`)
		})

		t.Run("publish params carry the document URI alongside its diagnostics", func(t *testing.T) {
			requireEncodesTo(t, lsp.PublishDiagnosticsParams{
				URI: "file:///test.emod",
				Diagnostics: []lsp.Diagnostic{{
					Range:    lsp.Range{Start: lsp.Position{}, End: lsp.Position{}},
					Severity: lsp.SeverityError,
					Message:  "syntax error",
					Source:   "emod",
				}},
			}, `{"uri":"file:///test.emod","diagnostics":[{"range":{"start":{"line":0,"character":0},`+
				`"end":{"line":0,"character":0}},"severity":1,"message":"syntax error","source":"emod"}]}`)
		})
	})

	t.Run("documents", func(t *testing.T) {
		t.Run("a text document item carries URI, language, version and text", func(t *testing.T) {
			requireEncodesTo(t, lsp.TextDocumentItem{
				URI:        "file:///test.emod",
				LanguageID: "emod",
				Version:    1,
				Text:       "model User {}",
			}, `{"uri":"file:///test.emod","languageId":"emod","version":1,"text":"model User {}"}`)
		})

		t.Run("a versioned identifier carries URI and version", func(t *testing.T) {
			requireEncodesTo(t, lsp.VersionedTextDocumentIdentifier{
				URI:     "file:///test.emod",
				Version: 5,
			}, `{"uri":"file:///test.emod","version":5}`)
		})

		t.Run("an identifier carries only the URI", func(t *testing.T) {
			requireEncodesTo(t, lsp.TextDocumentIdentifier{URI: "file:///test.emod"},
				`{"uri":"file:///test.emod"}`)
		})
	})

	t.Run("completion", func(t *testing.T) {
		t.Run("kinds use the spec's numbering", func(t *testing.T) {
			require.Equal(t, lsp.CompletionItemKind(14), lsp.KeywordCompletion)
			require.Equal(t, lsp.CompletionItemKind(3), lsp.FunctionCompletion)
			require.Equal(t, lsp.CompletionItemKind(6), lsp.VariableCompletion)
		})

		t.Run("an item encodes label, kind, detail and documentation", func(t *testing.T) {
			requireEncodesTo(t, lsp.CompletionItem{
				Label:         "keyword",
				Kind:          lsp.KeywordCompletion,
				Detail:        "the keyword keyword",
				Documentation: "a language keyword",
			}, `{"label":"keyword","kind":14,"detail":"the keyword keyword","documentation":"a language keyword"}`)
		})

		t.Run("an item omits the fields it has no value for", func(t *testing.T) {
			requireEncodesTo(t, lsp.CompletionItem{Label: "bare"}, `{"label":"bare"}`)
		})

		t.Run("a list encodes its completeness flag and items", func(t *testing.T) {
			requireEncodesTo(t, lsp.CompletionList{
				IsIncomplete: true,
				Items: []lsp.CompletionItem{
					{Label: "func", Kind: lsp.FunctionCompletion},
					{Label: "var", Kind: lsp.VariableCompletion},
				},
			}, `{"isIncomplete":true,"items":[{"label":"func","kind":3},{"label":"var","kind":6}]}`)
		})

		t.Run("params carry the document and cursor position", func(t *testing.T) {
			requireEncodesTo(t, lsp.CompletionParams{
				TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.emod"},
				Position:     lsp.Position{Line: 3, Character: 10},
			}, `{"textDocument":{"uri":"file:///test.emod"},"position":{"line":3,"character":10}}`)
		})

		t.Run("options encode trigger characters, and omit them when there are none", func(t *testing.T) {
			requireEncodesTo(t, lsp.CompletionOptions{TriggerCharacters: []string{".", ":"}},
				`{"triggerCharacters":[".",":"]}`)
			requireEncodesTo(t, lsp.CompletionOptions{}, `{}`)
		})
	})

	t.Run("initialize result", func(t *testing.T) {
		t.Run("encodes the sync kind and any providers that are set", func(t *testing.T) {
			requireEncodesTo(t, lsp.InitializeResult{
				Capabilities: lsp.ServerCapabilities{
					TextDocumentSync:   lsp.SyncFull,
					CompletionProvider: &lsp.CompletionOptions{TriggerCharacters: []string{"."}},
				},
			}, `{"capabilities":{"textDocumentSync":1,"completionProvider":{"triggerCharacters":["."]}}}`)
		})

		t.Run("omits providers that are not offered", func(t *testing.T) {
			requireEncodesTo(t, lsp.InitializeResult{
				Capabilities: lsp.ServerCapabilities{TextDocumentSync: lsp.SyncNone},
			}, `{"capabilities":{"textDocumentSync":0}}`)
		})
	})

	t.Run("initialize params", func(t *testing.T) {
		t.Run("decodes the process ID and root URI a client sends", func(t *testing.T) {
			var params lsp.InitializeParams
			require.NoError(t, json.Unmarshal(
				[]byte(`{"processId":12345,"rootUri":"file:///workspace","capabilities":{}}`), &params))

			require.NotNil(t, params.ProcessID)
			require.Equal(t, 12345, *params.ProcessID)
			require.Equal(t, "file:///workspace", params.RootURI)
		})

		t.Run("tolerates a client that sends no process ID", func(t *testing.T) {
			var params lsp.InitializeParams
			require.NoError(t, json.Unmarshal(
				[]byte(`{"rootUri":"file:///workspace","capabilities":{}}`), &params))

			require.Nil(t, params.ProcessID)
			require.Equal(t, "file:///workspace", params.RootURI)
		})
	})
}

func requireEncodesTo(t *testing.T, value any, wantJSON string) {
	t.Helper()

	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	require.JSONEq(t, wantJSON, string(encoded))
}
