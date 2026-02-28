package diagnostic

import "fmt"

type Entry struct {
	Filename string
	Line     int
	Column   int
	Message  string
}

func (d Entry) String() string {
	return fmt.Sprintf("%s:%d: %s", d.Filename, d.Line, d.Message)
}
