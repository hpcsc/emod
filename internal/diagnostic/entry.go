package diagnostic

import "fmt"

// Entry represents a parse error or warning.
type Entry struct {
	Filename string
	Line     int
	Column   int
	Message  string
}

// String formats the diagnostic as "filename:line: message".
func (d Entry) String() string {
	return fmt.Sprintf("%s:%d: %s", d.Filename, d.Line, d.Message)
}
