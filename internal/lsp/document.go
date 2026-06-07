package lsp

// DocumentManager tracks open documents by URI with their current content in memory.
type DocumentManager struct {
	documents map[string]string
}

// NewDocumentManager creates a new DocumentManager.
func NewDocumentManager() *DocumentManager {
	return &DocumentManager{
		documents: make(map[string]string),
	}
}

// Open stores content for the given URI, replacing any existing content.
func (dm *DocumentManager) Open(uri, content string) {
	dm.documents[uri] = content
}

// Update stores content for the given URI.
func (dm *DocumentManager) Update(uri, content string) {
	dm.documents[uri] = content
}

// Close removes the document identified by URI from the store.
func (dm *DocumentManager) Close(uri string) {
	delete(dm.documents, uri)
}

// GetContent returns the current in-memory content for the given URI.
// The boolean is false if the URI has not been opened.
func (dm *DocumentManager) GetContent(uri string) (string, bool) {
	content, ok := dm.documents[uri]
	return content, ok
}
