//go:build unit

package desktop_test

import (
	"encoding/xml"
	"os"
	"regexp"
	"slices"
	"strings"
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
		require.Contains(t, iconsWrittenByPackaging(t), plistString(t, "CFBundleIconFile"),
			"macOS shows the icon Info.plist names, and package:desktop writes a different one")
	})

	t.Run("the packaging task writes the document icon the .emod type names", func(t *testing.T) {
		document := dictString(t, documentTypeFor(t, exportedIdentifier(t)), "CFBundleTypeIconFile", documentTypeIn)

		require.Contains(t, iconsWrittenByPackaging(t), document,
			"Finder draws the icon the document type names, and package:desktop writes a different one")
		require.NotEqual(t, plistString(t, "CFBundleIconFile"), document,
			"a folder of models drawn with the application's own icon says every one of them is a program")
	})

	t.Run("the document icon is named the same wherever macOS looks for it", func(t *testing.T) {
		require.Equal(t,
			dictString(t, exportedType(t), "UTTypeIconFile", exportedTypeIn),
			dictString(t, documentTypeFor(t, exportedIdentifier(t)), "CFBundleTypeIconFile", documentTypeIn),
			"the exported type and the document type must not point Finder at two different icons")
	})

	t.Run("the bundle keeps the identifier macOS has already filed it under", func(t *testing.T) {
		require.Equal(t, "au.pnguyen.emod", plistString(t, "CFBundleIdentifier"),
			"every machine that has run the app has this identifier recorded in LaunchServices, "+
				"and the file associations of a later story bind to it; changing it strands both")
	})

	// What makes a .emod file open in emod is a chain: the bundle identifier
	// names the exported type, the exported type tags the extension, and a
	// document type entry names that same identifier back. Each hop is compared
	// to the value it must agree with rather than to a literal, because two
	// assertions that each tie one end to a third value leave the middle free.
	t.Run("the exported type is filed under the bundle's own identifier", func(t *testing.T) {
		bundleIdentifier := plistString(t, "CFBundleIdentifier")
		exported := dictString(t, exportedType(t), "UTTypeIdentifier", exportedTypeIn)

		require.True(t, strings.HasPrefix(exported, bundleIdentifier+"."),
			exported+" is not under "+bundleIdentifier+"; a type identifier outside the bundle's own "+
				"namespace is one any other vendor can declare too, and the last one registered wins")
	})

	t.Run("the exported type is what tags a .emod file", func(t *testing.T) {
		exported := exportedType(t)

		require.Contains(t, dictStrings(t, exported, "UTTypeConformsTo", exportedTypeIn), "public.plain-text",
			"a model is text, and a type that conforms to nothing inherits none of the behaviour "+
				"macOS gives text — Quick Look, and the editors offered beneath emod")
		require.Equal(t, []string{"emod"},
			dictStrings(t, dictEntries(t, exported, "UTTypeTagSpecification", exportedTypeIn),
				"public.filename-extension", exportedTypeIn),
			"the extension tag is the whole of what attaches the declared type to a file on disk: "+
				"without emod macOS keeps reading .emod as an anonymous dyn. type, and a second "+
				"extension here would claim that one outright, whatever rank the document types give it")
	})

	t.Run("the app claims the type it exports", func(t *testing.T) {
		claimed := documentTypeFor(t, exportedIdentifier(t))

		require.Equal(t, "Owner", dictString(t, claimed, "LSHandlerRank", documentTypeIn),
			"the app that exports a type is the one that owns it; anything lower leaves .emod files "+
				"opening in whatever else happens to claim plain text")
		require.Equal(t, "Editor", dictString(t, claimed, "CFBundleTypeRole", documentTypeIn),
			"the role is what says the app opens this type at all: None leaves the rank above "+
				"declaring an ownership macOS never acts on, and double-clicking a model stops "+
				"reaching emod with nothing here to say so")
	})

	// The handler rank and the exported type's own tag are two routes to owning a
	// type, and an imported type declaration is a third that neither of them
	// reads. A plist that grew one later would claim .json outright with both of
	// the assertions below still passing, so this reads every extension tag the
	// file declares wherever it sits.
	t.Run("the bundle tags no file extension but its own", func(t *testing.T) {
		require.Equal(t, []string{"emod"}, filenameExtensionsIn(bundlePlist(t)),
			"an extension tagged anywhere in this file is one emod declares a type for, "+
				"whatever rank the document types give it")
	})

	t.Run("the app offers itself for JSON without taking it", func(t *testing.T) {
		offered := documentTypeFor(t, "public.json")

		require.Equal(t, "Alternate", dictString(t, offered, "LSHandlerRank", documentTypeIn),
			"Owner or Default would make emod the default application for every .json file on the "+
				"machine, which the story forbids; None would keep emod out of Open With altogether")
		require.Equal(t, "Editor", dictString(t, offered, "CFBundleTypeRole", documentTypeIn),
			"a rank of Alternate offers an application that the role must first say can open the "+
				"type; None offers nothing, and emod never reaches Open With for a .json file")
	})
}

const (
	plistPath    = "build/darwin/Info.plist"
	taskfilePath = "Taskfile.yml"

	exportedTypeIn = plistPath + "'s exported type declaration"
	documentTypeIn = plistPath + "'s document type entry"
)

// The type the bundle declares as its own. One, because a second would be a
// second thing every document type entry has to be kept in step with.
func exportedType(t *testing.T) map[string]any {
	t.Helper()

	declared := dictArray(t, bundlePlist(t), "UTExportedTypeDeclarations", plistPath)
	require.Len(t, declared, 1, plistPath+" must export exactly one type")

	return asDict(t, declared[0], exportedTypeIn)
}

// Every icon package:desktop writes into the bundle. All of them rather than
// the first: the bundle carries the application's icon and the document's, and
// a reader that stops at one leaves whichever comes second unguarded.
func iconsWrittenByPackaging(t *testing.T) []string {
	t.Helper()

	return capturesIn(t, taskBody(t, "package:desktop"),
		regexp.MustCompile(`Contents/Resources/(\S+)\.icns`),
		"package:desktop must write the bundle's icons into Contents/Resources")
}

func exportedIdentifier(t *testing.T) string {
	t.Helper()

	return dictString(t, exportedType(t), "UTTypeIdentifier", exportedTypeIn)
}

// Every filename extension the plist tags, at any depth. An exported type
// declaration, an imported one and a document type entry each carry their own
// tag specification, so a reader that opens one of them answers for one of them.
func filenameExtensionsIn(value any) []string {
	var found []string
	switch typed := value.(type) {
	case map[string]any:
		for key, entry := range typed {
			if key == "public.filename-extension" {
				found = append(found, textsIn(entry)...)

				continue
			}
			found = append(found, filenameExtensionsIn(entry)...)
		}
	case []any:
		for _, entry := range typed {
			found = append(found, filenameExtensionsIn(entry)...)
		}
	}
	slices.Sort(found)

	return found
}

func textsIn(value any) []string {
	if text, isString := value.(string); isString {
		return []string{text}
	}

	entries, isArray := value.([]any)
	if !isArray {
		return nil
	}

	var texts []string
	for _, entry := range entries {
		if text, isString := entry.(string); isString {
			texts = append(texts, text)
		}
	}

	return texts
}

// The entry that tells macOS this app opens contentType.
func documentTypeFor(t *testing.T, contentType string) map[string]any {
	t.Helper()

	for _, entry := range dictArray(t, bundlePlist(t), "CFBundleDocumentTypes", plistPath) {
		declared := asDict(t, entry, documentTypeIn)
		if slices.Contains(dictStrings(t, declared, "LSItemContentTypes", documentTypeIn), contentType) {
			return declared
		}
	}
	require.FailNow(t, plistPath+" declares no document type naming "+contentType)

	return nil
}

// The value of a top-level plist key.
func plistString(t *testing.T, key string) string {
	t.Helper()

	return dictString(t, bundlePlist(t), key, plistPath)
}

func dictString(t *testing.T, entries map[string]any, key string, where string) string {
	t.Helper()

	value, declared := entries[key]
	require.True(t, declared, where+" declares no value for "+key)

	text, isString := value.(string)
	require.True(t, isString, where+" declares "+key+" as something other than a string")
	require.NotEmpty(t, text, where+" leaves "+key+" empty")

	return text
}

func dictArray(t *testing.T, entries map[string]any, key string, where string) []any {
	t.Helper()

	value, declared := entries[key]
	require.True(t, declared, where+" declares no "+key)

	values, isArray := value.([]any)
	require.True(t, isArray, where+" declares "+key+" as something other than an array")
	require.NotEmpty(t, values, where+" leaves "+key+" empty")

	return values
}

func dictStrings(t *testing.T, entries map[string]any, key string, where string) []string {
	t.Helper()

	var texts []string
	for _, value := range dictArray(t, entries, key, where) {
		text, isString := value.(string)
		require.True(t, isString, where+" holds a non-string in "+key)
		texts = append(texts, text)
	}

	return texts
}

func dictEntries(t *testing.T, entries map[string]any, key string, where string) map[string]any {
	t.Helper()

	value, declared := entries[key]
	require.True(t, declared, where+" declares no "+key)

	return asDict(t, value, where+"'s "+key)
}

func asDict(t *testing.T, value any, where string) map[string]any {
	t.Helper()

	entries, isDict := value.(map[string]any)
	require.True(t, isDict, where+" is not a dictionary")

	return entries
}

// The bundle's plist as nested maps and slices. A property list puts a value in
// the element after its key rather than inside it, so a search for `<key>X</key>`
// followed by a value matches a key nested three dictionaries down as readily as
// a top-level one — which is why this decodes the document instead.
func bundlePlist(t *testing.T) map[string]any {
	t.Helper()

	decoder := xml.NewDecoder(strings.NewReader(readRepoFile(t, plistPath)))
	for {
		token, err := decoder.Token()
		require.NoError(t, err, plistPath+" holds no dictionary")

		if start, isElement := token.(xml.StartElement); isElement && start.Name.Local == "dict" {
			return decodeDict(t, decoder)
		}
	}
}

func decodeDict(t *testing.T, decoder *xml.Decoder) map[string]any {
	t.Helper()

	entries := map[string]any{}
	key := ""
	awaitingValue := false
	for {
		token, err := decoder.Token()
		require.NoError(t, err, "a plist dictionary ends before its closing tag")

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local == "key" {
				key = decodeText(t, decoder, element)
				awaitingValue = true

				continue
			}
			require.True(t, awaitingValue, "a plist dictionary holds a value with no key before it")
			entries[key] = decodeValue(t, decoder, element)
			awaitingValue = false
		case xml.EndElement:
			if element.Name.Local == "dict" {
				require.False(t, awaitingValue, "a plist dictionary ends on the key "+key+" with no value after it")

				return entries
			}
		}
	}
}

func decodeArray(t *testing.T, decoder *xml.Decoder) []any {
	t.Helper()

	values := []any{}
	for {
		token, err := decoder.Token()
		require.NoError(t, err, "a plist array ends before its closing tag")

		switch element := token.(type) {
		case xml.StartElement:
			values = append(values, decodeValue(t, decoder, element))
		case xml.EndElement:
			if element.Name.Local == "array" {
				return values
			}
		}
	}
}

func decodeValue(t *testing.T, decoder *xml.Decoder, element xml.StartElement) any {
	t.Helper()

	switch element.Name.Local {
	case "dict":
		return decodeDict(t, decoder)
	case "array":
		return decodeArray(t, decoder)
	case "true", "false":
		require.NoError(t, decoder.Skip())

		return element.Name.Local == "true"
	default:
		return decodeText(t, decoder, element)
	}
}

func decodeText(t *testing.T, decoder *xml.Decoder, element xml.StartElement) string {
	t.Helper()

	var text string
	require.NoError(t, decoder.DecodeElement(&text, &element))

	return text
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

func capturesIn(t *testing.T, text string, pattern *regexp.Regexp, missing string) []string {
	t.Helper()

	matches := pattern.FindAllStringSubmatch(text, -1)
	require.NotEmpty(t, matches, missing)

	var captured []string
	for _, match := range matches {
		require.NotEmpty(t, match[1], missing)
		captured = append(captured, match[1])
	}

	return captured
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
