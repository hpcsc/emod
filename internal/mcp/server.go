package mcp

import (
	"context"
	"io"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/version"
)

const inlineName = "inline.emod"

type sourceInput struct {
	Content string `json:"content" jsonschema:"The .emod source text to check."`
	Name    string `json:"name,omitempty" jsonschema:"The file name that each record carries. The default is inline.emod."`
}

type checkResult struct {
	Diagnostics []diagnostic.Record `json:"diagnostics"`
}

var checkTool = &sdk.Tool{
	Name: "emod_check",
	Description: "Check .emod source with the parser, the validator and the linter of emod. " +
		"The result is a list of diagnostics, the same records as `emod validate --format json`: " +
		"file, line, rule (only when a rule name goes with the diagnostic), severity and message. " +
		"An empty list means that the source passes the check. " +
		"A warning is a finding to fix, not a failure of the tool: fix the source and check it again.",
}

type Server struct {
	in  io.ReadCloser
	out io.WriteCloser
}

func NewServer(in io.ReadCloser, out io.WriteCloser) *Server {
	return &Server{in: in, out: out}
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
	name := input.Name
	if name == "" {
		name = inlineName
	}
	return nil, checkResult{Diagnostics: diagnostic.Records(oracle.Check(input.Content, name))}, nil
}
