package ast

import (
	"cmp"
	"slices"
)

// SupportedVersion is the version of the emod language this tool understands.
const SupportedVersion = 1

type Position struct {
	Filename string
	Line     int
	Column   int
}

// Compare orders positions by filename, then line, then column.
func (p Position) Compare(o Position) int {
	if c := cmp.Compare(p.Filename, o.Filename); c != 0 {
		return c
	}
	if c := cmp.Compare(p.Line, o.Line); c != 0 {
		return c
	}
	return cmp.Compare(p.Column, o.Column)
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
	Screens         []*Screen
	Contexts        []*Context
	OpenPos         Position
	ClosePos        Position
}

// SliceRefs returns every slice in the model in source order, each paired with
// the context and (when present) aggregate that declares it.
func (m *Model) SliceRefs() []SliceRef {
	if m == nil {
		return nil
	}

	var refs []SliceRef
	for _, ctx := range m.Contexts {
		refs = append(refs, ctx.SliceRefs()...)
	}

	return refs
}

// AllSlices returns every slice in the model in source order.
func (m *Model) AllSlices() []*Slice {
	refs := m.SliceRefs()
	all := make([]*Slice, len(refs))
	for i, ref := range refs {
		all[i] = ref.Slice
	}

	return all
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

// SliceRefs returns every slice the context declares — directly or via its
// aggregates — in source order. Direct slices and aggregates live in separate
// collections, so sorting by position is what recovers how they were written.
func (c *Context) SliceRefs() []SliceRef {
	if c == nil {
		return nil
	}

	refs := make([]SliceRef, 0, len(c.Slices))
	for _, s := range c.Slices {
		refs = append(refs, SliceRef{Slice: s, Context: c})
	}
	for _, agg := range c.Aggregates {
		for _, s := range agg.Slices {
			refs = append(refs, SliceRef{Slice: s, Context: c, Aggregate: agg})
		}
	}

	slices.SortStableFunc(refs, func(a, b SliceRef) int {
		return a.Slice.NamePos.Compare(b.Slice.NamePos)
	})

	return refs
}

// AllSlices returns every slice the context declares — directly or via its
// aggregates — in source order.
func (c *Context) AllSlices() []*Slice {
	refs := c.SliceRefs()
	all := make([]*Slice, len(refs))
	for i, ref := range refs {
		all[i] = ref.Slice
	}

	return all
}

// SliceRef pairs a slice with where it was declared: Aggregate is nil for the
// slices a `mode dcb` context declares directly.
type SliceRef struct {
	Slice     *Slice
	Context   *Context
	Aggregate *Aggregate
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
	Appearances    []*Appearance
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

type Screen struct {
	Comments       []*Comment
	Name           string
	NamePos        Position
	Description    string
	DescriptionPos Position
	OpenPos        Position
	ClosePos       Position
}

type Appearance struct {
	Comments    []*Comment
	Screen      string
	ScreenPos   Position
	Reads       []string
	ReadsPos    []Position
	Commands    []string
	CommandsPos []Position
	OpenPos     Position
	ClosePos    Position
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
	WireType        string
	WireTypePos     Position
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
	Reads          []string
	ReadsPos       []Position
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
	After            string
	AfterPos         Position
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
