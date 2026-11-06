//go:build unit

package mcp_test

import (
	"go/build"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/test"
)

const (
	module      = "github.com/hpcsc/emod"
	providerSDK = "github.com/anthropics/anthropic-sdk-go"
	mcpSDK      = "github.com/modelcontextprotocol/go-sdk"
)

func TestIsolation(t *testing.T) {
	t.Run("imports", func(t *testing.T) {
		t.Run("the MCP server reaches the oracle but no AI package and no provider SDK", func(t *testing.T) {
			reached := test.ImportsReachedFrom(t, build.Default.GOOS, build.Default.GOARCH, module+"/internal/mcp")

			require.True(t, reached[module+"/internal/oracle"], "the walk must reach the packages the MCP server uses")
			test.RequireNoneWithPrefix(t, reached, module+"/internal/llm", providerSDK)
		})

		t.Run("the WASM build reaches neither the MCP server nor the MCP SDK", func(t *testing.T) {
			reached := test.ImportsReachedFrom(t, "js", "wasm", module+"/cmd/emod-wasm")

			require.True(t, reached[module+"/internal/oracle"], "the walk must reach the packages the WASM build uses")
			test.RequireNoneWithPrefix(t, reached, module+"/internal/mcp", mcpSDK)
		})

		t.Run("the desktop build reaches neither the MCP server nor the MCP SDK", func(t *testing.T) {
			reached := test.ImportsReachedFrom(t, build.Default.GOOS, build.Default.GOARCH, module+"/cmd/emod-desktop")

			require.True(t, reached[module+"/internal/desktop"], "the walk must reach the packages the desktop build uses")
			test.RequireNoneWithPrefix(t, reached, module+"/internal/mcp", mcpSDK)
		})

		t.Run("no package but the MCP server imports the MCP SDK", func(t *testing.T) {
			require.Equal(t, []string{module + "/internal/mcp"}, test.PackagesImporting(t, mcpSDK))
		})
	})
}
