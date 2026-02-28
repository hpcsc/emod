package diagnostic

import "fmt"

// Diagnostic represents a parse error or warning.
type Diagnostic struct {
	Filename string
	Line     int
	Column   int
	Message  string
}

// String formats the diagnostic as "filename:line: message".
func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%d: %s", d.Filename, d.Line, d.Message)
}
