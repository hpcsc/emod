package diagnostic

import "fmt"

type Severity int

const (
	Error   Severity = iota
	Warning Severity = iota
	Info    Severity = iota
)

func (s Severity) String() string {
	switch s {
	case Info:
		return "info"
	case Warning:
		return "warning"
	default:
		return "error"
	}
}

type Entry struct {
	Filename string
	Line     int
	Column   int
	Message  string
	Severity Severity
	RuleName string
}

func (d Entry) String() string {
	if d.RuleName != "" {
		return fmt.Sprintf("%s:%d: [%s] %s", d.Filename, d.Line, d.RuleName, d.Message)
	}
	return fmt.Sprintf("%s:%d: %s", d.Filename, d.Line, d.Message)
}
