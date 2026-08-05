//go:build unit

package lsp_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

// serverPair holds the I/O plumbing for a server instance under test.
type serverPair struct {
	inWriter  *io.PipeWriter
	outReader *io.PipeReader
	done      chan struct{}
	cancel    context.CancelFunc
}

// startServer creates a new Server with io.Pipe I/O and starts it in a
// background goroutine. The server is shut down during test cleanup.
func startServer(t *testing.T) *serverPair {
	t.Helper()

	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()

	server := lsp.NewServer(inReader, outWriter)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		server.Run(ctx)
		close(done)
	}()

	t.Cleanup(func() {
		cancel()
		inWriter.Close()
		outReader.Close()
		<-done
	})

	return &serverPair{
		inWriter:  inWriter,
		outReader: outReader,
		done:      done,
		cancel:    cancel,
	}
}

func (p *serverPair) writeMsg(t *testing.T, msg *lsp.Message) {
	t.Helper()
	err := lsp.WriteMessage(p.inWriter, msg)
	require.NoError(t, err)
}

func (p *serverPair) readMsg(t *testing.T) *lsp.Message {
	t.Helper()
	msg, err := lsp.ReadMessage(p.outReader)
	require.NoError(t, err)
	return msg
}

func (p *serverPair) writeInitialize(t *testing.T) int {
	t.Helper()
	id := 1
	p.writeMsg(t, &lsp.Message{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "initialize",
		Params:  []byte("{}"),
	})
	return id
}

func (p *serverPair) readInitializeResult(t *testing.T, expectedID int) lsp.InitializeResult {
	t.Helper()
	resp := p.readMsg(t)
	require.NotNil(t, resp.ID)
	require.Equal(t, expectedID, *resp.ID)
	require.Nil(t, resp.Error)
	require.NotNil(t, resp.Result)

	var result lsp.InitializeResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	require.Equal(t, lsp.SyncFull, result.Capabilities.TextDocumentSync)

	return result
}

func (p *serverPair) openDocument(t *testing.T, uri, text string) lsp.PublishDiagnosticsParams {
	t.Helper()
	p.writeMsg(t, &lsp.Message{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params: mustMarshal(t, map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri":        uri,
				"languageId": "emod",
				"version":    1,
				"text":       text,
			},
		}),
	})

	notif := p.readMsg(t)
	require.Equal(t, "textDocument/publishDiagnostics", notif.Method)

	return mustUnmarshalDiagnostics(t, notif.Params)
}

func (p *serverPair) writeCompletion(t *testing.T, uri string, line, character int) int {
	t.Helper()
	id := 2
	p.writeMsg(t, &lsp.Message{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  "textDocument/completion",
		Params: mustMarshal(t, map[string]interface{}{
			"textDocument": map[string]interface{}{
				"uri": uri,
			},
			"position": map[string]interface{}{
				"line":      line,
				"character": character,
			},
		}),
	})
	return id
}

func (p *serverPair) readCompletionResult(t *testing.T, expectedID int) lsp.CompletionList {
	t.Helper()
	resp := p.readMsg(t)
	require.NotNil(t, resp.ID)
	require.Equal(t, expectedID, *resp.ID)
	require.Nil(t, resp.Error)
	require.NotNil(t, resp.Result)

	var list lsp.CompletionList
	require.NoError(t, json.Unmarshal(resp.Result, &list))

	return list
}

// readMsgTimeout attempts to read a message within the given duration.
// Returns nil if no message arrives before the timeout.
func readMsgTimeout(r io.Reader, timeout time.Duration) *lsp.Message {
	ch := make(chan *lsp.Message, 1)
	go func() {
		msg, err := lsp.ReadMessage(r)
		if err == nil {
			ch <- msg
		}
	}()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(timeout):
		return nil
	}
}

func TestServer(t *testing.T) {
	t.Run("initialize", func(t *testing.T) {
		t.Run("advertises every capability the server implements", func(t *testing.T) {
			p := startServer(t)
			id := p.writeInitialize(t)

			result := p.readInitializeResult(t, id)

			// Each capability advertised here is exercised end-to-end by the
			// request groups below; the client only sends what it sees.
			require.Equal(t, lsp.ServerCapabilities{
				TextDocumentSync:           lsp.SyncFull,
				CompletionProvider:         &lsp.CompletionOptions{TriggerCharacters: []string{" "}},
				DefinitionProvider:         true,
				ReferencesProvider:         true,
				DocumentFormattingProvider: true,
				HoverProvider:              true,
				SemanticTokensProvider: &lsp.SemanticTokensProviderOptions{
					Legend: lsp.GetSemanticTokensLegend(),
				},
			}, result.Capabilities)
		})
	})

	t.Run("initialized", func(t *testing.T) {
		t.Run("accepts notification without response", func(t *testing.T) {
			p := startServer(t)

			// Confirm server is alive by initializing first.
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			// Send initialized notification (no ID — no response expected).
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				Method:  "initialized",
			})

			// Verify no output was written in response to the notification.
			msg := readMsgTimeout(p.outReader, 200*time.Millisecond)
			require.Nil(t, msg, "expected no response for initialized notification")
		})
	})

	t.Run("didOpen", func(t *testing.T) {
		t.Run("stores document and pushes diagnostics", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///test.emod"
			content := `model "test"`

			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				Method:  "textDocument/didOpen",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri":        uri,
						"languageId": "emod",
						"version":    1,
						"text":       content,
					},
				}),
			})

			// Read the publishDiagnostics notification.
			notif := p.readMsg(t)
			require.Equal(t, "textDocument/publishDiagnostics", notif.Method)
			require.Nil(t, notif.ID)

			var params lsp.PublishDiagnosticsParams
			err := json.Unmarshal(notif.Params, &params)
			require.NoError(t, err)
			require.Equal(t, uri, params.URI)
			// Well-formed content should produce no diagnostics.
			require.Empty(t, params.Diagnostics)
		})
	})

	t.Run("didChange", func(t *testing.T) {
		t.Run("updates document and pushes updated diagnostics", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///test.emod"

			p.openDocument(t, uri, `model "test"`)

			// Change to invalid content.
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				Method:  "textDocument/didChange",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri":     uri,
						"version": 2,
					},
					"contentChanges": []interface{}{
						map[string]interface{}{
							"text": "invalid syntax",
						},
					},
				}),
			})

			// Read updated diagnostics.
			notif := p.readMsg(t)
			require.Equal(t, "textDocument/publishDiagnostics", notif.Method)

			var params lsp.PublishDiagnosticsParams
			err := json.Unmarshal(notif.Params, &params)
			require.NoError(t, err)
			require.Equal(t, uri, params.URI)
			require.NotEmpty(t, params.Diagnostics, "expected diagnostics for invalid content")

			// All diagnostics from invalid content should be error-severity.
			for _, d := range params.Diagnostics {
				require.Equal(t, lsp.SeverityError, d.Severity, "expected error severity for lexer/parser error")
			}
		})
	})

	t.Run("shutdown", func(t *testing.T) {
		t.Run("sends empty response and stops event loop", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			shutdownID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &shutdownID,
				Method:  "shutdown",
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, shutdownID, *resp.ID)
			require.NotNil(t, resp.Result)
			require.Equal(t, "{}", string(resp.Result))
			require.Nil(t, resp.Error)

			// Verify the server loop exited.
			select {
			case <-p.done:
			case <-time.After(time.Second):
				require.Fail(t, "server did not exit after shutdown")
			}
		})
	})

	t.Run("diagnostics", func(t *testing.T) {
		t.Run("parser errors produce error-severity diagnostics", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///errors.emod"

			params := p.openDocument(t, uri, "invalid syntax here")
			require.Equal(t, uri, params.URI)
			require.NotEmpty(t, params.Diagnostics)

			for _, d := range params.Diagnostics {
				require.Equal(t, lsp.SeverityError, d.Severity, "expected error severity for parser errors")
				require.Equal(t, "emod", d.Source)
			}
		})

		t.Run("linter warnings appear as warning-severity diagnostics", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///linter.emod"
			// A valid model with a view that doesn't end in "View" triggers the
			// view-naming linter rule (warning).
			content := `model "test"
actor "A"
context "C" {
  aggregate "Agg" {
    slice "Sl" {
      command Cmd {}
      event Evt {
        fields {
          id String
        }
      }
      flow {
        command -> event: Cmd -> Evt
      }
      view Bad {
        subscribes [Evt]
      }
    }
  }
}`

			params := p.openDocument(t, uri, content)
			require.Equal(t, uri, params.URI)
			require.NotEmpty(t, params.Diagnostics,
				"expected at least the view-naming linter warning")

			// At least one diagnostic should be a linter warning (severity=2).
			foundWarning := false
			for _, d := range params.Diagnostics {
				if d.Severity == lsp.SeverityWarning {
					foundWarning = true
					break
				}
			}
			require.True(t, foundWarning, "expected at least one warning-severity diagnostic from linter")
		})

		t.Run("well-formed document produces empty diagnostics", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///valid.emod"

			params := p.openDocument(t, uri, `model "valid"`)
			require.Equal(t, uri, params.URI)
			require.Empty(t, params.Diagnostics)
		})
	})

	t.Run("unknown method", func(t *testing.T) {
		t.Run("with ID returns method-not-found error", func(t *testing.T) {
			p := startServer(t)

			id := 1
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &id,
				Method:  "unknownMethod",
				Params:  []byte("{}"),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, id, *resp.ID)
			require.NotNil(t, resp.Error)
			require.Equal(t, -32601, resp.Error.Code)
			require.Contains(t, resp.Error.Message, "method not found")
		})

		t.Run("without ID is silently ignored", func(t *testing.T) {
			p := startServer(t)

			// First confirm server is alive.
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			// Send an unknown notification (no ID).
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				Method:  "unknownNotification",
			})

			// No response expected — verify by timeout.
			msg := readMsgTimeout(p.outReader, 200*time.Millisecond)
			require.Nil(t, msg, "expected no response for unknown notification")
		})
	})

	t.Run("completion", func(t *testing.T) {
		t.Run("returns keyword completions for open document", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///test.emod"
			p.openDocument(t, uri, `model "test"`)

			completionID := p.writeCompletion(t, uri, 0, 5)

			list := p.readCompletionResult(t, completionID)

			require.False(t, list.IsIncomplete)
			require.NotEmpty(t, list.Items)
			for _, item := range list.Items {
				require.Equal(t, lsp.KeywordCompletion, item.Kind)
			}
		})

		t.Run("returns automation entries for a position inside an automation block", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///automation.emod"
			p.openDocument(t, uri, `context Ctx {
	aggregate Agg {
		slice Slc {
			automation Auto {

			}
		}
	}
}`)

			completionID := p.writeCompletion(t, uri, 4, 4)

			list := p.readCompletionResult(t, completionID)

			require.Equal(t, []string{"on", "every", "reads", "command", "target context"}, extractLabels(list.Items))
		})

		t.Run("returns error for unknown document URI", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			completionID := p.writeCompletion(t, "file:///unknown.emod", 0, 0)

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, completionID, *resp.ID)
			require.NotNil(t, resp.Error)
			require.Contains(t, resp.Error.Message, "document not found")
		})

		t.Run("gracefully handles partial content at cursor", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///partial.emod"
			p.openDocument(t, uri, `model `)

			completionID := p.writeCompletion(t, uri, 0, 6)

			list := p.readCompletionResult(t, completionID)

			require.NotEmpty(t, list.Items)
			labels := extractLabels(list.Items)
			require.Contains(t, labels, "model")
			require.Contains(t, labels, "actor")
			require.Contains(t, labels, "context")
		})
	})

	t.Run("definition", func(t *testing.T) {
		t.Run("returns location for known reference in open document", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///test.emod"
			// A document with a view that subscribes to an event.
			content := `context "Orders" {
    aggregate "Sales" {
        slice "OrderSlice" {
            event OrderSubmitted {
            }
            view OrderView {
                subscribes [OrderSubmitted]
            }
        }
    }
}`
			p.openDocument(t, uri, content)

			defID := 2
			// Position is on "OrderSubmitted" in the subscribes line.
			// In the content, "subscribes [OrderSubmitted]" is at line 6 (0-based).
			// The text "OrderSubmitted" starts at column 28 (0-based) on that line.
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &defID,
				Method:  "textDocument/definition",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
					"position": map[string]interface{}{
						"line":      6,
						"character": 28,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, defID, *resp.ID)
			require.Nil(t, resp.Error)
			require.NotNil(t, resp.Result)

			var loc lsp.Location
			err := json.Unmarshal(resp.Result, &loc)
			require.NoError(t, err)
			require.Equal(t, uri, loc.URI)
			// The event "OrderSubmitted" definition is at line 3 (0-based),
			// column 18 (0-based), in the content above.
			require.Equal(t, 3, loc.Range.Start.Line)
			require.Equal(t, 18, loc.Range.Start.Character)
			require.Equal(t, 3, loc.Range.End.Line)
			require.Equal(t, 32, loc.Range.End.Character)
		})

		t.Run("returns null result when cursor not on a reference", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///test.emod"
			content := `model "test"`
			p.openDocument(t, uri, content)

			defID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &defID,
				Method:  "textDocument/definition",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
					"position": map[string]interface{}{
						"line":      0,
						"character": 0,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, defID, *resp.ID)
			require.Nil(t, resp.Error)
			// When no definition is found, the result should be JSON null.
			require.Equal(t, "null", string(resp.Result))
		})

		t.Run("returns error for unknown document URI", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			defID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &defID,
				Method:  "textDocument/definition",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": "file:///unknown.emod",
					},
					"position": map[string]interface{}{
						"line":      0,
						"character": 0,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, defID, *resp.ID)
			require.NotNil(t, resp.Error)
			require.Equal(t, -32602, resp.Error.Code)
			require.Contains(t, resp.Error.Message, "document not found")
		})
	})

	t.Run("references", func(t *testing.T) {
		t.Run("returns all reference locations for known name in open document", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///test.emod"
			content := `context "Orders" {
    aggregate "Sales" {
        slice "OrderSlice" {
            command SubmitOrder {
            }
            event OrderSubmitted {
            }
            view OrderView {
                subscribes [OrderSubmitted]
            }
            automation AutoSubmit {
                on OrderSubmitted
                command SubmitOrder
            }
        }
    }
}`
			p.openDocument(t, uri, content)

			refID := 2
			// Cursor on "OrderSubmitted" in event definition: line 5, character 18.
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &refID,
				Method:  "textDocument/references",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
					"position": map[string]interface{}{
						"line":      5,
						"character": 18,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, refID, *resp.ID)
			require.Nil(t, resp.Error)
			require.NotNil(t, resp.Result)

			var locs []lsp.Location
			err := json.Unmarshal(resp.Result, &locs)
			require.NoError(t, err)
			require.NotEmpty(t, locs, "expected at least one reference location")
			require.Equal(t, uri, locs[0].URI)
		})

		t.Run("returns null when cursor not on a resolvable name", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///test.emod"
			content := `context "Orders" {
    aggregate "Sales" {
        slice "OrderSlice" {
            event OrderSubmitted {
            }
            view OrderView {
                subscribes [OrderSubmitted]
            }
        }
    }
}`
			p.openDocument(t, uri, content)

			refID := 2
			// Cursor on "context" keyword: line 0, character 2.
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &refID,
				Method:  "textDocument/references",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
					"position": map[string]interface{}{
						"line":      0,
						"character": 2,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, refID, *resp.ID)
			require.Nil(t, resp.Error)
			require.Equal(t, "null", string(resp.Result))
		})

		t.Run("returns error for unknown document URI", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			refID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &refID,
				Method:  "textDocument/references",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": "file:///unknown.emod",
					},
					"position": map[string]interface{}{
						"line":      0,
						"character": 0,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, refID, *resp.ID)
			require.NotNil(t, resp.Error)
			require.Equal(t, -32602, resp.Error.Code)
			require.Contains(t, resp.Error.Message, "document not found")
		})
	})

	t.Run("hover", func(t *testing.T) {
		t.Run("returns null result for open document", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///hover.emod"
			content := `model "test"`
			p.openDocument(t, uri, content)

			hoverID := 2
			// Cursor at (0, 7) is on the string literal "test", not a keyword or definition.
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &hoverID,
				Method:  "textDocument/hover",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
					"position": map[string]interface{}{
						"line":      0,
						"character": 7,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, hoverID, *resp.ID)
			require.Nil(t, resp.Error)
			require.Equal(t, "null", string(resp.Result))
		})

		t.Run("returns keyword hover for open document", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///hover.emod"
			content := `context "Orders" {
    aggregate "Sales" {
        slice "OrderSlice" {
            automation ShipOnSubmit {
                on OrderSubmitted
                command ShipOrder
            }
            automation SweepStaleOrders {
                every "5m"
                command ExpireOrder
            }
        }
    }
}`
			p.openDocument(t, uri, content)

			for i, tc := range []struct {
				keyword   string
				line      int
				character int
				expected  string
			}{
				{
					keyword:   "automation",
					line:      3,
					character: 12,
					expected:  "Defines an automation, the reactive processor of the Automation pattern: activated by an on event or an every schedule, optionally reads a view, and sends a command.",
				},
				{
					keyword:   "on",
					line:      4,
					character: 16,
					expected:  "Names the event whose occurrence activates the automation.",
				},
				{
					keyword:   "every",
					line:      8,
					character: 16,
					expected:  `Sets the schedule that activates the automation: a duration such as "5m", or a five-field cron expression such as "0 2 * * *".`,
				},
			} {
				hoverID := 2 + i
				p.writeMsg(t, &lsp.Message{
					JSONRPC: "2.0",
					ID:      &hoverID,
					Method:  "textDocument/hover",
					Params: mustMarshal(t, map[string]interface{}{
						"textDocument": map[string]interface{}{
							"uri": uri,
						},
						"position": map[string]interface{}{
							"line":      tc.line,
							"character": tc.character,
						},
					}),
				})

				resp := p.readMsg(t)
				require.NotNil(t, resp.ID)
				require.Equal(t, hoverID, *resp.ID)
				require.Nil(t, resp.Error)
				require.NotNil(t, resp.Result)
				require.NotEqual(t, "null", string(resp.Result), "hover on %s", tc.keyword)

				var hover lsp.Hover
				err := json.Unmarshal(resp.Result, &hover)
				require.NoError(t, err)
				require.Equal(t, lsp.Markdown, hover.Contents.Kind)
				require.Equal(t, tc.expected, hover.Contents.Value, "hover on %s", tc.keyword)

				direct := lsp.GetHover(content, tc.line, tc.character)
				require.NotNil(t, direct, "direct hover on %s", tc.keyword)
				require.Equal(t, direct.Contents.Value, hover.Contents.Value, "hover on %s", tc.keyword)
			}
		})

		t.Run("returns error for unknown document URI", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			hoverID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &hoverID,
				Method:  "textDocument/hover",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": "file:///unknown.emod",
					},
					"position": map[string]interface{}{
						"line":      0,
						"character": 0,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, hoverID, *resp.ID)
			require.NotNil(t, resp.Error)
			require.Equal(t, -32602, resp.Error.Code)
			require.Contains(t, resp.Error.Message, "document not found")
		})
	})

	t.Run("formatting", func(t *testing.T) {
		t.Run("returns full-document TextEdit with formatted output for valid document", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///format.emod"
			content := "emod 1\nmodel \"test\"\n\nactor \"Guest\"\n\ncontext \"Orders\" {\n  aggregate \"Order\" {\n  }\n}\n"

			p.openDocument(t, uri, content)

			formatID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &formatID,
				Method:  "textDocument/formatting",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, formatID, *resp.ID)
			require.Nil(t, resp.Error)
			require.NotNil(t, resp.Result)

			var edits []lsp.TextEdit
			err := json.Unmarshal(resp.Result, &edits)
			require.NoError(t, err)

			require.Equal(t, []lsp.TextEdit{{
				Range: lsp.Range{
					Start: lsp.Position{Line: 0, Character: 0},
					End:   lsp.Position{Line: 9, Character: 0},
				},
				NewText: content,
			}}, edits)
		})

		t.Run("preserves comments in formatted output", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///comments.emod"
			content := "# System description\nmodel \"test\"\n"

			p.openDocument(t, uri, content)

			formatID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &formatID,
				Method:  "textDocument/formatting",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, formatID, *resp.ID)
			require.Nil(t, resp.Error)
			require.NotNil(t, resp.Result)

			var edits []lsp.TextEdit
			err := json.Unmarshal(resp.Result, &edits)
			require.NoError(t, err)
			require.Len(t, edits, 1)
			require.Contains(t, edits[0].NewText, "# System description")
		})

		t.Run("returns error for unknown document URI", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			formatID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &formatID,
				Method:  "textDocument/formatting",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": "file:///unknown.emod",
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, formatID, *resp.ID)
			require.NotNil(t, resp.Error)
			require.Equal(t, -32602, resp.Error.Code)
			require.Contains(t, resp.Error.Message, "document not found")
		})

		t.Run("returns empty TextEdit array for syntactically invalid document", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///invalid.emod"
			content := "invalid syntax here"

			p.openDocument(t, uri, content)

			formatID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &formatID,
				Method:  "textDocument/formatting",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, formatID, *resp.ID)
			require.Nil(t, resp.Error)
			require.NotNil(t, resp.Result)
			require.Equal(t, "[]", string(resp.Result))
		})
	})

	t.Run("semanticTokens", func(t *testing.T) {
		t.Run("returns delta-encoded semantic tokens for known document URI", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///st.emod"
			content := `context "Orders" {
    aggregate "Sales" {
        slice "OrderSlice" {
            command SubmitOrder {
            }
            event OrderSubmitted {
            }
        }
    }
}
`
			p.openDocument(t, uri, content)

			stID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &stID,
				Method:  "textDocument/semanticTokens/full",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, stID, *resp.ID)
			require.Nil(t, resp.Error)
			require.NotNil(t, resp.Result)

			var st lsp.SemanticTokens
			err := json.Unmarshal(resp.Result, &st)
			require.NoError(t, err)
			require.NotEmpty(t, st.Data, "expected semantic token data for valid document")
			// Delta-encoded data comes in groups of 5: deltaLine, deltaChar, length, tokenType, tokenModifiers.
			require.Equal(t, 0, len(st.Data)%5, "data length must be a multiple of 5")
		})

		t.Run("returns error for unknown document URI", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			stID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &stID,
				Method:  "textDocument/semanticTokens/full",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": "file:///unknown.emod",
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, stID, *resp.ID)
			require.NotNil(t, resp.Error)
			require.Equal(t, -32602, resp.Error.Code)
			require.Contains(t, resp.Error.Message, "document not found")
		})

		t.Run("returns empty data for unparseable document", func(t *testing.T) {
			p := startServer(t)
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			uri := "file:///invalid.emod"
			p.openDocument(t, uri, "this is not valid emod syntax")

			stID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &stID,
				Method:  "textDocument/semanticTokens/full",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri": uri,
					},
				}),
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, stID, *resp.ID)
			require.Nil(t, resp.Error)
			require.NotNil(t, resp.Result)

			var st lsp.SemanticTokens
			err := json.Unmarshal(resp.Result, &st)
			require.NoError(t, err)
			require.Empty(t, st.Data, "expected empty semantic token data for unparseable document")
		})
	})

	t.Run("full LSP session", func(t *testing.T) {
		t.Run("initialize, initialized, didOpen, didChange, shutdown", func(t *testing.T) {
			p := startServer(t)

			// 1. Initialize
			p.writeInitialize(t)
			p.readInitializeResult(t, 1)

			// 2. Initialized (no response)
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				Method:  "initialized",
			})

			// 3. didOpen with valid content
			uri := "file:///session.emod"
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				Method:  "textDocument/didOpen",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri":        uri,
						"languageId": "emod",
						"version":    1,
						"text":       `model "session"`,
					},
				}),
			})

			diag1 := p.readMsg(t)
			require.Equal(t, "textDocument/publishDiagnostics", diag1.Method)
			require.Empty(t, mustUnmarshalDiagnostics(t, diag1.Params).Diagnostics)

			// 4. didChange with invalid content
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				Method:  "textDocument/didChange",
				Params: mustMarshal(t, map[string]interface{}{
					"textDocument": map[string]interface{}{
						"uri":     uri,
						"version": 2,
					},
					"contentChanges": []interface{}{
						map[string]interface{}{
							"text": "invalid",
						},
					},
				}),
			})

			diag2 := p.readMsg(t)
			require.Equal(t, "textDocument/publishDiagnostics", diag2.Method)
			require.NotEmpty(t, mustUnmarshalDiagnostics(t, diag2.Params).Diagnostics)

			// 5. Shutdown
			shutdownID := 2
			p.writeMsg(t, &lsp.Message{
				JSONRPC: "2.0",
				ID:      &shutdownID,
				Method:  "shutdown",
			})

			resp := p.readMsg(t)
			require.NotNil(t, resp.ID)
			require.Equal(t, shutdownID, *resp.ID)
			require.Equal(t, "{}", string(resp.Result))

			select {
			case <-p.done:
			case <-time.After(time.Second):
				require.Fail(t, "server did not exit after shutdown")
			}
		})
	})
}

// mustMarshal serializes v to JSON and fails the test on error.
func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}

// mustUnmarshalDiagnostics unmarshals PublishDiagnosticsParams from raw params.
func mustUnmarshalDiagnostics(t *testing.T, raw json.RawMessage) lsp.PublishDiagnosticsParams {
	t.Helper()
	var params lsp.PublishDiagnosticsParams
	err := json.Unmarshal(raw, &params)
	require.NoError(t, err)
	return params
}
