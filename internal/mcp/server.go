package mcp

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hpcsc/emod/docs"
	"github.com/hpcsc/emod/examples"
	"github.com/hpcsc/emod/internal/arrange"
	"github.com/hpcsc/emod/internal/ast"
	"github.com/hpcsc/emod/internal/cue"
	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/diagram"
	"github.com/hpcsc/emod/internal/export"
	"github.com/hpcsc/emod/internal/formatter"
	"github.com/hpcsc/emod/internal/glossary"
	"github.com/hpcsc/emod/internal/linter"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/version"
)

const inlineName = "inline.emod"

type sourceInput struct {
	Content *string `json:"content,omitempty"`
	Path    string  `json:"path,omitempty"`
	Name    string  `json:"name,omitempty"`
}

var sourceSchema = &jsonschema.Schema{
	Type: "object",
	Properties: map[string]*jsonschema.Schema{
		"content": {
			Type:        "string",
			Description: "The .emod source text.",
		},
		"path": {
			Type: "string",
			Description: "The path of an .emod file under the root of the server, relative to the root or absolute. " +
				"The tool reads the file only when the call gives no content.",
		},
		"name": {
			Type:        "string",
			Description: "The file name that the diagnostics of content carry. The default is path, else inline.emod.",
		},
	},
	AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
}

type formatInput struct {
	sourceInput
	Format string `json:"format,omitempty"`
}

type writeInput struct {
	sourceInput
	Write bool `json:"write,omitempty"`
}

type diagramInput struct {
	sourceInput
	Format string `json:"format,omitempty"`
	Style  string `json:"style,omitempty"`
	Specs  bool   `json:"specs,omitempty"`
}

type checkResult struct {
	Diagnostics []diagnostic.Record `json:"diagnostics"`
}

var checkTool = &sdk.Tool{
	Name: "emod_check",
	Description: "Check .emod source with the parser, the validator and the linter of emod. " +
		"Give the source in content, or give the path of an .emod file under the root of the server in path. " +
		"When a call gives both, the tool checks content and does not read path. " +
		"The result is a list of diagnostics, the same records as `emod validate --format json`: " +
		"file, line, rule (only when a rule name goes with the diagnostic), severity and message. " +
		"An empty list means that the source passes the check. " +
		"A warning is a finding to fix, not a failure of the tool: fix the source and check it again.",
	InputSchema: sourceSchema,
}

type rule struct {
	Name        string `json:"rule"`
	Description string `json:"description"`
}

type ruleInput struct {
	Rule string `json:"rule" jsonschema:"The name of a lint rule, as the rule key of a diagnostic record gives it, for example state-obsession."`
}

var explainRuleTool = &sdk.Tool{
	Name: "emod_explain_rule",
	Description: "Explain a lint rule of emod: why the rule exists and how to fix a finding of the rule. " +
		"Give the name from the rule key of a diagnostic record of emod_check. " +
		"The result has rule and description, and the description is the text that `emod lint --explain <rule>` prints. " +
		"The resource emod://rules lists each rule with its description.",
}

var rulesResource = &sdk.Resource{
	URI:         "emod://rules",
	Name:        "rules",
	Title:       "Lint rules of emod",
	MIMEType:    "application/json",
	Description: `Each lint rule that emod can report, with its description, as {"rules":[{"rule":...,"description":...}]}.`,
}

var schemaTool = &sdk.Tool{
	Name: "emod_schema",
	Description: "Give the CUE schema of emod, the same text that `emod schema` prints. " +
		"The resource emod://schema gives the same text.",
}

var schemaResource = &sdk.Resource{
	URI:         "emod://schema",
	Name:        "schema",
	Title:       "CUE schema of emod",
	MIMEType:    "text/plain",
	Description: "The CUE schema of emod, the same text that `emod schema` prints.",
}

var dslReferenceResource = &sdk.Resource{
	URI:      "emod://docs/dsl-reference",
	Name:     "dsl-reference.md",
	Title:    "DSL reference of emod",
	MIMEType: "text/markdown",
	Description: "The reference of the emod language: each construct, its syntax, the slice patterns and the specs, " +
		"for the version of emod that serves it.",
}

type slicesResult struct {
	Slices []ast.SliceRecord `json:"slices"`
}

type parseDiagnostics struct {
	Diagnostics []diagnostic.Record `json:"diagnostics"`
}

var slicesListTool = &sdk.Tool{
	Name: "emod_slices_list",
	Description: "List the slices of an .emod model: one record for each slice in source order, with name, pattern, context and keyElements, " +
		"the same records as `emod slices list --format json`. " +
		"Give the source in content, or give the path of an .emod file under the root of the server in path, as for emod_check. " +
		"The pattern is command, view, automation, translation or unknown, and keyElements names the parts that make the slice that pattern. " +
		"A source that does not parse gives its parse diagnostics in diagnostics, with no slices.",
	InputSchema:  sourceSchema,
	OutputSchema: oneOf[slicesResult, parseDiagnostics](),
}

var exportTool = &sdk.Tool{
	Name: "emod_export",
	Description: "Export an .emod model, the same output as `emod export --format <format>`. " +
		"Give the source in content, or give the path of an .emod file under the root of the server in path, as for emod_check. " +
		"json is the document of the model, and diagram-json holds each node and each edge of the diagram; " +
		"both carry the diagnostics of the check, also for a source with errors. " +
		"cue gives the CUE text of a source with no diagnostic; a source with any diagnostic, also a warning, gives the diagnostics in diagnostics and no CUE.",
	InputSchema: formatSchema("json", []string{"json", "diagram-json", "cue"}),
}

var glossaryTool = &sdk.Tool{
	Name: "emod_glossary",
	Description: "Give the glossary of an .emod model, the terms that the model defines, the same text as `emod glossary --format <format>`. " +
		"Give the source in content, or give the path of an .emod file under the root of the server in path, as for emod_check. " +
		"markdown gives the glossary as one text block, and json gives the same terms as a document. " +
		"A source that does not parse gives its parse diagnostics in diagnostics, with no glossary.",
	InputSchema: formatSchema("markdown", []string{"markdown", "json"}),
}

type diagramResult struct {
	Diagram     string              `json:"diagram"`
	Diagnostics []diagnostic.Record `json:"diagnostics"`
}

var diagramTool = &sdk.Tool{
	Name: "emod_diagram",
	Description: "Draw an .emod model, the same text that `emod diagram --format <format>` gives. " +
		"Give the source in content, or give the path of an .emod file under the root of the server in path, as for emod_check. " +
		"mermaid and ascii are text for a chat; svg and event-flow are SVG text, which the tool does not write to a file. " +
		"A source with an error gives its diagnostics and no diagram. " +
		"A source with only warnings gives the diagram and the warnings, under diagram and diagnostics.",
	InputSchema:  diagramSchema(),
	OutputSchema: oneOf[diagramResult, checkResult](),
}

type fmtResult struct {
	Formatted string `json:"formatted"`
	Changed   bool   `json:"changed"`
}

var fmtTool = &sdk.Tool{
	Name: "emod_fmt",
	Description: "Format an .emod model in the canonical layout, the same text that `emod fmt` writes. " +
		"Give the source in content, or give the path of an .emod file under the root of the server in path, as for emod_check. " +
		"formatted is the text in the canonical layout, and changed is true when formatted differs from the source, " +
		"the same answer as the exit code of `emod fmt --check`. " +
		"The tool writes the file only for a call with write: true, path and no content, on a server that runs `emod mcp --allow-write`, and only when changed is true; " +
		"else write formatted with the file tool of the host. " +
		"A source that does not parse gives its parse diagnostics in diagnostics, with no formatted text.",
	InputSchema:  writeSchema(),
	OutputSchema: oneOf[fmtResult, parseDiagnostics](),
}

type arrangeResult struct {
	Arranged       string              `json:"arranged"`
	Changed        bool                `json:"changed"`
	Moved          int                 `json:"moved"`
	BackwardBefore int                 `json:"backwardBefore"`
	BackwardAfter  int                 `json:"backwardAfter"`
	Backward       []backwardReference `json:"backward"`
}

type backwardReference struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	From  string `json:"from"`
	To    string `json:"to"`
}

var slicesArrangeTool = &sdk.Tool{
	Name: "emod_slices_arrange",
	Description: "Arrange the slices of an .emod model so that each reference points forward, the same text that `emod slices arrange` writes. " +
		"Give the source in content, or give the path of an .emod file under the root of the server in path, as for emod_check. " +
		"Only view slices move. arranged is the text with the slices in the new order, in the canonical layout; when no slice moves, arranged is the source as it is. " +
		"changed is true when a slice moves, the same answer as the exit code of `emod slices arrange --check`. " +
		"moved, backwardBefore and backwardAfter are the counts of the report that the command prints, and backward lists each reference that still points backward, " +
		"with kind, label, and the names of the slices in from and to. Only view slices move, so the arrange cannot fix such a reference, for example a reference between two slices that are not views. " +
		"The tool writes the file only for a call with write: true, path and no content, on a server that runs `emod mcp --allow-write`, and only when changed is true; " +
		"else write arranged with the file tool of the host. " +
		"A source that does not parse gives its parse diagnostics in diagnostics, with no report.",
	InputSchema:  writeSchema(),
	OutputSchema: oneOf[arrangeResult, parseDiagnostics](),
}

type Server struct {
	in         io.ReadCloser
	out        io.WriteCloser
	root       Root
	allowWrite bool
}

func NewServer(in io.ReadCloser, out io.WriteCloser, root Root, allowWrite bool) *Server {
	return &Server{in: in, out: out, root: root, allowWrite: allowWrite}
}

func (s *Server) Run(ctx context.Context) error {
	server := sdk.NewServer(
		&sdk.Implementation{Name: "emod", Version: version.Current()},
		&sdk.ServerOptions{Capabilities: &sdk.ServerCapabilities{}},
	)
	sdk.AddTool(server, checkTool, s.check)
	sdk.AddTool(server, explainRuleTool, s.explainRule)
	server.AddResource(rulesResource, s.readRules)
	sdk.AddTool(server, schemaTool, s.schema)
	server.AddResource(schemaResource, s.readText(cue.Schema))
	for _, example := range examples.All() {
		server.AddResource(&sdk.Resource{
			URI:         "emod://examples/" + example.Name,
			Name:        example.Name + ".emod",
			Title:       "Example " + example.Name,
			MIMEType:    "text/plain",
			Description: example.Description,
		}, s.readText(example.Source))
	}
	server.AddResource(dslReferenceResource, s.readText(docs.DSLReference))
	sdk.AddTool(server, slicesListTool, s.listSlices)
	sdk.AddTool(server, exportTool, s.exportModel)
	sdk.AddTool(server, glossaryTool, s.renderGlossary)
	sdk.AddTool(server, diagramTool, s.drawDiagram)
	sdk.AddTool(server, fmtTool, s.formatModel)
	sdk.AddTool(server, slicesArrangeTool, s.arrangeSlices)
	return server.Run(ctx, &sdk.IOTransport{Reader: s.in, Writer: s.out})
}

func (s *Server) check(_ context.Context, _ *sdk.CallToolRequest, input sourceInput) (*sdk.CallToolResult, checkResult, error) {
	source, name, err := s.source(input)
	if err != nil {
		return nil, checkResult{}, err
	}
	return nil, checkResult{Diagnostics: diagnostic.Records(oracle.Check(source, name))}, nil
}

func (s *Server) source(input sourceInput) (source string, name string, err error) {
	if input.Content != nil {
		return *input.Content, cmp.Or(input.Name, input.Path, inlineName), nil
	}
	if input.Path == "" {
		return "", "", errors.New("the call needs content or path")
	}
	source, err = s.root.Read(input.Path)
	return source, input.Path, err
}

func (s *Server) explainRule(_ context.Context, _ *sdk.CallToolRequest, input ruleInput) (*sdk.CallToolResult, rule, error) {
	description, err := linter.RuleDescription(input.Rule)
	if err != nil {
		return nil, rule{}, err
	}
	explained := rule{Name: input.Rule, Description: description}
	result, err := s.jsonResult(explained)
	if err != nil {
		return nil, rule{}, err
	}
	return result, explained, nil
}

func (s *Server) listSlices(_ context.Context, _ *sdk.CallToolRequest, input sourceInput) (*sdk.CallToolResult, any, error) {
	source, name, err := s.source(input)
	if err != nil {
		return nil, nil, err
	}
	model, diagnostics := oracle.Parse(source, name)
	if len(diagnostics) > 0 {
		return s.structuredResult(parseDiagnostics{Diagnostics: diagnostic.Records(diagnostics)})
	}
	return s.structuredResult(slicesResult{Slices: ast.SliceRecords(model)})
}

func (s *Server) exportModel(_ context.Context, _ *sdk.CallToolRequest, input formatInput) (*sdk.CallToolResult, any, error) {
	source, name, err := s.source(input.sourceInput)
	if err != nil {
		return nil, nil, err
	}
	model, diagnostics := oracle.Run(source, name)
	switch input.Format {
	case "diagram-json":
		return s.documentResult(export.ExportDiagramJSONDiagnostics(model, diagnostics))
	case "cue":
		if len(diagnostics) > 0 {
			return s.structuredResult(checkResult{Diagnostics: diagnostic.Records(diagnostics)})
		}
		text, err := export.ExportCUE(model)
		if err != nil {
			return nil, nil, err
		}
		return s.textResult(string(text)), nil, nil
	default:
		return s.documentResult(export.ExportJSONDiagnostics(model, diagnostics))
	}
}

func (s *Server) renderGlossary(_ context.Context, _ *sdk.CallToolRequest, input formatInput) (*sdk.CallToolResult, any, error) {
	source, name, err := s.source(input.sourceInput)
	if err != nil {
		return nil, nil, err
	}
	model, diagnostics := oracle.Parse(source, name)
	if len(diagnostics) > 0 {
		return s.structuredResult(parseDiagnostics{Diagnostics: diagnostic.Records(diagnostics)})
	}
	if input.Format == "json" {
		return s.documentResult(glossary.RenderJSON(model))
	}
	text, err := glossary.RenderMarkdown(model)
	if err != nil {
		return nil, nil, err
	}
	return s.textResult(string(text)), nil, nil
}

func (s *Server) drawDiagram(_ context.Context, _ *sdk.CallToolRequest, input diagramInput) (*sdk.CallToolResult, any, error) {
	source, name, err := s.source(input.sourceInput)
	if err != nil {
		return nil, nil, err
	}
	model, diagnostics := oracle.Run(source, name)
	if slices.ContainsFunc(diagnostics, func(d *diagnostic.Entry) bool { return d.Severity == diagnostic.Error }) {
		return s.structuredResult(checkResult{Diagnostics: diagnostic.Records(diagnostics)})
	}
	if input.Specs && !diagram.DrawsSpecCards(input.Format) {
		return nil, nil, fmt.Errorf("specs is not supported for format %q; only svg draws spec cards", input.Format)
	}
	style, err := diagram.ParseStyle(input.Style)
	if err != nil {
		return nil, nil, err
	}
	var options []diagram.Option
	if input.Specs {
		options = append(options, diagram.WithSpecs())
	}
	drawn, err := diagram.Export(model, input.Format, style, options...)
	if err != nil {
		return nil, nil, err
	}

	result := s.textResult(string(drawn))
	if len(diagnostics) > 0 {
		lines := make([]string, 0, len(diagnostics))
		for _, d := range diagnostics {
			lines = append(lines, d.String())
		}
		result.Content = append(result.Content, &sdk.TextContent{Text: strings.Join(lines, "\n")})
	}
	return result, diagramResult{Diagram: string(drawn), Diagnostics: diagnostic.Records(diagnostics)}, nil
}

func (s *Server) formatModel(_ context.Context, _ *sdk.CallToolRequest, input writeInput) (*sdk.CallToolResult, any, error) {
	if err := s.checkWrite(input); err != nil {
		return nil, nil, err
	}
	source, name, err := s.source(input.sourceInput)
	if err != nil {
		return nil, nil, err
	}
	model, diagnostics := oracle.Parse(source, name)
	if len(diagnostics) > 0 {
		return s.structuredResult(parseDiagnostics{Diagnostics: diagnostic.Records(diagnostics)})
	}
	formatted := formatter.Format(model)
	changed := formatted != source
	if err := s.writeChange(input, source, formatted); err != nil {
		return nil, nil, err
	}
	return s.structuredResult(fmtResult{Formatted: formatted, Changed: changed})
}

func (s *Server) arrangeSlices(_ context.Context, _ *sdk.CallToolRequest, input writeInput) (*sdk.CallToolResult, any, error) {
	if err := s.checkWrite(input); err != nil {
		return nil, nil, err
	}
	source, name, err := s.source(input.sourceInput)
	if err != nil {
		return nil, nil, err
	}
	model, diagnostics := oracle.Parse(source, name)
	if len(diagnostics) > 0 {
		return s.structuredResult(parseDiagnostics{Diagnostics: diagnostic.Records(diagnostics)})
	}
	report := arrange.Model(model)
	arranged := source
	if report.Changed() {
		arranged = formatter.Format(model)
	}
	if err := s.writeChange(input, source, arranged); err != nil {
		return nil, nil, err
	}
	backward := make([]backwardReference, 0, len(report.Backward))
	for _, r := range report.Backward {
		backward = append(backward, backwardReference{Kind: r.Kind, Label: r.Label, From: r.From.Name, To: r.To.Name})
	}
	return s.structuredResult(arrangeResult{
		Arranged:       arranged,
		Changed:        report.Changed(),
		Moved:          report.Moved,
		BackwardBefore: report.BackwardBefore,
		BackwardAfter:  report.BackwardAfter,
		Backward:       backward,
	})
}

func (s *Server) checkWrite(input writeInput) error {
	if !input.Write {
		return nil
	}
	if !s.allowWrite {
		return errors.New("write needs a server that allows writes: start it with emod mcp --allow-write")
	}
	if input.Content != nil || input.Path == "" {
		return errors.New("write needs path and no content: the tool writes only a new layout of the file on disk")
	}
	return nil
}

func (s *Server) writeChange(input writeInput, source, text string) error {
	if !input.Write || text == source {
		return nil
	}
	return s.root.Replace(input.Path, source, text)
}

func (s *Server) documentResult(document []byte, err error) (*sdk.CallToolResult, any, error) {
	if err != nil {
		return nil, nil, err
	}
	return s.structuredResult(json.RawMessage(document))
}

func (s *Server) structuredResult(value any) (*sdk.CallToolResult, any, error) {
	result, err := s.jsonResult(value)
	if err != nil {
		return nil, nil, err
	}
	return result, value, nil
}

func (s *Server) readRules(_ context.Context, _ *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
	described := linter.Rules()
	rules := make([]rule, 0, len(described))
	for _, r := range described {
		rules = append(rules, rule{Name: r.Name, Description: r.Description})
	}
	text, err := s.jsonText(struct {
		Rules []rule `json:"rules"`
	}{rules})
	if err != nil {
		return nil, err
	}
	return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{Text: text}}}, nil
}

func (s *Server) schema(_ context.Context, _ *sdk.CallToolRequest, _ struct{}) (*sdk.CallToolResult, any, error) {
	return s.textResult(cue.Schema), nil, nil
}

func (s *Server) readText(text string) sdk.ResourceHandler {
	return func(context.Context, *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
		return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{Text: text}}}, nil
	}
}

func (s *Server) jsonResult(value any) (*sdk.CallToolResult, error) {
	text, err := s.jsonText(value)
	if err != nil {
		return nil, err
	}
	return s.textResult(text), nil
}

func (s *Server) textResult(text string) *sdk.CallToolResult {
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}}
}

func (s *Server) jsonText(value any) (string, error) {
	var text bytes.Buffer
	encoder := json.NewEncoder(&text)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return text.String(), nil
}

func oneOf[First, Second any]() *jsonschema.Schema {
	first, err := jsonschema.For[First](nil)
	if err != nil {
		panic(err)
	}
	second, err := jsonschema.For[Second](nil)
	if err != nil {
		panic(err)
	}
	return &jsonschema.Schema{Type: "object", OneOf: []*jsonschema.Schema{first, second}}
}

func formatSchema(defaultFormat string, formats []string) *jsonschema.Schema {
	enum := make([]any, 0, len(formats))
	for _, format := range formats {
		enum = append(enum, format)
	}
	properties := maps.Clone(sourceSchema.Properties)
	properties["format"] = &jsonschema.Schema{
		Type:        "string",
		Description: "The output format. The default is " + defaultFormat + ".",
		Enum:        enum,
		Default:     json.RawMessage(`"` + defaultFormat + `"`),
	}
	return &jsonschema.Schema{
		Type:                 "object",
		Properties:           properties,
		AdditionalProperties: sourceSchema.AdditionalProperties,
	}
}

func writeSchema() *jsonschema.Schema {
	properties := maps.Clone(sourceSchema.Properties)
	properties["write"] = &jsonschema.Schema{
		Type: "boolean",
		Description: "Write the result to the file in path when changed is true. " +
			"The server must run `emod mcp --allow-write`, and the call must give path and no content.",
		Default: json.RawMessage(`false`),
	}
	return &jsonschema.Schema{
		Type:                 "object",
		Properties:           properties,
		AdditionalProperties: sourceSchema.AdditionalProperties,
	}
}

func diagramSchema() *jsonschema.Schema {
	schema := formatSchema("svg", diagram.Formats())
	schema.Properties["style"] = &jsonschema.Schema{
		Type:        "string",
		Description: "The layout style. The default is auto, which picks the style of each context from its mode.",
		Enum:        []any{"auto", "projected", "dcb"},
		Default:     json.RawMessage(`"auto"`),
	}
	schema.Properties["specs"] = &jsonschema.Schema{
		Type:        "boolean",
		Description: "Draw the Given-When-Then cards of the specs. Only svg draws them.",
		Default:     json.RawMessage(`false`),
	}
	return schema
}
