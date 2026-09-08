//go:build unit

package desktop_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// cmd/emod-desktop is the only package here that links CGO, and it does not
// compile at all until `task build:desktop` has assembled the directory its
// //go:embed names. Everything that builds or releases the CLI and the web
// viewer therefore has to keep it at arm's length, and each way of failing to
// is invisible on a machine that has just run the desktop build: `go list`
// prints nothing at all when a package will not load, so a pipeline that drops
// -e hands `go test` no packages and tests the repository root instead, and a
// release that reaches for the desktop command acquires a C toolchain the
// ubuntu job does not have.
func TestDesktopIsolation(t *testing.T) {
	t.Run("the test package lists survive a desktop package that cannot load", func(t *testing.T) {
		taskfile := readRepoFile(t, taskfilePath)
		enumerations := strings.Count(taskfile, "go list")

		require.NotZero(t, enumerations, taskfilePath+" must enumerate the test packages with go list")
		require.Equal(t, enumerations, strings.Count(taskfile, "go list -e"),
			"a go list without -e prints nothing when cmd/emod-desktop cannot load, "+
				"and the test task then runs against the repository root")
	})

	t.Run("the test package lists exclude the desktop command", func(t *testing.T) {
		for _, task := range []string{"test:unit", "test:integration"} {
			require.Contains(t, taskBody(t, task), "-e /cmd/emod-desktop",
				task+" must exclude the package whose embed target only the desktop build writes")
		}
	})

	t.Run("building the CLI and the web viewer reaches no desktop target", func(t *testing.T) {
		for _, task := range []string{"build", "build:wasm", "build:web"} {
			require.NotContains(t, taskBody(t, task), "desktop",
				task+" must not depend on the desktop build")
		}
	})

	t.Run("the CLI and the web viewer are built with cgo off", func(t *testing.T) {
		for _, task := range []string{"build", "build:wasm"} {
			require.Contains(t, taskBody(t, task), "CGO_ENABLED: '0'",
				task+" must keep producing an artifact that needs no C toolchain")
		}
		for _, task := range []string{"build", "build:wasm", "build:web"} {
			require.NotContains(t, taskBody(t, task), "CGO_ENABLED: '1'",
				task+" must not turn on the cgo only the desktop build needs")
		}
	})

	// on-demand-build.yml is left out on purpose: it is where the downloadable
	// desktop artifacts are meant to land, while these two are the everyday push
	// and the CLI release, which the desktop build must never lengthen.
	t.Run("the everyday CI and the CLI release reach no desktop target", func(t *testing.T) {
		for _, workflow := range []string{".github/workflows/ci.yml", ".github/workflows/release.yml", "Taskfile.release.yml"} {
			require.NotContains(t, readRepoFile(t, workflow), "desktop",
				workflow+" must not build the desktop app: it needs a C toolchain and a macOS runner")
		}
	})

	t.Run("the release builds the CLI command alone, with cgo off", func(t *testing.T) {
		release := readRepoFile(t, ".goreleaser.yaml")

		require.Contains(t, release, "main: ./cmd/emod\n",
			"the release must build the CLI command, not the desktop one")
		require.Contains(t, release, "CGO_ENABLED=0",
			"the release must stay free of the C toolchain the desktop app needs")
		require.NotContains(t, release, "emod-desktop",
			"a second build entry for the desktop command leaves the two checks above green")
		require.NotContains(t, release, "CGO_ENABLED=1",
			"a release entry that turns cgo on needs the toolchain this release does without")
	})
}
