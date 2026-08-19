//go:build unit

package desktop_test

import (
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// The desktop frontend reaches these services through bindings generated from
// the Go source, and calls them by name. Nothing else connects the two:
// renaming an exported method here leaves every Go and JS suite green while the
// assembled desktop app imports a method that no longer exists, and the window
// comes up blank with no error. This reads the names off both sides and
// requires them to agree, the way internal/diagram pins the viewer's palette
// against the exporters'.
func TestBindingNames(t *testing.T) {
	for _, service := range []struct {
		receiver string
		file     string
	}{
		{receiver: "ModelService", file: "service.go"},
		{receiver: "FileService", file: "file_service.go"},
	} {
		t.Run(service.receiver, func(t *testing.T) {
			methods := exportedMethodsOn(t, service.file, service.receiver)
			called := methodsCalledBy(t, "../frontend/desktop/platform.desktop.js", service.receiver)

			require.NotEmpty(t, methods, service.file+" must declare exported "+service.receiver+" methods")
			require.NotEmpty(t, called, "platform.desktop.js must call the generated "+service.receiver+" bindings")
			require.Subset(t, methods, called,
				"platform.desktop.js calls a "+service.receiver+" method Go does not export")
		})
	}
}

func exportedMethodsOn(t *testing.T, path, receiver string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	pattern := regexp.MustCompile(`func \(\w+ \*` + receiver + `\) ([A-Z]\w*)\(`)
	var names []string
	for _, m := range pattern.FindAllStringSubmatch(string(raw), -1) {
		names = append(names, m[1])
	}
	sort.Strings(names)

	return names
}

func methodsCalledBy(t *testing.T, path, receiver string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	pattern := regexp.MustCompile(receiver + `\.([A-Z]\w*)\(`)
	seen := map[string]bool{}
	var names []string
	for _, m := range pattern.FindAllStringSubmatch(string(raw), -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	sort.Strings(names)

	return names
}
