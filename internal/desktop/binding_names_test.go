//go:build unit

package desktop_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
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
	// Derived rather than listed: a service added to main.go and called from the
	// frontend would otherwise get no method guard, and nothing would say so.
	for _, receiver := range servicesRegisteredBy(t, "../../cmd/emod-desktop/main.go") {
		t.Run("every "+receiver+" method the frontend calls is exported by Go", func(t *testing.T) {
			methods := exportedMethodsOnReceiver(t, receiver)
			called := methodsCalledBy(t, "../frontend/desktop/platform.desktop.js", receiver)

			require.NotEmpty(t, methods, "no file in this package declares exported "+receiver+" methods")
			require.NotEmpty(t, called, "platform.desktop.js must call the generated "+receiver+" bindings")
			require.Subset(t, methods, called,
				"platform.desktop.js calls a "+receiver+" method Go does not export")
		})
	}
}

// Every non-test file in the package, unioned: the convention is that a type and
// its methods live in one file, and scanning them all means a method that broke
// that convention still counts here rather than silently losing its guard.
func exportedMethodsOnReceiver(t *testing.T, receiver string) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	var methods []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		methods = append(methods, exportedMethodsOn(t, entry.Name(), receiver)...)
	}
	sort.Strings(methods)

	return methods
}

// Importing a service in the frontend and registering it with the app are two
// separate edits, and only the second is what makes `wails3 generate bindings`
// emit it. Dropping a registration leaves every Go and JS suite green while the
// generated index stops exporting that service and the window fails on an
// import it cannot resolve — the same silent-blank-window failure the method
// guard above exists for.
func TestServiceRegistrations(t *testing.T) {
	t.Run("every service the frontend imports is registered with the app", func(t *testing.T) {
		registered := servicesRegisteredBy(t, "../../cmd/emod-desktop/main.go")
		imported := servicesImportedBy(t, "../frontend/desktop/platform.desktop.js")

		require.NotEmpty(t, registered, "main.go must register at least one service")
		require.NotEmpty(t, imported, "platform.desktop.js must import at least one service")
		require.Subset(t, registered, imported,
			"platform.desktop.js imports a service main.go does not register with the app")
	})
}

func servicesRegisteredBy(t *testing.T, path string) []string {
	t.Helper()

	return uniqueMatches(t, path, regexp.MustCompile(`application\.NewService\(&desktop\.(\w+)\{`))
}

func servicesImportedBy(t *testing.T, path string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	// Every block, not just the first: splitting the import in two would
	// otherwise narrow this guard to whichever half came first.
	blocks := regexp.MustCompile(`import \{([^}]*)\} from '\.\./bindings/[^']*'`).FindAllStringSubmatch(string(raw), -1)
	require.NotEmpty(t, blocks, "platform.desktop.js must import the generated bindings by name")

	var names []string
	for _, block := range blocks {
		for _, name := range strings.Split(block[1], ",") {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				names = append(names, trimmed)
			}
		}
	}
	sort.Strings(names)

	return names
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

	return uniqueMatches(t, path, regexp.MustCompile(receiver+`\.([A-Z]\w*)\(`))
}

// The frontend reads the answer's fields by name off a decoded object, and Go
// writes them from struct tags. Renaming a key on one side is invisible to both
// compilers and to every suite: the viewer simply reads undefined and the window
// renders nothing, which is the same silent failure the guards above exist for.
func TestOpenedFileWireKeys(t *testing.T) {
	t.Run("every field the viewer reads off an opened file is a key the service writes", func(t *testing.T) {
		written := jsonKeysWrittenFor(t, "file_service.go", "openedFile")
		read := fieldsReadBy(t, "../frontend/static/viewer.js", "opened")

		require.NotEmpty(t, written, "file_service.go must tag the fields it puts on the wire")
		require.NotEmpty(t, read, "viewer.js must read the opened file's fields")
		require.Subset(t, written, read,
			"viewer.js reads a field of the opened file that file_service.go does not write")
	})
}

func jsonKeysWrittenFor(t *testing.T, path, structName string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	body := regexp.MustCompile(`type ` + structName + ` struct \{([^}]*)\}`).FindStringSubmatch(string(raw))
	require.Len(t, body, 2, path+" must declare "+structName)

	return uniqueMatchesIn(body[1], regexp.MustCompile("`json:\"(\\w+)\""))
}

func fieldsReadBy(t *testing.T, path, receiver string) []string {
	t.Helper()

	// error is the failure envelope rather than a field of a file that opened.
	read := uniqueMatches(t, path, regexp.MustCompile(receiver+`\.(\w+)`))
	kept := read[:0]
	for _, name := range read {
		if name != "error" {
			kept = append(kept, name)
		}
	}

	return kept
}
