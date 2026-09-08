//go:build unit

package desktop_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// macOS reads the packaged app entirely through Contents/Info.plist: it runs
// the executable that file names, shows the name and icon that file points at,
// and files the bundle under the identifier that file declares. Nothing
// connects those names to the build and the packaging task that produce what
// they name, so renaming the built binary on one side alone leaves every suite
// green and produces a bundle that dies at launch with "You can't open the
// application because it may be damaged". This reads both sides and requires
// them to agree, the way TestBindingNames pins the frontend's calls against the
// Go methods. Whether the task assembles the bundle correctly around the plist
// is TestPackagingTask's subject, not this one's.
func TestBundlePlist(t *testing.T) {
	t.Run("the bundle runs the binary the desktop build produces", func(t *testing.T) {
		built := binaryNameIn(t, "build:desktop", regexp.MustCompile(`go build [^\n]*-o \./bin/(\S+)`))
		declared := plistString(t, "CFBundleExecutable")

		require.Equal(t, built, declared,
			"build:desktop writes a binary Contents/Info.plist does not name")
	})

	t.Run("the packaging task installs the binary under the name the plist declares", func(t *testing.T) {
		installed := binaryNameIn(t, "package:desktop", regexp.MustCompile(`Contents/MacOS/(\S+)`))
		declared := plistString(t, "CFBundleExecutable")

		require.Equal(t, declared, installed,
			"package:desktop puts the binary in Contents/MacOS under a name Info.plist does not name")
	})

	t.Run("the Dock and the app switcher are told to say emod", func(t *testing.T) {
		require.Equal(t, "emod", plistString(t, "CFBundleName"))
	})

	t.Run("the packaging task writes the icon file the plist names", func(t *testing.T) {
		written := captureIn(t, taskBody(t, "package:desktop"),
			regexp.MustCompile(`Contents/Resources/(\S+)\.icns`),
			"package:desktop must write the bundle's icon into Contents/Resources")

		require.Equal(t, plistString(t, "CFBundleIconFile"), written,
			"macOS shows the icon Info.plist names, and package:desktop writes a different one")
	})

	t.Run("the bundle keeps the identifier macOS has already filed it under", func(t *testing.T) {
		require.Equal(t, "au.pnguyen.emod", plistString(t, "CFBundleIdentifier"),
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

func binaryNameIn(t *testing.T, task string, pattern *regexp.Regexp) string {
	t.Helper()

	return captureIn(t, taskBody(t, task), pattern, taskfilePath+"'s "+task+" must name the desktop binary")
}

// A task's own lines, ending at whatever sits at the next task's indentation —
// a comment there introduces the task below it, so a body that ran past one
// would carry prose describing a different task and answer for it.
func taskBody(t *testing.T, name string) string {
	t.Helper()

	pattern := regexp.MustCompile(`(?ms)^  ` + regexp.QuoteMeta(name) + `:$(.*?)(?:^  \S|\z)`)

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
