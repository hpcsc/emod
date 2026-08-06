package lsp

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/oracle"
)

// Server is an LSP server that reads JSON-RPC messages from an injected io.Reader,
// dispatches to method handlers, and writes responses/notifications to an injected io.Writer.
type Server struct {
	in        io.Reader
	out       io.Writer
	documents *DocumentManager
	shutdown  bool
}

// NewServer creates a new Server with dependency-injectable I/O.
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		in:        in,
		out:       out,
		documents: NewDocumentManager(),
	}
}

// Run starts the message read-dispatch-write loop. It blocks until shutdown
// completes, the context is cancelled, or a read/write error occurs.
func (s *Server) Run(ctx context.Context) error {
	for {
		if s.shutdown {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := ReadMessage(s.in)
		if err != nil {
			return err
		}

		if err := s.dispatch(ctx, msg); err != nil {
			return err
		}
	}
}

func (s *Server) dispatch(ctx context.Context, msg *Message) error {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "initialized":
		return nil
	case "textDocument/didOpen":
		return s.handleDidOpen(msg)
	case "textDocument/didChange":
		return s.handleDidChange(msg)
	case "textDocument/completion":
		return s.handleCompletion(msg)
	case "textDocument/definition":
		return s.handleDefinition(msg)
	case "textDocument/references":
		return s.handleReferences(msg)
	case "textDocument/formatting":
		return s.handleFormatting(msg)
	case "textDocument/hover":
		return s.handleHover(msg)
	case "textDocument/semanticTokens/full":
		return s.handleSemanticTokensFull(msg)
	case "shutdown":
		return s.handleShutdown(msg)
	default:
		if msg.ID != nil {
			return s.writeMessage(&Message{
				JSONRPC: Version,
				ID:      msg.ID,
				Error: &ErrorObject{
					Code:    -32601,
					Message: "method not found",
				},
			})
		}
		return nil
	}
}

func (s *Server) handleInitialize(msg *Message) error {
	triggerChars := []string{" "}
	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: SyncFull,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: triggerChars,
			},
			DefinitionProvider:         true,
			ReferencesProvider:         true,
			DocumentFormattingProvider: true,
			HoverProvider:              true,
			SemanticTokensProvider: &SemanticTokensProviderOptions{
				Legend: GetSemanticTokensLegend(),
			},
		},
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return s.writeMessage(&Message{
		JSONRPC: Version,
		ID:      msg.ID,
		Result:  resultBytes,
	})
}

func (s *Server) handleDidOpen(msg *Message) error {
	var params struct {
		TextDocument TextDocumentItem `json:"textDocument"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	uri := params.TextDocument.URI
	text := params.TextDocument.Text

	s.documents.Open(uri, text)
	s.pushDiagnostics(uri, text)

	return nil
}

func (s *Server) handleDidChange(msg *Message) error {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	uri := params.TextDocument.URI
	// For SyncFull, take the text from the last content change event.
	var text string
	if len(params.ContentChanges) > 0 {
		text = params.ContentChanges[len(params.ContentChanges)-1].Text
	}

	s.documents.Update(uri, text)
	s.pushDiagnostics(uri, text)

	return nil
}

func (s *Server) handleShutdown(msg *Message) error {
	if err := s.writeMessage(&Message{
		JSONRPC: Version,
		ID:      msg.ID,
		Result:  json.RawMessage(`{}`),
	}); err != nil {
		return err
	}
	s.shutdown = true
	return nil
}

func (s *Server) handleCompletion(msg *Message) error {
	var params CompletionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	uri := params.TextDocument.URI
	doc, ok := s.documents.GetContent(uri)
	if !ok {
		return s.writeMessage(&Message{
			JSONRPC: Version,
			ID:      msg.ID,
			Error: &ErrorObject{
				Code:    -32602,
				Message: "document not found: " + uri,
			},
		})
	}

	completions := GetCompletions(doc, params.Position.Line, params.Position.Character)
	resultBytes, err := json.Marshal(completions)
	if err != nil {
		return err
	}

	return s.writeMessage(&Message{
		JSONRPC: Version,
		ID:      msg.ID,
		Result:  resultBytes,
	})
}

func (s *Server) handleDefinition(msg *Message) error {
	var params DefinitionParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	uri := params.TextDocument.URI
	doc, ok := s.documents.GetContent(uri)
	if !ok {
		return s.writeMessage(&Message{
			JSONRPC: Version,
			ID:      msg.ID,
			Error: &ErrorObject{
				Code:    -32602,
				Message: "document not found: " + uri,
			},
		})
	}

	loc := GetDefinition(doc, params.Position.Line, params.Position.Character, uri)
	resultBytes, err := json.Marshal(loc)
	if err != nil {
		return err
	}

	return s.writeMessage(&Message{
		JSONRPC: Version,
		ID:      msg.ID,
		Result:  resultBytes,
	})
}

func (s *Server) handleReferences(msg *Message) error {
	var params ReferenceParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	uri := params.TextDocument.URI
	doc, ok := s.documents.GetContent(uri)
	if !ok {
		return s.writeMessage(&Message{
			JSONRPC: Version,
			ID:      msg.ID,
			Error: &ErrorObject{
				Code:    -32602,
				Message: "document not found: " + uri,
			},
		})
	}

	locs := GetReferences(doc, params.Position.Line, params.Position.Character, uri)
	resultBytes, err := json.Marshal(locs)
	if err != nil {
		return err
	}

	return s.writeMessage(&Message{
		JSONRPC: Version,
		ID:      msg.ID,
		Result:  resultBytes,
	})
}

func (s *Server) handleHover(msg *Message) error {
	var params HoverParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	uri := params.TextDocument.URI
	doc, ok := s.documents.GetContent(uri)
	if !ok {
		return s.writeMessage(&Message{
			JSONRPC: Version,
			ID:      msg.ID,
			Error: &ErrorObject{
				Code:    -32602,
				Message: "document not found: " + uri,
			},
		})
	}

	hover := GetHover(doc, params.Position.Line, params.Position.Character)
	resultBytes, err := json.Marshal(hover)
	if err != nil {
		return err
	}

	return s.writeMessage(&Message{
		JSONRPC: Version,
		ID:      msg.ID,
		Result:  resultBytes,
	})
}

func (s *Server) handleSemanticTokensFull(msg *Message) error {
	var params SemanticTokensParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	uri := params.TextDocument.URI
	doc, ok := s.documents.GetContent(uri)
	if !ok {
		return s.writeMessage(&Message{
			JSONRPC: Version,
			ID:      msg.ID,
			Error: &ErrorObject{
				Code:    -32602,
				Message: "document not found: " + uri,
			},
		})
	}

	tokens := GetSemanticTokens(doc)
	resultBytes, err := json.Marshal(tokens)
	if err != nil {
		return err
	}

	return s.writeMessage(&Message{
		JSONRPC: Version,
		ID:      msg.ID,
		Result:  resultBytes,
	})
}

func (s *Server) handleFormatting(msg *Message) error {
	var params DocumentFormattingParams
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return err
	}

	uri := params.TextDocument.URI
	doc, ok := s.documents.GetContent(uri)
	if !ok {
		return s.writeMessage(&Message{
			JSONRPC: Version,
			ID:      msg.ID,
			Error: &ErrorObject{
				Code:    -32602,
				Message: "document not found: " + uri,
			},
		})
	}

	// Lex → parse; if either produces errors, return empty TextEdit array.
	model, parseDiags := oracle.Parse(doc, uri)
	if len(parseDiags) > 0 {
		return s.writeMessage(&Message{
			JSONRPC: Version,
			ID:      msg.ID,
			Result:  json.RawMessage(`[]`),
		})
	}

	formatted := formatter.Format(model)

	// Build a full-document range.
	lines := strings.Split(doc, "\n")
	lastLine := len(lines) - 1
	lastLineLength := len(lines[lastLine])
	edit := TextEdit{
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: lastLine, Character: lastLineLength},
		},
		NewText: formatted,
	}

	resultBytes, err := json.Marshal([]TextEdit{edit})
	if err != nil {
		return err
	}

	return s.writeMessage(&Message{
		JSONRPC: Version,
		ID:      msg.ID,
		Result:  resultBytes,
	})
}

// pushDiagnostics runs the lex→parse→validate→lint pipeline on the given text
// and sends a textDocument/publishDiagnostics notification with the results.
func (s *Server) pushDiagnostics(uri, text string) {
	_, diags := oracle.Run(text, uri)

	lspDiags := ConvertDiagnostics(uri, diags)

	params := PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: lspDiags,
	}

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return
	}

	_ = s.writeMessage(&Message{
		JSONRPC: Version,
		Method:  "textDocument/publishDiagnostics",
		Params:  paramsBytes,
	})
}

func (s *Server) writeMessage(msg *Message) error {
	return WriteMessage(s.out, msg)
}
