package diagram

import "github.com/hpcsc/emod/internal/ast"

// declaresFlow reports whether the slice already states, as an explicit flow,
// that command emits event.
//
// A translation implies the same connection between its command and its nested
// event, so exporters draw that arrow themselves. When the model also spells the
// flow out, both would draw it and the reader sees two identical arrows between
// the same pair of boxes — so exporters draw the implied arrow only when no
// explicit flow covers it.
func declaresFlow(s *ast.Slice, command, event string) bool {
	for _, flow := range s.Flows {
		if flow.CommandName == command && flow.EventName == event {
			return true
		}
	}
	return false
}
