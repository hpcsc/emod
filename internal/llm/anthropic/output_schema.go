package anthropic

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

type outputSchema struct {
	document map[string]any
	resolved *jsonschema.Resolved
}

func newOutputSchema(raw json.RawMessage) (*outputSchema, error) {
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, fmt.Errorf("the request schema is not a valid JSON Schema: %w", err)
	}
	if document == nil {
		return nil, errors.New("the request schema is not a valid JSON Schema: it is null")
	}

	var schema jsonschema.Schema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("the request schema is not a valid JSON Schema: %w", err)
	}

	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("the request schema is not a valid JSON Schema: %w", err)
	}

	return &outputSchema{document: document, resolved: resolved}, nil
}

func (s *outputSchema) check(text string) error {
	var instance any
	if err := json.Unmarshal([]byte(text), &instance); err != nil {
		return fmt.Errorf("the structured output is not JSON: %w", err)
	}

	if err := s.resolved.Validate(instance); err != nil {
		return fmt.Errorf("the structured output does not conform to the request schema: %w", err)
	}

	return nil
}
