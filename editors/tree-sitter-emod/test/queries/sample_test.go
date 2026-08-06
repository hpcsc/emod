//go:build grammar

package queries

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const samplePath = "test/highlight/constructs.emod"

type blockExtent struct {
	Start      point
	End        point
	OpenBrace  point
	CloseBrace point
}

type braceAt struct {
	Column  int
	Opening bool
}

// The extent is derived from the sample source rather than from the CLI's own
// output, which would assert the queries against themselves.
func locateBlock(t *testing.T, block sampleBlock) blockExtent {
	t.Helper()

	lines := sampleLines(t)
	extent, closed := extentFrom(lines, headerRow(t, lines, block.header))
	require.True(t, closed, "%q is never closed in %s", block.header, samplePath)

	return extent
}

func headerRow(t *testing.T, lines []string, header string) int {
	t.Helper()

	found := -1
	for row, line := range lines {
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), header) {
			require.Equal(t, -1, found, "%q heads more than one line of %s", header, samplePath)
			found = row
		}
	}
	require.NotEqual(t, -1, found, "%q heads no line of %s", header, samplePath)

	return found
}

func extentFrom(lines []string, header int) (blockExtent, bool) {
	extent := blockExtent{Start: point{header, indentOf(lines[header])}}
	depth := 0
	for row := header; row < len(lines); row++ {
		for _, brace := range bracesIn(lines[row]) {
			if brace.Opening {
				if depth == 0 {
					extent.OpenBrace = point{row, brace.Column}
				}
				depth++

				continue
			}

			depth--
			if depth == 0 {
				extent.CloseBrace = point{row, brace.Column}
				extent.End = columnAfter(extent.CloseBrace)

				return extent, true
			}
		}
	}

	return extent, false
}

func bracesIn(line string) []braceAt {
	var found []braceAt
	for column := 0; column < len(line); column++ {
		switch line[column] {
		case '"':
			column = closingQuote(line, column)
		case '#':
			return found
		case '{':
			found = append(found, braceAt{Column: column, Opening: true})
		case '}':
			found = append(found, braceAt{Column: column})
		}
	}

	return found
}

func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

func closingQuote(line string, opening int) int {
	if end := strings.IndexByte(line[opening+1:], '"'); end >= 0 {
		return opening + 1 + end
	}

	return len(line)
}

func sampleCharacterBefore(t *testing.T, p point) string {
	t.Helper()

	lines := sampleLines(t)
	require.Less(t, p.Row, len(lines))
	require.Positive(t, p.Column)

	return string(lines[p.Row][p.Column-1])
}

func sampleLines(t *testing.T) []string {
	t.Helper()

	return strings.Split(readGrammarFile(t, samplePath), "\n")
}
