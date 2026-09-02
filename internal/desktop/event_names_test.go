//go:build unit

package desktop_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The shell's menus are the only things that ask the frontend to open a file,
// and they ask by emitting an event the frontend subscribes to by name. Nothing
// else connects the two: renaming it on either side alone leaves every suite
// green while Open becomes a menu item that silently does nothing. The whole set
// is compared rather than one direction of it, because an event nobody listens
// for and a listener nothing emits are the same defect wearing different
// clothes.
func TestShellEventNames(t *testing.T) {
	t.Run("the shell and the frontend name the same events", func(t *testing.T) {
		emitted := eventsEmittedBy(t, "../../cmd/emod-desktop")
		subscribed := eventsSubscribedBy(t, "../frontend/desktop/platform.desktop.js")

		require.NotEmpty(t, emitted, "cmd/emod-desktop must emit at least one event to the frontend")
		require.NotEmpty(t, subscribed, "platform.desktop.js must subscribe to at least one shell event")
		require.Equal(t, emitted, subscribed,
			"the shell emits an event the frontend does not listen for, or listens for one it never emits")
	})
}

// Every non-test file in the shell, unioned: a menu that lives in its own file
// emits from there, and a guard reading main.go alone would call its event one
// nothing emits.
func eventsEmittedBy(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var source strings.Builder
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err)
		source.Write(raw)
		source.WriteByte('\n')
	}

	return uniqueMatchesIn(source.String(), regexp.MustCompile(`EmitEvent\("([^"]+)"`))
}

func eventsSubscribedBy(t *testing.T, path string) []string {
	t.Helper()

	return uniqueMatches(t, path, regexp.MustCompile(`Events\.On\('([^']+)'`))
}

func uniqueMatches(t *testing.T, path string, pattern *regexp.Regexp) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	return uniqueMatchesIn(string(raw), pattern)
}

func uniqueMatchesIn(text string, pattern *regexp.Regexp) []string {
	seen := map[string]bool{}
	var names []string
	for _, m := range pattern.FindAllStringSubmatch(text, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	sort.Strings(names)

	return names
}
