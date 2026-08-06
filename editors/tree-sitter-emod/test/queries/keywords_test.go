//go:build grammar

package queries

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hpcsc/emod/internal/lexer"
	"github.com/stretchr/testify/require"
)

// Each editor surface restates the language by hand, and none is reachable from
// `go test ./...`, so nothing but this file notices when a keyword added to
// internal/lexer never reaches them.
type editorSurface struct {
	name       string
	file       string
	spellings  func(t *testing.T, path string) map[string]struct{}
	addKeyword string
}

var editorSurfaces = []editorSurface{
	{
		name:       "the tree-sitter grammar",
		file:       filepath.Join("tree-sitter-emod", "grammar.js"),
		spellings:  singleQuotedTokens,
		addKeyword: "add it as a token literal in a rule",
	},
	{
		name:       "the highlight query",
		file:       filepath.Join("tree-sitter-emod", "queries", "highlights.scm"),
		spellings:  doubleQuotedTokens,
		addKeyword: "add it to the @keyword list",
	},
	{
		name:       "the TextMate grammar",
		file:       filepath.Join("vscode", "syntaxes", "emod.tmLanguage.json"),
		spellings:  patternWords,
		addKeyword: "add it to a match/begin pattern, usually the #keywords alternation",
	},
}

func TestEditorKeywordCoverage(t *testing.T) {
	t.Run("coverage", func(t *testing.T) {
		require.NotEmpty(t, lexer.Keywords(), "the lexer reports no keywords at all")

		for _, surface := range editorSurfaces {
			for _, keyword := range lexer.Keywords() {
				t.Run(surface.name+" spells "+keyword, func(t *testing.T) {
					require.Contains(
						t,
						surface.spellings(t, editorFile(t, surface.file)),
						keyword,
						"%s never spells the lexer keyword %q: %s",
						surface.file, keyword, surface.addKeyword,
					)
				})
			}
		}
	})

	t.Run("extraction", func(t *testing.T) {
		for _, surface := range editorSurfaces {
			t.Run(surface.name+" yields a spelling the file does not carry", func(t *testing.T) {
				require.NotContains(t, surface.spellings(t, editorFile(t, surface.file)), "decides_upon")
			})

			t.Run(surface.name+" yields distinct spellings rather than one run of text", func(t *testing.T) {
				spellings := surface.spellings(t, editorFile(t, surface.file))

				require.Greater(t, len(spellings), len(lexer.Keywords()))
			})
		}
	})
}

// A quote inside a comment would otherwise pair with the next one further down
// the file and swallow every literal between them, so a spelling never runs past
// the end of its line.
var (
	singleQuotedToken = regexp.MustCompile(`'([^'\n]*)'`)
	doubleQuotedToken = regexp.MustCompile(`"([^"\n]*)"`)
)

func singleQuotedTokens(t *testing.T, path string) map[string]struct{} {
	t.Helper()

	return tokensMatching(singleQuotedToken, readEditorFile(t, path))
}

func doubleQuotedTokens(t *testing.T, path string) map[string]struct{} {
	t.Helper()

	return tokensMatching(doubleQuotedToken, readEditorFile(t, path))
}

func tokensMatching(pattern *regexp.Regexp, content string) map[string]struct{} {
	found := map[string]struct{}{}
	for _, match := range pattern.FindAllStringSubmatch(content, -1) {
		found[match[1]] = struct{}{}
	}

	return found
}

var (
	regexEscape = regexp.MustCompile(`\\.`)
	patternWord = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

func patternWords(t *testing.T, path string) map[string]struct{} {
	t.Helper()

	var grammar any
	require.NoError(t, json.Unmarshal([]byte(readEditorFile(t, path)), &grammar))

	found := map[string]struct{}{}
	for _, pattern := range matchPatterns(grammar) {
		// An escape abuts the word it bounds, so \bevery reads as one word until
		// the escape is replaced: dropping this step spells the keyword "bevery".
		for _, word := range patternWord.FindAllString(regexEscape.ReplaceAllString(pattern, " "), -1) {
			found[word] = struct{}{}
		}
	}

	return found
}

// "name" and "comment" hold prose, not syntax, so only the keys TextMate reads
// as regular expressions contribute spellings.
func matchPatterns(node any) []string {
	var found []string
	switch typed := node.(type) {
	case map[string]any:
		for key, value := range typed {
			pattern, isString := value.(string)
			if isString && (key == "match" || key == "begin" || key == "end") {
				found = append(found, pattern)

				continue
			}
			found = append(found, matchPatterns(value)...)
		}
	case []any:
		for _, value := range typed {
			found = append(found, matchPatterns(value)...)
		}
	}

	return found
}

func editorFile(t *testing.T, relative string) string {
	t.Helper()

	dir, err := grammarDir()
	require.NoError(t, err)

	return filepath.Join(dir, "..", relative)
}

func readEditorFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)

	return string(content)
}
