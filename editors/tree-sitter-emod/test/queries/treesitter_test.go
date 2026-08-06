//go:build grammar

package queries

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type point struct {
	Row    int
	Column int
}

type capture struct {
	Name  string
	Start point
	End   point
	Text  string
}

type queryMatch struct {
	Captures []capture
}

type span struct {
	Start point
	End   point
}

func runQuery(t *testing.T, query string) []queryMatch {
	t.Helper()

	matches, err := queryMatches(grammarPath(t, query), samplePath)
	require.NoError(t, err)
	require.NotEmpty(t, matches, "%s captured nothing in %s", query, samplePath)

	return matches
}

func queryMatches(queryPath string, sourcePath string) ([]queryMatch, error) {
	if err := parserGenerated(); err != nil {
		return nil, err
	}

	dir, err := grammarDir()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("tree-sitter", "query", queryPath, sourcePath)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s over %s: %w\n%s", queryPath, sourcePath, err, stderr.String())
	}

	return parseMatches(stdout.String()), nil
}

// Both shapes the CLI prints, the second for a capture that fits on one line:
//
//	capture: fold, start: (18, 0), end: (76, 1)
//	capture: 0 - indent, start: (27, 15), end: (27, 16), text: `{`
var (
	patternLine = regexp.MustCompile(`^\s*pattern: \d+`)
	captureLine = regexp.MustCompile("^\\s*capture: (?:\\d+ - )?([A-Za-z_][A-Za-z0-9_.]*), start: \\((\\d+), (\\d+)\\), end: \\((\\d+), (\\d+)\\)(?:, text: `(.*)`)?$")
)

func parseMatches(output string) []queryMatch {
	var matches []queryMatch
	for _, line := range strings.Split(output, "\n") {
		if patternLine.MatchString(line) {
			matches = append(matches, queryMatch{})
			continue
		}
		fields := captureLine.FindStringSubmatch(line)
		if fields == nil || len(matches) == 0 {
			continue
		}
		last := len(matches) - 1
		matches[last].Captures = append(matches[last].Captures, capture{
			Name:  fields[1],
			Start: point{atoi(fields[2]), atoi(fields[3])},
			End:   point{atoi(fields[4]), atoi(fields[5])},
			Text:  fields[6],
		})
	}

	return matches
}

func atoi(field string) int {
	value, err := strconv.Atoi(field)
	if err != nil {
		panic(err)
	}

	return value
}

func capturesNamed(matches []queryMatch, name string) []capture {
	var found []capture
	for _, match := range matches {
		for _, c := range match.Captures {
			if c.Name == name {
				found = append(found, c)
			}
		}
	}

	return found
}

func soleCaptureAt(t *testing.T, matches []queryMatch, name string, start point) capture {
	t.Helper()

	return soleMatchAt(t, matches, name, start).capture(t, name)
}

func soleMatchAt(t *testing.T, matches []queryMatch, name string, start point) queryMatch {
	t.Helper()

	var found []queryMatch
	for _, match := range matches {
		for _, c := range match.Captures {
			if c.Name == name && c.Start == start {
				found = append(found, match)
				break
			}
		}
	}
	require.Len(t, found, 1, "expected one match capturing @%s at %+v", name, start)

	return found[0]
}

func (m queryMatch) capture(t *testing.T, name string) capture {
	t.Helper()

	for _, c := range m.Captures {
		if c.Name == name {
			return c
		}
	}
	require.FailNowf(t, "capture not found", "match carries no @%s: %+v", name, m.Captures)

	return capture{}
}

func spans(captures []capture) []span {
	var found []span
	for _, c := range captures {
		found = append(found, span{Start: c.Start, End: c.End})
	}

	return found
}

func columnAfter(p point) point {
	return point{p.Row, p.Column + 1}
}

func precedes(earlier point, later point) bool {
	if earlier.Row != later.Row {
		return earlier.Row < later.Row
	}

	return earlier.Column < later.Column
}

func readGrammarFile(t *testing.T, relative string) string {
	t.Helper()

	content, err := os.ReadFile(grammarPath(t, relative))
	require.NoError(t, err)

	return string(content)
}

func grammarPath(t *testing.T, relative string) string {
	t.Helper()

	dir, err := grammarDir()
	require.NoError(t, err)

	return filepath.Join(dir, relative)
}

func grammarDir() (string, error) {
	return filepath.Abs(filepath.Join("..", ".."))
}

// A stale src/ compiles the queries against whichever grammar generated it, so
// every capture the CLI reports would be read from the wrong parser.
var parserGenerated = sync.OnceValue(func() error {
	dir, err := grammarDir()
	if err != nil {
		return err
	}

	cmd := exec.Command("tree-sitter", "generate")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tree-sitter generate in %s: %w\n%s", dir, err, output)
	}

	return nil
})
