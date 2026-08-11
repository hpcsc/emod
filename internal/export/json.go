package export

import (
	"encoding/json"
	"strings"

	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/diagnostic"
)

// JSON intermediate types with struct tags for serialization.
// These are separate from AST types to avoid coupling serialization concerns
// into the domain types.

// jsonPosition represents source position information for JSON output.
type jsonPosition struct {
	Filename string `json:"filename,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

type jsonComment struct {
	Text     string        `json:"text"`
	Position *jsonPosition `json:"position,omitempty"`
}

type jsonModel struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Position    *jsonPosition  `json:"position,omitempty"`
	Comments    []*jsonComment `json:"comments,omitempty"`
	Actors      []*jsonActor   `json:"actors,omitempty"`
	Contexts    []*jsonContext `json:"contexts,omitempty"`
}

type jsonActor struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Position    *jsonPosition  `json:"position,omitempty"`
	Comments    []*jsonComment `json:"comments,omitempty"`
}

type jsonContext struct {
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	Position      *jsonPosition    `json:"position,omitempty"`
	OpenPosition  *jsonPosition    `json:"open_position,omitempty"`
	ClosePosition *jsonPosition    `json:"close_position,omitempty"`
	Comments      []*jsonComment   `json:"comments,omitempty"`
	Invariants    []*jsonInvariant `json:"invariants,omitempty"`
	Aggregates    []*jsonAggregate `json:"aggregates,omitempty"`
	Slices        []*jsonSlice     `json:"slices,omitempty"`
}

type jsonAggregate struct {
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	Position      *jsonPosition    `json:"position,omitempty"`
	OpenPosition  *jsonPosition    `json:"open_position,omitempty"`
	ClosePosition *jsonPosition    `json:"close_position,omitempty"`
	Comments      []*jsonComment   `json:"comments,omitempty"`
	Invariants    []*jsonInvariant `json:"invariants,omitempty"`
	Slices        []*jsonSlice     `json:"slices,omitempty"`
}

type jsonInvariant struct {
	Comments  []*jsonComment `json:"comments,omitempty"`
	Name      string         `json:"name"`
	Statement string         `json:"statement"`
}

type jsonSlice struct {
	Name          string             `json:"name"`
	Description   string             `json:"description,omitempty"`
	Position      *jsonPosition      `json:"position,omitempty"`
	OpenPosition  *jsonPosition      `json:"open_position,omitempty"`
	ClosePosition *jsonPosition      `json:"close_position,omitempty"`
	Comments      []*jsonComment     `json:"comments,omitempty"`
	Trigger       *jsonTrigger       `json:"trigger,omitempty"`
	Commands      []*jsonCommand     `json:"commands,omitempty"`
	Events        []*jsonEvent       `json:"events,omitempty"`
	Fields        []*jsonField       `json:"fields,omitempty"`
	Flows         []*jsonFlow        `json:"flows,omitempty"`
	Rejections    []*jsonRejection   `json:"rejections,omitempty"`
	Views         []*jsonView        `json:"views,omitempty"`
	Automations   []*jsonAutomation  `json:"automations,omitempty"`
	Translations  []*jsonTranslation `json:"translations,omitempty"`
	Specs         []*jsonSpec        `json:"specs,omitempty"`
}

type jsonCommand struct {
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Position      *jsonPosition  `json:"position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
	Comments      []*jsonComment `json:"comments,omitempty"`
	Fields        []*jsonField   `json:"fields,omitempty"`
}

type jsonEvent struct {
	Name                 string         `json:"name"`
	Description          string         `json:"description,omitempty"`
	Position             *jsonPosition  `json:"position,omitempty"`
	SourcePosition       *jsonPosition  `json:"source_position,omitempty"`
	ExternalNamePosition *jsonPosition  `json:"external_name_position,omitempty"`
	OpenPosition         *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition        *jsonPosition  `json:"close_position,omitempty"`
	Comments             []*jsonComment `json:"comments,omitempty"`
	Source               string         `json:"source,omitempty"`
	ExternalName         string         `json:"external_name,omitempty"`
	Fields               []*jsonField   `json:"fields,omitempty"`
}

type jsonField struct {
	Name             string        `json:"name"`
	Position         *jsonPosition `json:"position,omitempty"`
	TypePosition     *jsonPosition `json:"type_position,omitempty"`
	ModifierPosition *jsonPosition `json:"modifier_position,omitempty"`
	Type             string        `json:"type"`
	Modifier         string        `json:"modifier,omitempty"`
}

type jsonFlow struct {
	Comments        []*jsonComment `json:"comments,omitempty"`
	CommandName     string         `json:"command_name"`
	CommandPosition *jsonPosition  `json:"command_position,omitempty"`
	EventName       string         `json:"event_name"`
	EventPosition   *jsonPosition  `json:"event_position,omitempty"`
}

// jsonRejection files its keys like jsonFlow rather than like the json* family,
// whose documents open with Name. The two are the entry kinds of one flow block,
// and filing them differently from each other would read worse than filing both
// differently from the family.
type jsonRejection struct {
	Comments          []*jsonComment `json:"comments,omitempty"`
	CommandName       string         `json:"command_name"`
	CommandPosition   *jsonPosition  `json:"command_position,omitempty"`
	InvariantName     string         `json:"invariant_name"`
	InvariantPosition *jsonPosition  `json:"invariant_position,omitempty"`
}

type jsonTrigger struct {
	Comments      []*jsonComment `json:"comments,omitempty"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Position      *jsonPosition  `json:"position,omitempty"`
	Actor         string         `json:"actor,omitempty"`
	ActorPosition *jsonPosition  `json:"actor_position,omitempty"`
	Reads         string         `json:"reads,omitempty"`
	ReadsPosition *jsonPosition  `json:"reads_position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
}

type jsonView struct {
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	Position      *jsonPosition  `json:"position,omitempty"`
	OpenPosition  *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition *jsonPosition  `json:"close_position,omitempty"`
	Comments      []*jsonComment `json:"comments,omitempty"`
	Fields        []*jsonField   `json:"fields,omitempty"`
	Subscribes    []string       `json:"subscribes,omitempty"`
}

type jsonAutomation struct {
	Name                  string         `json:"name"`
	Description           string         `json:"description,omitempty"`
	Position              *jsonPosition  `json:"position,omitempty"`
	OnEventPosition       *jsonPosition  `json:"on_event_position,omitempty"`
	SchedulePosition      *jsonPosition  `json:"every_position,omitempty"`
	ReadsPosition         *jsonPosition  `json:"reads_position,omitempty"`
	CommandPosition       *jsonPosition  `json:"command_position,omitempty"`
	TargetContextPosition *jsonPosition  `json:"target_context_position,omitempty"`
	OpenPosition          *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition         *jsonPosition  `json:"close_position,omitempty"`
	Comments              []*jsonComment `json:"comments,omitempty"`
	OnEvent               string         `json:"on_event,omitempty"`
	Schedule              string         `json:"every,omitempty"`
	Reads                 string         `json:"reads,omitempty"`
	Command               string         `json:"command,omitempty"`
	TargetContext         string         `json:"target_context,omitempty"`
}

type jsonTranslation struct {
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	Position         *jsonPosition  `json:"position,omitempty"`
	ExternalPosition *jsonPosition  `json:"external_position,omitempty"`
	ReadsPosition    *jsonPosition  `json:"reads_position,omitempty"`
	CommandPosition  *jsonPosition  `json:"command_position,omitempty"`
	OpenPosition     *jsonPosition  `json:"open_position,omitempty"`
	ClosePosition    *jsonPosition  `json:"close_position,omitempty"`
	Comments         []*jsonComment `json:"comments,omitempty"`
	ExternalSystem   string         `json:"external_system,omitempty"`
	Reads            string         `json:"reads,omitempty"`
	Command          string         `json:"command,omitempty"`
	Event            *jsonEvent     `json:"event,omitempty"`
}

type jsonSpec struct {
	Name          string             `json:"name"`
	Position      *jsonPosition      `json:"position,omitempty"`
	WhenPosition  *jsonPosition      `json:"when_position,omitempty"`
	OpenPosition  *jsonPosition      `json:"open_position,omitempty"`
	ClosePosition *jsonPosition      `json:"close_position,omitempty"`
	Comments      []*jsonComment     `json:"comments,omitempty"`
	Given         []*jsonSpecElement `json:"given,omitempty"`
	When          *jsonSpecElement   `json:"when,omitempty"`
	Then          *jsonSpecOutcome   `json:"then,omitempty"`
}

type jsonSpecElement struct {
	Name    string              `json:"name"`
	Payload []*jsonPayloadField `json:"payload,omitempty"`
}

type jsonPayloadField struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type jsonSpecOutcome struct {
	RejectedPosition *jsonPosition      `json:"rejected_position,omitempty"`
	Events           []*jsonSpecElement `json:"events,omitempty"`
	Rejected         string             `json:"rejected,omitempty"`
	ViewPosition     *jsonPosition      `json:"view_position,omitempty"`
	View             string             `json:"view,omitempty"`
	CommandPosition  *jsonPosition      `json:"command_position,omitempty"`
	Command          string             `json:"command,omitempty"`
}

// jsonDiagnosticsWrapper is the top-level envelope for JSON diagnostics output.
type jsonDiagnosticsWrapper struct {
	Diagnostics []*jsonDiagnosticEntry `json:"diagnostics"`
	Model       json.RawMessage        `json:"model"`
}

// jsonDiagnosticEntry maps a diagnostic.Entry to JSON.
type jsonDiagnosticEntry struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	RuleName string `json:"rule_name,omitempty"`
}

// ExportJSONDiagnostics wraps the model JSON and a diagnostics slice into a structured envelope
// with top-level diagnostics array and model object.
func ExportJSONDiagnostics(model *ast.Model, diagnostics []*diagnostic.Entry) ([]byte, error) {
	modelJSON, err := ExportJSON(model)
	if err != nil {
		return nil, err
	}

	wrapper := jsonDiagnosticsWrapper{
		Diagnostics: convertDiagnostics(diagnostics),
		Model:       json.RawMessage(modelJSON),
	}

	return json.Marshal(wrapper)
}

// convertDiagnostics converts a slice of diagnostic Entries to JSON diagnostic entries.
// Nil input produces an empty slice (not null) to ensure diagnostics: [] in output.
func convertDiagnostics(diags []*diagnostic.Entry) []*jsonDiagnosticEntry {
	if diags == nil {
		return []*jsonDiagnosticEntry{}
	}

	out := make([]*jsonDiagnosticEntry, 0, len(diags))
	for _, d := range diags {
		out = append(out, convertDiagnostic(d))
	}
	return out
}

func convertDiagnostic(d *diagnostic.Entry) *jsonDiagnosticEntry {
	return &jsonDiagnosticEntry{
		File:     d.Filename,
		Line:     d.Line,
		Column:   d.Column,
		Message:  d.Message,
		Severity: d.Severity.String(),
		RuleName: d.RuleName,
	}
}

// convertPosition converts an AST Position to a *jsonPosition for serialization.
// Returns nil for zero-value positions so omitempty excludes them from output.
func convertPosition(p ast.Position) *jsonPosition {
	if p == (ast.Position{}) {
		return nil
	}
	return &jsonPosition{
		Filename: p.Filename,
		Line:     p.Line,
		Column:   p.Column,
	}
}

// ExportJSON serializes the given AST model to a JSON byte slice.
func ExportJSON(model *ast.Model) ([]byte, error) {
	j := convertModel(model)
	return json.Marshal(j)
}

func convertList[T, U any](items []T, convert func(T) U) []U {
	if items == nil {
		return nil
	}
	out := make([]U, 0, len(items))
	for _, item := range items {
		out = append(out, convert(item))
	}
	return out
}

func convertModel(m *ast.Model) *jsonModel {
	if m == nil {
		return nil
	}
	out := &jsonModel{
		Name:        m.Name,
		Description: m.Description,
		Position:    convertPosition(m.NamePos),
		Comments:    convertComments(m.Comments),
		Actors:      convertActors(m.Actors),
		Contexts:    convertContexts(m.Contexts),
	}
	return out
}

func convertComments(comments []*ast.Comment) []*jsonComment {
	return convertList(comments, convertComment)
}

func convertComment(c *ast.Comment) *jsonComment {
	return &jsonComment{
		Text:     c.Text,
		Position: convertPosition(c.Position),
	}
}

func convertActors(actors []*ast.Actor) []*jsonActor {
	return convertList(actors, convertActor)
}

func convertActor(a *ast.Actor) *jsonActor {
	if a == nil {
		return nil
	}
	return &jsonActor{
		Name:        a.Name,
		Description: a.Description,
		Position:    convertPosition(a.NamePos),
		Comments:    convertComments(a.Comments),
	}
}

func convertContexts(ctxs []*ast.Context) []*jsonContext {
	return convertList(ctxs, convertContext)
}

func convertContext(c *ast.Context) *jsonContext {
	if c == nil {
		return nil
	}
	return &jsonContext{
		Name:          c.Name,
		Description:   c.Description,
		Position:      convertPosition(c.NamePos),
		OpenPosition:  convertPosition(c.OpenPos),
		ClosePosition: convertPosition(c.ClosePos),
		Comments:      convertComments(c.Comments),
		Invariants:    convertInvariants(c.Invariants),
		Aggregates:    convertAggregates(c.Aggregates),
		Slices:        convertSlices(c.Slices),
	}
}

func convertInvariants(invariants []*ast.Invariant) []*jsonInvariant {
	return convertList(invariants, convertInvariant)
}

func convertInvariant(inv *ast.Invariant) *jsonInvariant {
	return &jsonInvariant{
		Comments:  convertComments(inv.Comments),
		Name:      inv.Name,
		Statement: inv.Statement,
	}
}

func convertAggregates(aggs []*ast.Aggregate) []*jsonAggregate {
	return convertList(aggs, convertAggregate)
}

func convertAggregate(a *ast.Aggregate) *jsonAggregate {
	if a == nil {
		return nil
	}
	return &jsonAggregate{
		Name:          a.Name,
		Description:   a.Description,
		Position:      convertPosition(a.NamePos),
		OpenPosition:  convertPosition(a.OpenPos),
		ClosePosition: convertPosition(a.ClosePos),
		Comments:      convertComments(a.Comments),
		Invariants:    convertInvariants(a.Invariants),
		Slices:        convertSlices(a.Slices),
	}
}

func convertSlices(slices []*ast.Slice) []*jsonSlice {
	return convertList(slices, convertSlice)
}

func convertSlice(s *ast.Slice) *jsonSlice {
	if s == nil {
		return nil
	}
	return &jsonSlice{
		Name:          s.Name,
		Description:   s.Description,
		Position:      convertPosition(s.NamePos),
		OpenPosition:  convertPosition(s.OpenPos),
		ClosePosition: convertPosition(s.ClosePos),
		Comments:      convertComments(s.Comments),
		Trigger:       convertTrigger(s.Trigger),
		Commands:      convertCommands(s.Commands),
		Events:        convertEvents(s.Events),
		Fields:        convertFields(s.Fields),
		Flows:         convertFlows(s.Flows),
		Rejections:    convertRejections(s.Rejections),
		Views:         convertViews(s.Views),
		Automations:   convertAutomations(s.Automations),
		Translations:  convertTranslations(s.Translations),
		Specs:         convertSpecs(s.Specs),
	}
}

func convertSpecs(specs []*ast.Spec) []*jsonSpec {
	return convertList(specs, convertSpec)
}

func convertSpec(s *ast.Spec) *jsonSpec {
	if s == nil {
		return nil
	}
	return &jsonSpec{
		Name:          s.Name,
		Position:      convertPosition(s.NamePos),
		WhenPosition:  specElementPosition(s.When),
		OpenPosition:  convertPosition(s.OpenPos),
		ClosePosition: convertPosition(s.ClosePos),
		Comments:      convertComments(s.Comments),
		Given:         convertSpecElements(s.Given),
		When:          convertSpecElement(s.When),
		Then:          convertSpecOutcome(s.Then),
	}
}

func convertSpecOutcome(then ast.ThenClause) *jsonSpecOutcome {
	switch t := then.(type) {
	case *ast.ThenEvents:
		return &jsonSpecOutcome{Events: convertSpecElements(t.Events)}
	case *ast.ThenRejected:
		return &jsonSpecOutcome{
			RejectedPosition: convertPosition(t.InvariantPos),
			Rejected:         t.InvariantName,
		}
	case *ast.ThenView:
		return &jsonSpecOutcome{
			ViewPosition: convertPosition(t.ViewPos),
			View:         t.ViewName,
		}
	case *ast.ThenCommand:
		return &jsonSpecOutcome{
			CommandPosition: convertPosition(t.CommandPos),
			Command:         t.CommandName,
		}
	}
	return nil
}

func convertSpecElements(elements []*ast.SpecElement) []*jsonSpecElement {
	if len(elements) == 0 {
		return nil
	}
	converted := make([]*jsonSpecElement, 0, len(elements))
	for _, element := range elements {
		converted = append(converted, convertSpecElement(element))
	}
	return converted
}

func convertSpecElement(element *ast.SpecElement) *jsonSpecElement {
	if element == nil {
		return nil
	}
	converted := &jsonSpecElement{Name: element.Name}
	for _, field := range element.Payload {
		converted.Payload = append(converted.Payload, &jsonPayloadField{
			Name:  field.Name,
			Value: payloadValue(field),
		})
	}
	return converted
}

// payloadValue carries a literal as the JSON type it stands for — a string as a
// string, a number as a number, true as a boolean — because the declared field
// type already tells a consumer whether a number is an int or a decimal. A
// number travels as its digits rather than a float64: parsing it here would
// round a literal wider than an int64, so 99999999999999999999 would export as
// 1e+20.
func payloadValue(field *ast.PayloadField) any {
	switch field.Kind {
	case ast.IntegerLiteral, ast.DecimalLiteral:
		return json.Number(canonicalNumber(field.Value))
	case ast.BooleanLiteral:
		return field.Value == "true"
	default:
		return field.Value
	}
}

// canonicalNumber trims the leading zeros an emod number literal may carry: 007
// is a legal payload value and an illegal JSON and CUE number, so a document
// emitting it verbatim is one neither format can read back.
func canonicalNumber(text string) string {
	whole, fraction, hasFraction := strings.Cut(text, ".")
	whole = strings.TrimLeft(whole, "0")
	if whole == "" {
		whole = "0"
	}
	if hasFraction {
		return whole + "." + fraction
	}

	return whole
}

func specElementPosition(element *ast.SpecElement) *jsonPosition {
	if element == nil {
		return nil
	}
	return convertPosition(element.NamePos)
}

func convertCommands(cmds []*ast.Command) []*jsonCommand {
	return convertList(cmds, convertCommand)
}

func convertCommand(c *ast.Command) *jsonCommand {
	if c == nil {
		return nil
	}
	return &jsonCommand{
		Name:          c.Name,
		Description:   c.Description,
		Position:      convertPosition(c.NamePos),
		OpenPosition:  convertPosition(c.OpenPos),
		ClosePosition: convertPosition(c.ClosePos),
		Comments:      convertComments(c.Comments),
		Fields:        convertFields(c.Fields),
	}
}

func convertEvents(events []*ast.Event) []*jsonEvent {
	return convertList(events, convertEvent)
}

func convertEvent(e *ast.Event) *jsonEvent {
	if e == nil {
		return nil
	}
	return &jsonEvent{
		Name:                 e.Name,
		Description:          e.Description,
		Position:             convertPosition(e.NamePos),
		SourcePosition:       convertPosition(e.SourcePos),
		ExternalNamePosition: convertPosition(e.ExternalNamePos),
		OpenPosition:         convertPosition(e.OpenPos),
		ClosePosition:        convertPosition(e.ClosePos),
		Comments:             convertComments(e.Comments),
		Source:               e.Source,
		ExternalName:         e.ExternalName,
		Fields:               convertFields(e.Fields),
	}
}

func convertFields(fields []*ast.Field) []*jsonField {
	return convertList(fields, convertField)
}

func convertField(f *ast.Field) *jsonField {
	if f == nil {
		return nil
	}
	return &jsonField{
		Name:             f.Name,
		Position:         convertPosition(f.NamePos),
		TypePosition:     convertPosition(f.TypePos),
		ModifierPosition: convertPosition(f.ModPos),
		Type:             f.Type,
		Modifier:         f.Modifier,
	}
}

func convertFlows(flows []*ast.Flow) []*jsonFlow {
	return convertList(flows, convertFlow)
}

func convertFlow(f *ast.Flow) *jsonFlow {
	if f == nil {
		return nil
	}
	return &jsonFlow{
		Comments:        convertComments(f.Comments),
		CommandName:     f.CommandName,
		CommandPosition: convertPosition(f.CommandPos),
		EventName:       f.EventName,
		EventPosition:   convertPosition(f.EventPos),
	}
}

func convertRejections(rejections []*ast.Rejection) []*jsonRejection {
	return convertList(rejections, convertRejection)
}

func convertRejection(r *ast.Rejection) *jsonRejection {
	if r == nil {
		return nil
	}
	return &jsonRejection{
		Comments:          convertComments(r.Comments),
		CommandName:       r.CommandName,
		CommandPosition:   convertPosition(r.CommandPos),
		InvariantName:     r.InvariantName,
		InvariantPosition: convertPosition(r.InvariantPos),
	}
}

func convertTrigger(t *ast.Trigger) *jsonTrigger {
	if t == nil {
		return nil
	}
	return &jsonTrigger{
		Comments:      convertComments(t.Comments),
		Name:          t.Name,
		Description:   t.Description,
		Position:      convertPosition(t.NamePos),
		Actor:         t.Actor,
		ActorPosition: convertPosition(t.ActorPos),
		Reads:         t.Reads,
		ReadsPosition: convertPosition(t.ReadsPos),
		OpenPosition:  convertPosition(t.OpenPos),
		ClosePosition: convertPosition(t.ClosePos),
	}
}

func convertViews(views []*ast.View) []*jsonView {
	return convertList(views, convertView)
}

func convertView(v *ast.View) *jsonView {
	if v == nil {
		return nil
	}
	return &jsonView{
		Name:          v.Name,
		Description:   v.Description,
		Position:      convertPosition(v.NamePos),
		OpenPosition:  convertPosition(v.OpenPos),
		ClosePosition: convertPosition(v.ClosePos),
		Comments:      convertComments(v.Comments),
		Fields:        convertFields(v.Fields),
		Subscribes:    v.Subscribes,
	}
}

func convertAutomations(autos []*ast.Automation) []*jsonAutomation {
	return convertList(autos, convertAutomation)
}

func convertAutomation(a *ast.Automation) *jsonAutomation {
	if a == nil {
		return nil
	}
	return &jsonAutomation{
		Name:                  a.Name,
		Description:           a.Description,
		Position:              convertPosition(a.NamePos),
		OnEventPosition:       convertPosition(a.OnEventPos),
		SchedulePosition:      convertPosition(a.SchedulePos),
		ReadsPosition:         convertPosition(a.ReadsPos),
		CommandPosition:       convertPosition(a.CommandPos),
		TargetContextPosition: convertPosition(a.TargetContextPos),
		OpenPosition:          convertPosition(a.OpenPos),
		ClosePosition:         convertPosition(a.ClosePos),
		Comments:              convertComments(a.Comments),
		OnEvent:               a.OnEvent,
		Schedule:              a.Schedule,
		Reads:                 a.Reads,
		Command:               a.Command,
		TargetContext:         a.TargetContext,
	}
}

func convertTranslations(trans []*ast.Translation) []*jsonTranslation {
	return convertList(trans, convertTranslation)
}

func convertTranslation(t *ast.Translation) *jsonTranslation {
	if t == nil {
		return nil
	}
	return &jsonTranslation{
		Name:             t.Name,
		Description:      t.Description,
		Position:         convertPosition(t.NamePos),
		ExternalPosition: convertPosition(t.ExternalPos),
		ReadsPosition:    convertPosition(t.ReadsPos),
		CommandPosition:  convertPosition(t.CommandPos),
		OpenPosition:     convertPosition(t.OpenPos),
		ClosePosition:    convertPosition(t.ClosePos),
		Comments:         convertComments(t.Comments),
		ExternalSystem:   t.ExternalSystem,
		Reads:            t.Reads,
		Command:          t.Command,
		Event:            convertEvent(t.Event),
	}
}
