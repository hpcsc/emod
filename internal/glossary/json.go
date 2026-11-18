package glossary

import (
	"encoding/json"

	"github.com/hpcsc/emod/internal/ast"
)

// RenderJSON writes the same vocabulary as RenderMarkdown as one JSON document:
// the model, and each context with its own invariants, its aggregates each
// carrying the invariants it declares, and its commands, events and views in
// declaration order.
func RenderJSON(model *ast.Model) ([]byte, error) {
	if model == nil {
		return []byte{}, nil
	}

	encoded, err := json.Marshal(newDocument(model))
	if err != nil {
		return nil, err
	}

	return append(encoded, '\n'), nil
}
