package diagnostic

// Diagnostic represents a parse error or warning.
type Diagnostic struct {
	Filename string
	Line     int
	Column   int
	Message  string
}
