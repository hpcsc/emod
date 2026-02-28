package ast

type Position struct {
	Filename string
	Line     int
	Column   int
}

type Model struct {
	Name     string
	NamePos  Position
	Actors   []*Actor
	Contexts []*Context
}

type Actor struct {
	Name    string
	NamePos Position
}

type Context struct {
	Name       string
	NamePos    Position
	Aggregates []*Aggregate
	OpenPos    Position
	ClosePos   Position
}

type Aggregate struct {
	Name     string
	NamePos  Position
	Slices   []*Slice
	OpenPos  Position
	ClosePos Position
}

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

type Command struct {
	Name     string
	NamePos  Position
	Fields   []*Field
	OpenPos  Position
	ClosePos Position
}

type Event struct {
	Name     string
	NamePos  Position
	Fields   []*Field
	OpenPos  Position
	ClosePos Position
}

type Field struct {
	Name     string
	NamePos  Position
	Type     string
	TypePos  Position
	Modifier string
	ModPos   Position
}

type Flow struct {
	CommandName string
	CommandPos  Position
	EventName   string
	EventPos    Position
}
