//go:build unit

package lsp_test

import (
	"encoding/json"
	"testing"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

func TestProtocolTypes(t *testing.T) {
	t.Run("position marshals and unmarshals", func(t *testing.T) {
		original := lsp.Position{Line: 10, Character: 5}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"line":10,"character":5}`, string(data))

		var decoded lsp.Position
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, original, decoded)
	})

	t.Run("range marshals and unmarshals", func(t *testing.T) {
		original := lsp.Range{
			Start: lsp.Position{Line: 1, Character: 0},
			End:   lsp.Position{Line: 1, Character: 5},
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"start":{"line":1,"character":0},"end":{"line":1,"character":5}}`, string(data))

		var decoded lsp.Range
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, original, decoded)
	})

	t.Run("diagnostic marshals and unmarshals", func(t *testing.T) {
		original := lsp.Diagnostic{
			Range: lsp.Range{
				Start: lsp.Position{Line: 0, Character: 0},
				End:   lsp.Position{Line: 0, Character: 1},
			},
			Severity: lsp.SeverityError,
			Message:  "unexpected token",
			Source:   "emod",
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":1}},"severity":1,"message":"unexpected token","source":"emod"}`, string(data))

		var decoded lsp.Diagnostic
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, original, decoded)
	})

	t.Run("diagnostic with warning severity", func(t *testing.T) {
		d := lsp.Diagnostic{
			Range:    lsp.Range{Start: lsp.Position{}, End: lsp.Position{}},
			Severity: lsp.SeverityWarning,
			Message:  "unused variable",
			Source:   "emod",
		}
		require.Equal(t, lsp.DiagnosticSeverity(2), d.Severity)
	})

	t.Run("initialize result includes server capabilities with sync kind", func(t *testing.T) {
		result := lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: lsp.SyncFull,
			},
		}
		data, err := json.Marshal(result)
		require.NoError(t, err)
		require.JSONEq(t, `{"capabilities":{"textDocumentSync":1}}`, string(data))

		var decoded lsp.InitializeResult
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, lsp.SyncFull, decoded.Capabilities.TextDocumentSync)
	})

	t.Run("publish diagnostics params marshals and unmarshals", func(t *testing.T) {
		original := lsp.PublishDiagnosticsParams{
			URI: "file:///test.emod",
			Diagnostics: []lsp.Diagnostic{
				{
					Range:    lsp.Range{Start: lsp.Position{}, End: lsp.Position{}},
					Severity: lsp.SeverityError,
					Message:  "syntax error",
					Source:   "emod",
				},
			},
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"uri":"file:///test.emod","diagnostics":[{"range":{"start":{"line":0,"character":0},"end":{"line":0,"character":0}},"severity":1,"message":"syntax error","source":"emod"}]}`, string(data))

		var decoded lsp.PublishDiagnosticsParams
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, original.URI, decoded.URI)
		require.Len(t, decoded.Diagnostics, 1)
		require.Equal(t, original.Diagnostics[0], decoded.Diagnostics[0])
	})

	t.Run("initialize params marshals and unmarshals", func(t *testing.T) {
		pid := 12345
		original := lsp.InitializeParams{
			ProcessID:    &pid,
			RootURI:      "file:///workspace",
			Capabilities: map[string]interface{}{},
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.Contains(t, string(data), `"processId":12345`)
		require.Contains(t, string(data), `"rootUri":"file:///workspace"`)

		var decoded lsp.InitializeParams
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.NotNil(t, decoded.ProcessID)
		require.Equal(t, 12345, *decoded.ProcessID)
		require.Equal(t, "file:///workspace", decoded.RootURI)
	})

	t.Run("text document item marshals and unmarshals", func(t *testing.T) {
		original := lsp.TextDocumentItem{
			URI:        "file:///test.emod",
			LanguageID: "emod",
			Version:    1,
			Text:       "model User {}",
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"uri":"file:///test.emod","languageId":"emod","version":1,"text":"model User {}"}`, string(data))

		var decoded lsp.TextDocumentItem
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, original, decoded)
	})

	t.Run("versioned text document identifier marshals and unmarshals", func(t *testing.T) {
		original := lsp.VersionedTextDocumentIdentifier{
			URI:     "file:///test.emod",
			Version: 5,
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"uri":"file:///test.emod","version":5}`, string(data))

		var decoded lsp.VersionedTextDocumentIdentifier
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, original, decoded)
	})

	t.Run("zero-value position is valid (0,0)", func(t *testing.T) {
		p := lsp.Position{}
		require.Equal(t, 0, p.Line)
		require.Equal(t, 0, p.Character)
	})

	t.Run("completion item kind keyword has value 14", func(t *testing.T) {
		require.Equal(t, lsp.CompletionItemKind(14), lsp.KeywordCompletion)
	})

	t.Run("completion item marshals and unmarshals with all fields", func(t *testing.T) {
		original := lsp.CompletionItem{
			Label:         "keyword",
			Kind:          lsp.KeywordCompletion,
			Detail:        "the keyword keyword",
			Documentation: "a language keyword",
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"label":"keyword","kind":14,"detail":"the keyword keyword","documentation":"a language keyword"}`, string(data))

		var decoded lsp.CompletionItem
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, original, decoded)
	})

	t.Run("completion item omits optional fields when zero", func(t *testing.T) {
		item := lsp.CompletionItem{Label: "bare"}
		data, err := json.Marshal(item)
		require.NoError(t, err)
		require.JSONEq(t, `{"label":"bare"}`, string(data))

		var decoded lsp.CompletionItem
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, "bare", decoded.Label)
		require.Equal(t, lsp.CompletionItemKind(0), decoded.Kind)
		require.Empty(t, decoded.Detail)
		require.Empty(t, decoded.Documentation)
	})

	t.Run("completion list marshals and unmarshals", func(t *testing.T) {
		original := lsp.CompletionList{
			IsIncomplete: true,
			Items: []lsp.CompletionItem{
				{Label: "func", Kind: lsp.FunctionCompletion},
				{Label: "var", Kind: lsp.VariableCompletion},
			},
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"isIncomplete":true,"items":[{"label":"func","kind":3},{"label":"var","kind":6}]}`, string(data))

		var decoded lsp.CompletionList
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.True(t, decoded.IsIncomplete)
		require.Len(t, decoded.Items, 2)
		require.Equal(t, "func", decoded.Items[0].Label)
	})

	t.Run("text document identifier marshals and unmarshals", func(t *testing.T) {
		original := lsp.TextDocumentIdentifier{URI: "file:///test.emod"}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"uri":"file:///test.emod"}`, string(data))

		var decoded lsp.TextDocumentIdentifier
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, original, decoded)
	})

	t.Run("completion params marshals and unmarshals", func(t *testing.T) {
		original := lsp.CompletionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: "file:///test.emod"},
			Position:     lsp.Position{Line: 3, Character: 10},
		}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"textDocument":{"uri":"file:///test.emod"},"position":{"line":3,"character":10}}`, string(data))

		var decoded lsp.CompletionParams
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, "file:///test.emod", decoded.TextDocument.URI)
		require.Equal(t, 3, decoded.Position.Line)
		require.Equal(t, 10, decoded.Position.Character)
	})

	t.Run("completion options marshals and unmarshals with trigger characters", func(t *testing.T) {
		original := lsp.CompletionOptions{TriggerCharacters: []string{".", ":"}}
		data, err := json.Marshal(original)
		require.NoError(t, err)
		require.JSONEq(t, `{"triggerCharacters":[".",":"]}`, string(data))

		var decoded lsp.CompletionOptions
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.Equal(t, []string{".", ":"}, decoded.TriggerCharacters)
	})

	t.Run("completion options omits trigger characters when empty", func(t *testing.T) {
		opts := lsp.CompletionOptions{}
		data, err := json.Marshal(opts)
		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(data))
	})

	t.Run("initialize result with completion provider", func(t *testing.T) {
		result := lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: lsp.SyncFull,
				CompletionProvider: &lsp.CompletionOptions{
					TriggerCharacters: []string{"."},
				},
			},
		}
		data, err := json.Marshal(result)
		require.NoError(t, err)
		require.JSONEq(t, `{"capabilities":{"textDocumentSync":1,"completionProvider":{"triggerCharacters":["."]}}}`, string(data))

		var decoded lsp.InitializeResult
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		require.NotNil(t, decoded.Capabilities.CompletionProvider)
		require.Equal(t, []string{"."}, decoded.Capabilities.CompletionProvider.TriggerCharacters)
	})

	t.Run("initialize result omits completion provider when nil", func(t *testing.T) {
		result := lsp.InitializeResult{
			Capabilities: lsp.ServerCapabilities{
				TextDocumentSync: lsp.SyncNone,
			},
		}
		data, err := json.Marshal(result)
		require.NoError(t, err)
		require.JSONEq(t, `{"capabilities":{"textDocumentSync":0}}`, string(data))
	})
}
