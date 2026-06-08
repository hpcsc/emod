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

// CompletionItemKind represents the kind of a completion item.
type CompletionItemKind int

const (
	TextCompletion         CompletionItemKind = 1
	MethodCompletion       CompletionItemKind = 2
	FunctionCompletion     CompletionItemKind = 3
	ConstructorCompletion  CompletionItemKind = 4
	FieldCompletion        CompletionItemKind = 5
	VariableCompletion     CompletionItemKind = 6
	ClassCompletion        CompletionItemKind = 7
	InterfaceCompletion    CompletionItemKind = 8
	ModuleCompletion       CompletionItemKind = 9
	PropertyCompletion     CompletionItemKind = 10
	UnitCompletion         CompletionItemKind = 11
	ValueCompletion        CompletionItemKind = 12
	EnumCompletion         CompletionItemKind = 13
	KeywordCompletion      CompletionItemKind = 14
	SnippetCompletion      CompletionItemKind = 15
	ColorCompletion        CompletionItemKind = 16
	FileCompletion         CompletionItemKind = 17
	ReferenceCompletion    CompletionItemKind = 18
	FolderCompletion       CompletionItemKind = 19
	EnumMemberCompletion   CompletionItemKind = 20
	ConstantCompletion     CompletionItemKind = 21
	StructCompletion       CompletionItemKind = 22
	EventCompletion        CompletionItemKind = 23
	OperatorCompletion     CompletionItemKind = 24
	TypeParameterCompletion CompletionItemKind = 25
)

// CompletionItem represents a completion item in an LSP completion response.
type CompletionItem struct {
	Label         string             `json:"label"`
	Kind          CompletionItemKind `json:"kind,omitempty"`
	Detail        string             `json:"detail,omitempty"`
	Documentation string             `json:"documentation,omitempty"`
}

// CompletionList represents a list of completion items.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// TextDocumentIdentifier identifies a specific text document by URI.
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// CompletionParams represents parameters for a "textDocument/completion" request.
type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// CompletionOptions represents options for text document completion.
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// DefinitionParams represents parameters for a "textDocument/definition" request.
type DefinitionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// ServerCapabilities represents capabilities the server advertises to the client.
type ServerCapabilities struct {
	TextDocumentSync   TextDocumentSyncKind `json:"textDocumentSync"`
	CompletionProvider *CompletionOptions   `json:"completionProvider,omitempty"`
	DefinitionProvider bool                 `json:"definitionProvider,omitempty"`
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

// Location represents a location in a text document (as per the LSP spec).
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// DidChangeTextDocumentParams represents parameters for the "textDocument/didChange" notification.
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier     `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent    `json:"contentChanges"`
}
