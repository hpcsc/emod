package ast

type Position struct {
	Filename string
	Line     int
	Column   int
}

type Comment struct {
	Text string
	Position
}

type Model struct {
	Comments []*Comment
	Name     string
	NamePos  Position
	Actors   []*Actor
	Contexts []*Context
}

type Actor struct {
	Comments []*Comment
	Name     string
	NamePos Position
}

type Context struct {
	Comments   []*Comment
	Name       string
	NamePos    Position
	Aggregates []*Aggregate
	OpenPos    Position
	ClosePos   Position
}

type Aggregate struct {
	Comments []*Comment
	Name     string
	NamePos  Position
	Slices   []*Slice
	OpenPos  Position
	ClosePos Position
}

type Slice struct {
	Comments     []*Comment
	Name         string
	NamePos      Position
	Trigger      *Trigger
	Commands     []*Command
	Events       []*Event
	Fields       []*Field
	Flows        []*Flow
	Views        []*View
	Automations  []*Automation
	Translations []*Translation
	OpenPos      Position
	ClosePos     Position
}

type Command struct {
	Comments []*Comment
	Name     string
	NamePos  Position
	Fields   []*Field
	OpenPos  Position
	ClosePos Position
}

type Event struct {
	Comments        []*Comment
	Name            string
	NamePos         Position
	Source          string
	SourcePos       Position
	ExternalName    string
	ExternalNamePos Position
	Fields          []*Field
	OpenPos         Position
	ClosePos        Position
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
	Comments    []*Comment
	CommandName string
	CommandPos  Position
	EventName   string
	EventPos    Position
}

type Trigger struct {
	Comments []*Comment
	Kind     string
	KindPos  Position
	Name     string
	NamePos  Position
	Actor    string
	ActorPos Position
	Reads    string
	ReadsPos Position
	OpenPos  Position
	ClosePos Position
}

type View struct {
	Comments      []*Comment
	Name          string
	NamePos       Position
	Fields        []*Field
	Subscribes    []string
	SubscribesPos []Position
	OpenPos       Position
	ClosePos      Position
}

type Automation struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	TriggerEvent      string
	TriggerEventPos   Position
	Command           string
	CommandPos        Position
	TargetContext     string
	TargetContextPos  Position
	OpenPos        Position
	ClosePos       Position
}

type Translation struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	ExternalSystem string
	ExternalPos    Position
	Reads          string
	ReadsPos       Position
	Command        string
	CommandPos     Position
	Event          *Event
	OpenPos        Position
	ClosePos       Position
}
