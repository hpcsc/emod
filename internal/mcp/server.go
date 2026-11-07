package mcp

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hpcsc/emod/docs"
	"github.com/hpcsc/emod/examples"
	"github.com/hpcsc/emod/internal/cue"
	"github.com/hpcsc/emod/internal/diagnostic"
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
			Description: "The .emod source text to check.",
		},
		"path": {
			Type: "string",
			Description: "The path of an .emod file under the root of the server, relative to the root or absolute. " +
				"The tool reads the file only when the call gives no content.",
		},
		"name": {
			Type:        "string",
			Description: "The file name that the records of content carry. The default is path, else inline.emod.",
		},
	},
	AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
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

type Server struct {
	in   io.ReadCloser
	out  io.WriteCloser
	root Root
}

func NewServer(in io.ReadCloser, out io.WriteCloser, root Root) *Server {
	return &Server{in: in, out: out, root: root}
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
		return "", "", errors.New("emod_check needs content or path")
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
	text, err := s.jsonText(explained)
	if err != nil {
		return nil, rule{}, err
	}
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}}, explained, nil
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
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: cue.Schema}}}, nil, nil
}

func (s *Server) readText(text string) sdk.ResourceHandler {
	return func(context.Context, *sdk.ReadResourceRequest) (*sdk.ReadResourceResult, error) {
		return &sdk.ReadResourceResult{Contents: []*sdk.ResourceContents{{Text: text}}}, nil
	}
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
