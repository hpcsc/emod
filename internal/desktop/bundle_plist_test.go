//go:build unit

package desktop_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// macOS reads the packaged app entirely through Contents/Info.plist: it runs
// the executable that file names, shows the name that file carries, and files
// the bundle under the identifier that file declares. Nothing connects the
// plist to the task that assembles the bundle around it, so renaming the built
// binary on one side alone leaves every suite green and produces a bundle that
// dies at launch with "You can't open the application because it may be
// damaged". This reads both sides and requires them to agree, the way
// TestBindingNames pins the frontend's calls against the Go methods.
func TestBundlePlist(t *testing.T) {
	t.Run("the bundle runs the binary the desktop build produces", func(t *testing.T) {
		built := builtBinaryName(t)
		declared := plistString(t, "CFBundleExecutable")

		require.Equal(t, built, declared,
			"build:desktop writes a binary Contents/Info.plist does not name")
	})

	t.Run("the packaging task installs the binary under the name the plist declares", func(t *testing.T) {
		installed := installedBinaryName(t)
		declared := plistString(t, "CFBundleExecutable")

		require.Equal(t, declared, installed,
			"package:desktop puts the binary in Contents/MacOS under a name Info.plist does not name")
	})

	t.Run("the packaging task installs the plist the repository tracks", func(t *testing.T) {
		body := taskBody(t, "package:desktop")

		require.Contains(t, body, plistPath,
			"package:desktop must install the tracked plist as the bundle's Contents/Info.plist")
		require.FileExists(t, repoPath(plistPath))
	})

	t.Run("the Dock and the app switcher are told to say emod", func(t *testing.T) {
		require.Equal(t, "emod", plistString(t, "CFBundleName"))
	})

	t.Run("the bundle keeps the identifier macOS has already filed it under", func(t *testing.T) {
		require.Equal(t, "io.github.hpcsc.emod", plistString(t, "CFBundleIdentifier"),
			"every machine that has run the app has this identifier recorded in LaunchServices, "+
				"and the file associations of a later story bind to it; changing it strands both")
	})
}

const (
	plistPath    = "build/darwin/Info.plist"
	taskfilePath = "Taskfile.yml"
)

// The value of a top-level plist key. Property lists put the value in the
// element after the key rather than inside it, so the pairing is positional and
// a key whose value moved reads here as a key that is not there at all.
func plistString(t *testing.T, key string) string {
	t.Helper()

	pattern := regexp.MustCompile(`<key>` + regexp.QuoteMeta(key) + `</key>\s*<string>([^<]*)</string>`)
	match := pattern.FindStringSubmatch(readRepoFile(t, plistPath))
	require.Len(t, match, 2, plistPath+" declares no string value for "+key)

	value := match[1]
	require.NotEmpty(t, value, plistPath+" leaves "+key+" empty")

	return value
}

func builtBinaryName(t *testing.T) string {
	t.Helper()

	return captureIn(t, taskBody(t, "build:desktop"),
		regexp.MustCompile(`go build [^\n]*-o \./bin/(\S+)`),
		"build:desktop must build the desktop binary into ./bin")
}

func installedBinaryName(t *testing.T) string {
	t.Helper()

	return captureIn(t, taskBody(t, "package:desktop"),
		regexp.MustCompile(`Contents/MacOS/(\S+)`),
		"package:desktop must copy the binary into the bundle's Contents/MacOS")
}

// A task's own lines, from its name to the next task's. Comment lines are
// indented alongside the task names they introduce, so the boundary skips them
// rather than ending a task at the comment that describes the following one.
func taskBody(t *testing.T, name string) string {
	t.Helper()

	pattern := regexp.MustCompile(`(?ms)^  ` + regexp.QuoteMeta(name) + `:$(.*?)(?:^  [^ #]|\z)`)

	return captureIn(t, readRepoFile(t, taskfilePath), pattern, taskfilePath+" declares no task "+name)
}

func captureIn(t *testing.T, text string, pattern *regexp.Regexp, missing string) string {
	t.Helper()

	match := pattern.FindStringSubmatch(text)
	require.Len(t, match, 2, missing)
	require.NotEmpty(t, match[1], missing)

	return match[1]
}

func readRepoFile(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(repoPath(path))
	require.NoError(t, err)

	return string(raw)
}

func repoPath(path string) string {
	return "../../" + path
}
