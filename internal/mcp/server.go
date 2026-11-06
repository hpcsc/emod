package mcp

import (
	"cmp"
	"context"
	"errors"
	"io"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hpcsc/emod/internal/diagnostic"
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
