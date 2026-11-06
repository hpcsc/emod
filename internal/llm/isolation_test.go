//go:build unit

package llm_test

import (
	"go/build"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	module      = "github.com/hpcsc/emod"
	providerSDK = "github.com/anthropics/anthropic-sdk-go"
)

func importsReachedFrom(t *testing.T, goos, goarch, start string) map[string]bool {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	buildContext := build.Default
	buildContext.GOOS = goos
	buildContext.GOARCH = goarch
	buildContext.CgoEnabled = false

	reached := map[string]bool{}
	queue := []string{start}
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		if reached[path] {
			continue
		}
		reached[path] = true

		relative, inModule := strings.CutPrefix(path, module+"/")
		if !inModule {
			continue
		}
		pkg, err := buildContext.ImportDir(filepath.Join(root, relative), 0)
		require.NoError(t, err, path)
		queue = append(queue, pkg.Imports...)
	}

	return reached
}

func requireNoneWithPrefix(t *testing.T, reached map[string]bool, prefixes ...string) {
	t.Helper()

	for path := range reached {
		for _, prefix := range prefixes {
			require.False(t, path == prefix || strings.HasPrefix(path, prefix+"/"), "%s must not be reached, but %s is", prefix, path)
		}
	}
}

func TestIsolation(t *testing.T) {
	t.Run("imports", func(t *testing.T) {
		t.Run("the WASM build reaches the oracle but no AI package and no provider SDK", func(t *testing.T) {
			reached := importsReachedFrom(t, "js", "wasm", module+"/cmd/emod-wasm")

			require.True(t, reached[module+"/internal/oracle"], "the walk must reach the packages the WASM build uses")
			requireNoneWithPrefix(t, reached, module+"/internal/llm", providerSDK)
		})

		t.Run("the repair loop reaches the oracle but no provider adapter and no provider SDK", func(t *testing.T) {
			reached := importsReachedFrom(t, build.Default.GOOS, build.Default.GOARCH, module+"/internal/llm/repair")

			require.True(t, reached[module+"/internal/oracle"], "the walk must reach the packages the repair loop uses")
			requireNoneWithPrefix(t, reached, module+"/internal/llm/anthropic", module+"/internal/llm/openai", providerSDK)
		})
	})
}
