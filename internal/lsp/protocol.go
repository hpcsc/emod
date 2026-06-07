package lsp

// Position represents a zero-based line and character position in a text document.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range of text in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// DiagnosticSeverity represents the severity level of a diagnostic.
type DiagnosticSeverity int

const (
	SeverityError   DiagnosticSeverity = 1
	SeverityWarning DiagnosticSeverity = 2
)

// Diagnostic represents a diagnostic entry from the LSP server.
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity"`
	Message  string             `json:"message"`
	Source   string             `json:"source"`
}

// TextDocumentSyncKind represents how the client synchronizes document content with the server.
type TextDocumentSyncKind int

const (
	SyncNone        TextDocumentSyncKind = 0
	SyncFull        TextDocumentSyncKind = 1
	SyncIncremental TextDocumentSyncKind = 2
)

// InitializeParams represents parameters for the "initialize" request.
type InitializeParams struct {
	ProcessID    *int        `json:"processId,omitempty"`
	RootURI      string      `json:"rootUri,omitempty"`
	Capabilities interface{} `json:"capabilities,omitempty"`
}

// ServerCapabilities represents capabilities the server advertises to the client.
type ServerCapabilities struct {
	TextDocumentSync TextDocumentSyncKind `json:"textDocumentSync"`
}

// InitializeResult is the result returned from the "initialize" request.
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

// PublishDiagnosticsParams represents parameters for the "textDocument/publishDiagnostics" notification.
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// TextDocumentItem represents an open text document.
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// VersionedTextDocumentIdentifier identifies a specific version of a text document.
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentContentChangeEvent represents a content change to a text document.
// For SyncFull, only the Text field is used.
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// DidChangeTextDocumentParams represents parameters for the "textDocument/didChange" notification.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier     `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent    `json:"contentChanges"`
}
