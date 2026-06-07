package lsp

import (
	"github.com/hpcsc/emod/internal/diagnostic"
)

// ConvertDiagnostics converts internal diagnostic.Entry values to LSP Diagnostic types.
// It applies position mapping (1-based internal → 0-based LSP) and severity mapping.
func ConvertDiagnostics(_ string, entries []*diagnostic.Entry) []Diagnostic {
	if len(entries) == 0 {
		return nil
	}

	result := make([]Diagnostic, 0, len(entries))
	for _, entry := range entries {
		d := Diagnostic{
			Range: Range{
				Start: Position{
					Line:      entry.Line - 1,
					Character: entry.Column - 1,
				},
				End: Position{
					Line:      entry.Line - 1,
					Character: entry.Column,
				},
			},
			Message: entry.Message,
			Source:  "emod",
		}

		switch entry.Severity {
		case diagnostic.Warning:
			d.Severity = SeverityWarning
		default:
			d.Severity = SeverityError
		}

		result = append(result, d)
	}

	return result
}
