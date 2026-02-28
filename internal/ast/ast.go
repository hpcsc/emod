package ast

// Position represents the source location of an AST node.
type Position struct {
	Filename string
	Line     int
	Column   int
}

// Model represents the top-level emod model.
type Model struct {
	Name     string
	NamePos  Position
	Actors   []*Actor
	Contexts []*Context
}

// Actor represents an actor in the system.
type Actor struct {
	Name    string
	NamePos Position
}

// Context represents a bounded context.
type Context struct {
	Name       string
	NamePos    Position
	Aggregates []*Aggregate
	OpenPos    Position
	ClosePos   Position
}

// Aggregate represents an aggregate root.
type Aggregate struct {
	Name     string
	NamePos  Position
	Slices   []*Slice
	OpenPos  Position
	ClosePos Position
}

// Slice represents a slice (command/event pattern).
type Slice struct {
	Name     string
	NamePos  Position
	Commands []*Command
	Events   []*Event
	Fields   []*Field
	Flows    []*Flow
	OpenPos  Position
	ClosePos Position
}

// Command represents a command.
type Command struct {
	Name     string
	NamePos  Position
	Fields   []*Field
	OpenPos  Position
	ClosePos Position
}

// Event represents an event.
type Event struct {
	Name     string
	NamePos  Position
	Fields   []*Field
	OpenPos  Position
	ClosePos Position
}

// Field represents a field in a command or event.
type Field struct {
	Name     string
	NamePos  Position
	Type     string
	TypePos  Position
	Modifier string
	ModPos   Position
}

// Flow represents a command-to-event flow connection.
type Flow struct {
	CommandName string
	CommandPos  Position
	EventName   string
	EventPos    Position
}
