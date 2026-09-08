//go:build unit

package desktop_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A macOS application is a directory, and `package:desktop` is the only thing
// that builds this one: every piece has to arrive at the path macOS reads it
// from, all of them under one bundle, and the signature has to come last so it
// seals what the steps before it copied in. None of that is checked by running
// the task — a green suite says nothing about a bundle nobody assembled — so
// this reads the task as the recipe it is and pins the parts a later edit can
// silently break.
func TestPackagingTask(t *testing.T) {
	t.Run("the bundle carries the binary the desktop build produced", func(t *testing.T) {
		built := binaryNameIn(t, "build:desktop", regexp.MustCompile(`go build [^\n]*-o \./bin/(\S+)`))
		bundled := binaryNameIn(t, "package:desktop",
			regexp.MustCompile(`cp \./bin/(\S+) `+regexp.QuoteMeta(bundlePath)+`/Contents/MacOS/`))

		require.Equal(t, built, bundled,
			"package:desktop bundles a binary that is not the one build:desktop writes")
	})

	t.Run("packaging builds the binary it bundles", func(t *testing.T) {
		require.Regexp(t, `deps:\s*\n\s*- build:desktop\b`, taskBody(t, "package:desktop"),
			"without the dependency the task bundles whatever ./bin happens to hold, or nothing at all")
	})

	t.Run("every step builds up one bundle", func(t *testing.T) {
		named := regexp.MustCompile(`\./bin/\S*\.app`).FindAllString(taskBody(t, "package:desktop"), -1)

		require.NotEmpty(t, named, "package:desktop must assemble a bundle under ./bin")
		require.Equal(t, []string{bundlePath}, unique(named),
			"package:desktop spreads its steps over more than one bundle")
	})

	t.Run("each piece lands where macOS looks for it", func(t *testing.T) {
		body := taskBody(t, "package:desktop")

		require.Contains(t, body, bundlePath+"/Contents/Info.plist",
			"macOS reads a bundle's plist from Contents/Info.plist and nowhere else")
		require.Contains(t, body, bundlePath+"/Contents/MacOS/",
			"macOS runs the executable from Contents/MacOS")
		require.Contains(t, body, bundlePath+"/Contents/Resources/",
			"macOS reads the icon from Contents/Resources")
	})

	t.Run("the plist it installs is the one the repository tracks", func(t *testing.T) {
		require.Contains(t, taskBody(t, "package:desktop"), plistPath,
			"package:desktop must install the tracked plist as the bundle's Contents/Info.plist")
		require.FileExists(t, repoPath(plistPath))
	})

	t.Run("the icon is derived from an image the repository tracks", func(t *testing.T) {
		source := captureIn(t, taskBody(t, "package:desktop"),
			regexp.MustCompile(`generate icons [^\n]*-input \./(\S+)`),
			"package:desktop must derive the icon from an image under version control")

		require.FileExists(t, repoPath(source))
	})

	t.Run("the signature is the last thing applied to the bundle", func(t *testing.T) {
		require.Regexp(t, `codesign [^\n]*--sign - `+regexp.QuoteMeta(bundlePath)+`\s*$`,
			taskBody(t, "package:desktop"),
			"a signature over anything but the finished bundle seals neither the plist nor the icon "+
				"that macOS reads around the executable the linker already signed")
	})

	t.Run("the README names the bundle the task builds", func(t *testing.T) {
		require.Contains(t, readRepoFile(t, "README.md"), strings.TrimPrefix(bundlePath, "./"),
			"the README tells a reader to open a bundle package:desktop does not build")
	})

	t.Run("the README gives a downloaded copy its way past Gatekeeper", func(t *testing.T) {
		readme := readRepoFile(t, "README.md")

		require.Contains(t, readme, "xattr -dr com.apple.quarantine",
			"an unsigned bundle that arrived through a browser will not open until the "+
				"quarantine attribute is gone, and this command is the only way out that does "+
				"not depend on a settings pane's wording")
		require.Contains(t, readme, "Open Anyway",
			"the other way out is the one a reader who will not open a terminal needs")
	})
}

const bundlePath = "./bin/emod.app"

func unique(values []string) []string {
	seen := map[string]bool{}
	var distinct []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			distinct = append(distinct, value)
		}
	}

	return distinct
}
