package ast

// SupportedVersion is the version of the emod language this tool understands.
const SupportedVersion = 1

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
	Comments        []*Comment
	Version         int
	VersionDeclared bool
	Name            string
	NamePos         Position
	Description     string
	DescriptionPos  Position
	Actors          []*Actor
	Contexts        []*Context
	OpenPos         Position
	ClosePos        Position
}

type Actor struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	OpenPos        Position
	ClosePos       Position
}

type Context struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	Mode           string
	ModePos        Position
	Invariants     []*Invariant
	Slices         []*Slice
	Aggregates     []*Aggregate
	OpenPos        Position
	ClosePos       Position
}

type Aggregate struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	Invariants     []*Invariant
	Slices         []*Slice
	OpenPos        Position
	ClosePos       Position
}

type Invariant struct {
	Comments     []*Comment
	Name         string
	NamePos      Position
	Statement    string
	StatementPos Position
}

type Slice struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	Trigger        *Trigger
	Commands       []*Command
	Events         []*Event
	Fields         []*Field
	Flows          []*Flow
	Rejections     []*Rejection
	Views          []*View
	Automations    []*Automation
	Translations   []*Translation
	Specs          []*Spec
	OpenPos        Position
	ClosePos       Position
}

type Spec struct {
	Comments []*Comment
	Name     string
	NamePos  Position
	Given    []*SpecElement
	When     *SpecElement
	Then     ThenClause
	OpenPos  Position
	ClosePos Position
}

type SpecElement struct {
	Name    string
	NamePos Position
	Payload []*PayloadField
}

type PayloadField struct {
	Name     string
	NamePos  Position
	Value    string
	ValuePos Position
	Kind     LiteralKind
}

type LiteralKind int

const (
	StringLiteral LiteralKind = iota + 1
	IntegerLiteral
	DecimalLiteral
	BooleanLiteral
)

type ThenClause interface {
	thenNode()
}

type ThenEvents struct {
	Events []*SpecElement
}

func (*ThenEvents) thenNode() {}

type ThenRejected struct {
	InvariantName string
	InvariantPos  Position
}

func (*ThenRejected) thenNode() {}

type ThenView struct {
	ViewName string
	ViewPos  Position
}

func (*ThenView) thenNode() {}

type ThenCommand struct {
	CommandName string
	CommandPos  Position
}

func (*ThenCommand) thenNode() {}

type Command struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	Fields         []*Field
	DecidesOn      *DecidesOnClause
	OpenPos        Position
	ClosePos       Position
}

type Event struct {
	Comments        []*Comment
	Name            string
	NamePos         Position
	Description     string
	DescriptionPos  Position
	Source          string
	SourcePos       Position
	ExternalName    string
	ExternalNamePos Position
	Tags            []TagEntry
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

// Rejection is the other entry kind a flow block holds: a command refused by an
// invariant, appending nothing. It is a collection of its own rather than a
// variant of Flow because every reader of Slice.Flows means "a command that
// produced an event" by it.
type Rejection struct {
	Comments      []*Comment
	CommandName   string
	CommandPos    Position
	InvariantName string
	InvariantPos  Position
}

type Trigger struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	Actor          string
	ActorPos       Position
	Reads          string
	ReadsPos       Position
	OpenPos        Position
	ClosePos       Position
}

type View struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	Fields         []*Field
	Subscribes     []string
	SubscribesPos  []Position
	OpenPos        Position
	ClosePos       Position
}

type Automation struct {
	Comments         []*Comment
	Name             string
	NamePos          Position
	Description      string
	DescriptionPos   Position
	OnEvent          string
	OnEventPos       Position
	Schedule         string
	SchedulePos      Position
	Reads            string
	ReadsPos         Position
	Command          string
	CommandPos       Position
	TargetContext    string
	TargetContextPos Position
	OpenPos          Position
	ClosePos         Position
}

type Translation struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
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

// TagEntry represents a tag entry defined on an event with key:fieldRef syntax.
type TagEntry struct {
	Key         string
	KeyPos      Position
	FieldRef    string
	FieldRefPos Position
}

// DecidesOnClause represents a decides_on condition on a command.
type DecidesOnClause struct {
	Comments  []*Comment
	Events    []string
	EventsPos []Position
	Predicate PredicateExpr
	OpenPos   Position
	ClosePos  Position
}

// PredicateExpr is an interface for all predicate expression types.
type PredicateExpr interface {
	predicateNode()
}

// TagPredicate represents a tag condition like tag.priority == "high".
type TagPredicate struct {
	Field    string
	FieldPos Position
	Operator string
	OpPos    Position
	Value    string
	ValuePos Position
}

func (*TagPredicate) predicateNode() {}

// LogicalExpr represents a logical AND/OR combination of two predicates.
type LogicalExpr struct {
	Left     PredicateExpr
	Operator string
	OpPos    Position
	Right    PredicateExpr
}

func (*LogicalExpr) predicateNode() {}

// NotExpr represents a negated predicate.
type NotExpr struct {
	OpPos Position
	Expr  PredicateExpr
}

func (*NotExpr) predicateNode() {}
