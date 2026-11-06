//go:build unit

package llm_test

import (
	"go/build"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/test"
)

const (
	module      = "github.com/hpcsc/emod"
	providerSDK = "github.com/anthropics/anthropic-sdk-go"
)

func TestIsolation(t *testing.T) {
	t.Run("imports", func(t *testing.T) {
		t.Run("the WASM build reaches the oracle but no AI package and no provider SDK", func(t *testing.T) {
			reached := test.ImportsReachedFrom(t, "js", "wasm", module+"/cmd/emod-wasm")

			require.True(t, reached[module+"/internal/oracle"], "the walk must reach the packages the WASM build uses")
			test.RequireNoneWithPrefix(t, reached, module+"/internal/llm", providerSDK)
		})

		t.Run("the repair loop reaches the oracle but no provider adapter and no provider SDK", func(t *testing.T) {
			reached := test.ImportsReachedFrom(t, build.Default.GOOS, build.Default.GOARCH, module+"/internal/llm/repair")

			require.True(t, reached[module+"/internal/oracle"], "the walk must reach the packages the repair loop uses")
			test.RequireNoneWithPrefix(t, reached, module+"/internal/llm/anthropic", module+"/internal/llm/openai", providerSDK)
		})
	})
}
