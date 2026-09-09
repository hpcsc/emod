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

	t.Run("every icon is derived from an image the repository tracks", func(t *testing.T) {
		for source := range iconDerivations(t) {
			require.FileExists(t, repoPath(source))
		}
	})

	t.Run("each icon is derived from its own image", func(t *testing.T) {
		derivations := iconDerivations(t)
		written := make([]string, 0, len(derivations))
		for _, icon := range derivations {
			written = append(written, icon)
		}

		// Keyed by input, so two commands reading one image collapse to a single
		// entry and the count falls below the number of commands. Comparing the
		// map against a slice built from the map itself cannot fail.
		require.Len(t, derivations, strings.Count(taskBody(t, "package:desktop"), "generate icons"),
			"two icons derived from one image are one picture under two names, "+
				"and a document drawn with the application's icon says it is a program")
		require.Equal(t, unique(written), written,
			"two images written to one icon file leave whichever ran last as both icons")
	})

	// Reading each command's input and its output separately says both images are
	// used and both icons are written while leaving unsaid which produces which,
	// so the two -macfilename values can be exchanged with every assertion still
	// passing while Finder and the Dock draw each other's picture.
	t.Run("each image produces the icon named after it", func(t *testing.T) {
		require.Equal(t,
			map[string]string{"build/appicon.png": "icons", "build/docicon.png": "docicon"},
			iconDerivations(t))
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

	// The story's headline behaviour, and the paragraph a later edit is most
	// likely to shorten away. Its tokens are the three button labels, read out of
	// the frontend rather than written here, so the README cannot go on promising
	// a question whose buttons have been renamed.
	t.Run("the README says what a double-click does to a model already open", func(t *testing.T) {
		// Scoped to the paragraph rather than the file: every label it names is
		// also in the paragraph about dropping a file, so asserting them over the
		// whole README passes with this paragraph deleted.
		paragraph := captureIn(t, readRepoFile(t, "README.md"),
			regexp.MustCompile(`(?s)(\*\*Double-clicking [^\n]*\n.*?)\n\n`),
			"the README must tell a reader that double-clicking a model opens it in emod")

		for _, label := range unsavedEditLabels(t) {
			require.Contains(t, paragraph, label,
				"a double-click onto unsaved edits asks this question, and a reader told nothing "+
					"about it will not expect to be interrupted by it")
		}
	})

	t.Run("the README says what makes macOS open a model in emod", func(t *testing.T) {
		readme := readRepoFile(t, "README.md")

		require.Contains(t, readme, exportedIdentifier(t),
			"a reader whose double-click does nothing needs the type name to check against, and it is "+
				"the one the bundle exports rather than a name this file could invent")
		require.Contains(t, readme, "mdls -name kMDItemContentType",
			"the association is invisible until macOS has seen the bundle, so the reader needs the one "+
				"command that says whether it has")
	})

	t.Run("the README says how to reach emod for a JSON file, and what it costs", func(t *testing.T) {
		readme := readRepoFile(t, "README.md")

		require.Contains(t, readme, "Open With",
			"emod is offered for .json rather than given it, so this menu is the only way to reach it")
		require.Contains(t, readme, "Get Info",
			"the Finder route that changes which application opens a file starts here")
		require.Contains(t, readme, "Change All",
			"and this is the button that changes it for every .json on the machine, which a reader "+
				"must choose deliberately rather than meet by accident")
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

// The buttons the unsaved-edits question offers, read off the frontend that
// raises it.
func unsavedEditLabels(t *testing.T) []string {
	t.Helper()

	body := captureIn(t, readRepoFile(t, "internal/frontend/desktop/platform.desktop.js"),
		regexp.MustCompile(`(?s)const UNSAVED_EDIT_OUTCOMES = \{(.*?)\}`),
		"platform.desktop.js must map the question's button labels to outcomes")

	labels := uniqueMatchesIn(body, regexp.MustCompile(`(\w+):`))
	require.Len(t, labels, 3, "the question offers three buttons")

	return labels
}

// Each `generate icons` command in package:desktop, as the tracked image it
// reads against the icon name it writes. Paired per command rather than
// collected as two lists, because two lists say both images are used and both
// icons are written while leaving which produces which unsaid.
func iconDerivations(t *testing.T) map[string]string {
	t.Helper()

	pattern := regexp.MustCompile(`generate icons [^\n]*-input \./(\S+)[^\n]*-macfilename ` +
		regexp.QuoteMeta(bundlePath) + `/Contents/Resources/(\S+)\.icns`)
	commands := pattern.FindAllStringSubmatch(taskBody(t, "package:desktop"), -1)
	require.NotEmpty(t, commands,
		"package:desktop must derive each icon from a tracked image into Contents/Resources")

	derivations := map[string]string{}
	for _, command := range commands {
		derivations[command[1]] = command[2]
	}

	return derivations
}

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
