//go:build unit

package lsp_test

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/hpcsc/emod/internal/lsp"
	"github.com/stretchr/testify/require"
)

// The expected categories follow nvim-treesitter's queries/hcl/highlights.scm at
// revision 692b051b09935653befdb8f7ba8afdb640adf17b.
func TestGetTokens(t *testing.T) {
	t.Run("lexical elements", func(t *testing.T) {
		const doc = `emod = 1

# a hash comment
// a slash comment
/* a block comment */
operators = !a || b && c == d != e < f <= g > h >= i + j - k * l / m % n
brackets  = [call(x), { key = 2.5, other = 3, (paren) = 4 }]
access    = var.name.leaf
indexed   = matrix[0].cell
chained   = list[key].name[pick].end
called    = call().field
splats    = [list[*].id, list.*.id, list[*][0]]
specials  = [spread(xs...), p ? q : r, { for key, val in map : key => val if val }]
literals  = [true, false, null, 1e3]
template  = "say ${word} %{~ if loud }LOUD%{ else }soft%{ endif ~} %{ for n in names }${n}%{ endfor }"
flow = <<-FLOW
  command -> event: Place -> Placed
  FLOW
plain = <<END
text
END
`

		cases := []struct {
			container string
			text      string
			category  string
		}{
			{"!a", "!", "operator"},
			{"a || b", "||", "operator"},
			{"b && c", "&&", "operator"},
			{"c == d", "==", "operator"},
			{"d != e", "!=", "operator"},
			{"e < f", "<", "operator"},
			{"f <= g", "<=", "operator"},
			{"g > h", ">", "operator"},
			{"h >= i", ">=", "operator"},
			{"i + j", "+", "operator"},
			{"j - k", "-", "operator"},
			{"k * l", "*", "operator"},
			{"l / m", "/", "operator"},
			{"m % n", "%", "operator"},
			{"brackets  = [", "[", "punctuation.bracket"},
			{"call(", "(", "punctuation.bracket"},
			{"x)", ")", "punctuation.bracket"},
			{"{ key", "{", "punctuation.bracket"},
			{"4 }", "}", "punctuation.bracket"},
			{"4 }]", "]", "punctuation.bracket"},
			{"*][0]", "[", "punctuation.bracket"},
			{"var.name", ".", "punctuation.delimiter"},
			{"call(x),", ",", "punctuation.delimiter"},
			{"list[*]", "[*]", "punctuation.delimiter"},
			{"list.*", ".*", "punctuation.delimiter"},
			{"xs...", "...", "punctuation.special"},
			{"p ? q", "?", "punctuation.special"},
			{"key => val", "=>", "punctuation.special"},
			{"${word}", "${", "punctuation.special"},
			{"word}", "}", "punctuation.special"},
			{"%{~ if", "%{~", "punctuation.special"},
			{"endif ~}", "~}", "punctuation.special"},
			{"{ for key", "for", "keyword.repeat"},
			{"val in map", "in", "keyword.repeat"},
			{"%{ for n", "for", "keyword.repeat"},
			{"n in names", "in", "keyword.repeat"},
			{"%{ endfor", "endfor", "keyword.repeat"},
			{"val if val", "if", "keyword.conditional"},
			{"%{~ if loud", "if", "keyword.conditional"},
			{"%{ else", "else", "keyword.conditional"},
			{"%{ endif", "endif", "keyword.conditional"},
			{`"say`, `"`, "string"},
			{`"say `, "say ", "string"},
			{"}LOUD%", "LOUD", "string"},
			{`endfor }"`, `"`, "string"},
			{"command -> event", "command -> event: Place -> Placed", "string"},
			{"<<-FLOW", "<<-", "punctuation.delimiter"},
			{"<<-FLOW", "FLOW", "punctuation.delimiter"},
			{"  FLOW\nplain", "FLOW", "punctuation.delimiter"},
			{"<<END", "<<", "punctuation.delimiter"},
			{"<<END", "END", "punctuation.delimiter"},
			{"text\nEND", "END", "punctuation.delimiter"},
			{"emod = 1", "1", "number"},
			{"2.5", "2.5", "number"},
			{"1e3", "1e3", "number"},
			{"[true", "true", "boolean"},
			{"false", "false", "boolean"},
			{"null", "null", "constant"},
			{"# a hash", "# a hash comment", "comment"},
			{"// a slash", "// a slash comment", "comment"},
			{"/* a block", "/* a block comment */", "comment"},
			{"!a", "a", "variable"},
			{"p ? q : r", "r", "variable"},
			{"{ for key, val", "val", "variable"},
			{"call(x)", "call", "function"},
			{"spread(", "spread", "function"},
			{"operators =", "operators", "variable.member"},
			{"emod = 1", "emod", "variable.member"},
			{"{ key = 2.5", "key", "variable.member"},
			{"other = 3", "other", "variable.member"},
			{"(paren)", "paren", "variable"},
			{"var.name", "var", "variable.builtin"},
			{"var.name", "name", "variable.member"},
			{"name.leaf", "leaf", "variable.member"},
			{"matrix[0]", "matrix", "variable.builtin"},
			{"[0].cell", "cell", "variable.member"},
			{"list[key]", "list", "variable.builtin"},
			{"[key]", "key", "variable"},
			{"].name[pick]", "name", "variable.member"},
			{"[pick].end", "end", "variable.member"},
			{"call().field", "field", "variable"},
		}
		for _, c := range cases {
			t.Run(c.text+" in "+c.container+" is "+c.category, func(t *testing.T) {
				require.Equal(t, c.category, categoryOf(t, doc, c.container, c.text))
			})
		}

		t.Run("an attribute's equals sign has no token while its name and value do", func(t *testing.T) {
			require.Empty(t, tokensOver(t, doc, "emod = 1", "="))
			require.Equal(t, "variable.member", categoryOf(t, doc, "emod = 1", "emod"))
			require.Equal(t, "number", categoryOf(t, doc, "emod = 1", "1"))
		})

		t.Run("a conditional's colon has no token while its question mark does", func(t *testing.T) {
			require.Empty(t, tokensOver(t, doc, "q : r", ":"))
			require.Equal(t, "punctuation.special", categoryOf(t, doc, "p ? q", "?"))
		})

		t.Run("a line comment stops before the line break", func(t *testing.T) {
			require.Equal(t, "comment", categoryOf(t, doc, "# a hash", "# a hash comment"))
			require.Empty(t, tokensOver(t, doc, "# a hash comment\n", "\n"))
		})

		t.Run("whitespace between tokens has no token", func(t *testing.T) {
			require.Empty(t, tokensOver(t, doc, "a || b", " "))
			require.Equal(t, "operator", categoryOf(t, doc, "a || b", "||"))
		})

		t.Run("a heredoc's markers stop at their own text", func(t *testing.T) {
			require.Empty(t, tokensOver(t, doc, "<<-FLOW\n", "\n"))
			require.Empty(t, tokensOver(t, doc, "  FLOW\nplain", "  "))
			require.Equal(t, "punctuation.delimiter", categoryOf(t, doc, "  FLOW\nplain", "FLOW"))
		})
	})

	t.Run("for expressions across lines and comments", func(t *testing.T) {
		const doc = `multiline = [
  for key, val in map :
  key => val
  if val
]
commented = [ /* a */ for item /* b */ in /* c */ list : item /* d */ if /* e */ item ]
stacked = [ /* f */
  for entry in /* g */
  entries : entry ]
`

		cases := []struct {
			container string
			text      string
			category  string
		}{
			{"[\n  for key", "for", "keyword.repeat"},
			{"val in map", "in", "keyword.repeat"},
			{"  if val", "if", "keyword.conditional"},
			{"*/ for item", "for", "keyword.repeat"},
			{"*/ in /*", "in", "keyword.repeat"},
			{"*/ if /*", "if", "keyword.conditional"},
			{"*/\n  for entry", "for", "keyword.repeat"},
			{"entry in /* g", "in", "keyword.repeat"},
			{"/* a */", "/* a */", "comment"},
			{"/* c */", "/* c */", "comment"},
			{"/* e */", "/* e */", "comment"},
		}
		for _, c := range cases {
			t.Run(c.text+" in "+c.container+" is "+c.category, func(t *testing.T) {
				require.Equal(t, c.category, categoryOf(t, doc, c.container, c.text))
			})
		}

		t.Run("the line break after an opening bracket has no token", func(t *testing.T) {
			require.Empty(t, tokensOver(t, doc, "multiline = [\n", "\n"))
		})
	})

	t.Run("blocks", func(t *testing.T) {
		const doc = `context "Ordering" {
  mode = dcb

  slice "Place Order" {
    command "PlaceOrder" {
      fields {
        total = required(int)
      }
    }
  }
}

rule first second {
  nested bare {
  }
}
`

		cases := []struct {
			container string
			text      string
			category  string
		}{
			{`context "Ordering"`, "context", "keyword"},
			{`slice "Place`, "slice", "type"},
			{`command "PlaceOrder"`, "command", "type"},
			{"fields {", "fields", "type"},
			{"rule first", "rule", "keyword"},
			{"rule first", "first", "keyword"},
			{"first second", "second", "keyword"},
			{"nested bare", "nested", "type"},
			{"nested bare", "bare", "type"},
			{"mode = dcb", "mode", "variable.member"},
			{"mode = dcb", "dcb", "variable"},
			{"required(int)", "required", "function"},
			{"required(int)", "int", "variable"},
			{`slice "Place Order"`, "Place Order", "string"},
		}
		for _, c := range cases {
			t.Run(c.text+" in "+c.container+" is "+c.category, func(t *testing.T) {
				require.Equal(t, c.category, categoryOf(t, doc, c.container, c.text))
			})
		}
	})

	t.Run("construct names", func(t *testing.T) {
		const doc = `emod = 1

model "Shop" {
}

actor "Customer" {
}

actor "Clerk" {
}

context "Orders" {
  aggregate "Sales" {
    slice "Place Order" {
      command "PlaceOrder" {
      }

      command "CancelOrder" {
      }

      event "OrderPlaced" {
      }

      event "OrderCancelled" {
      }

      view "OrderList" {
        subscribes = [OrderPlaced]
      }

      view "OrderTotals" {
      }
    }
  }

  aggregate "Returns" {
  }
}

context "Billing" {
  aggregate "Invoices" {
    slice "Unnamed" {
      command "" {
      }
    }
  }
}
`

		cases := []struct {
			container string
			name      string
			category  string
		}{
			{`actor "Customer"`, "Customer", "lsp.type.parameter"},
			{`actor "Clerk"`, "Clerk", "lsp.type.parameter"},
			{`context "Orders"`, "Orders", "lsp.type.namespace"},
			{`context "Billing"`, "Billing", "lsp.type.namespace"},
			{`aggregate "Sales"`, "Sales", "lsp.type.struct"},
			{`aggregate "Returns"`, "Returns", "lsp.type.struct"},
			{`command "PlaceOrder"`, "PlaceOrder", "lsp.type.function"},
			{`command "CancelOrder"`, "CancelOrder", "lsp.type.function"},
			{`event "OrderPlaced"`, "OrderPlaced", "lsp.type.event"},
			{`event "OrderCancelled"`, "OrderCancelled", "lsp.type.event"},
			{`view "OrderList"`, "OrderList", "lsp.type.class"},
			{`view "OrderTotals"`, "OrderTotals", "lsp.type.class"},
		}
		for _, c := range cases {
			t.Run("the name "+c.name+" is "+c.category+" and its quotes stay string", func(t *testing.T) {
				require.Equal(t, c.category, categoryOf(t, doc, c.container, c.name))
				require.Equal(t, "string", categoryOf(t, doc, c.container, `"`))
				require.Equal(t, "string", categoryOf(t, doc, c.name, `"`))
			})
		}

		t.Run("an empty name leaves both quotes string", func(t *testing.T) {
			require.Equal(t, "string", categoryOf(t, doc, `command ""`, `""`))
		})

		t.Run("a bare label takes no construct colour, and neither does the token after it", func(t *testing.T) {
			const doc = "context A{\n}\n"

			requireOrdered(t, doc, lsp.GetTokens(doc))
			require.Equal(t, "keyword", categoryOf(t, doc, "context A", "A"))
			require.Equal(t, "punctuation.bracket", categoryOf(t, doc, "A{", "{"))
		})

		t.Run("a bare label at the end of the text leaves no empty token", func(t *testing.T) {
			const doc = "context A"

			tokens := lsp.GetTokens(doc)
			requireOrdered(t, doc, tokens)
			require.NotEmpty(t, tokens)
		})

		t.Run("a label emod lsp does not colour stays string", func(t *testing.T) {
			require.Equal(t, "string", categoryOf(t, doc, `slice "Place Order"`, "Place Order"))
			require.Equal(t, "string", categoryOf(t, doc, `model "Shop"`, "Shop"))
		})

		t.Run("a name where it is used keeps the category HCL gives it", func(t *testing.T) {
			require.Equal(t, "variable", categoryOf(t, doc, "[OrderPlaced]", "OrderPlaced"))
			require.Equal(t, "lsp.type.event", categoryOf(t, doc, `event "OrderPlaced"`, "OrderPlaced"))
		})

		t.Run("every bundled example names the constructs emod lsp colours, where it colours them", func(t *testing.T) {
			for _, path := range examplePaths(t) {
				t.Run(filepath.Base(path), func(t *testing.T) {
					source, err := os.ReadFile(path)
					require.NoError(t, err)
					doc := string(source)

					fromSemanticTokens := constructNamesInSemanticTokens(lsp.GetSemanticTokens(doc).Data)
					require.NotEmpty(t, fromSemanticTokens)
					require.Equal(t, fromSemanticTokens, constructNamesInTokens(doc, lsp.GetTokens(doc)))
				})
			}
		})
	})

	t.Run("offsets", func(t *testing.T) {
		const doc = "note = \"e\u0301 \U0001F468\u200d\U0001F469\u200d\U0001F467 \U0001D11E \U00010000 \uffff\" + tail\n"

		t.Run("count UTF-16 units, so each token selects its own text past combining marks, ZWJ sequences and astral characters", func(t *testing.T) {
			tokens := lsp.GetTokens(doc)
			require.Equal(t, []textToken{
				{"note", "variable.member"},
				{`"`, "string"},
				{"e\u0301 \U0001F468\u200d\U0001F469\u200d\U0001F467 \U0001D11E \U00010000 \uffff", "string"},
				{`"`, "string"},
				{"+", "operator"},
				{"tail", "variable"},
			}, textsOf(doc, tokens))
		})
	})

	t.Run("broken source", func(t *testing.T) {
		t.Run("an unclosed context keeps the colours of what HCL reads and of the names recovered above it", func(t *testing.T) {
			source, err := os.ReadFile("../../examples/all_patterns.emod")
			require.NoError(t, err)
			doc := string(source) + "\ncontext \"Unclosed\" {\n  slice \"Never Closed\" {\n"

			requireOrdered(t, doc, lsp.GetTokens(doc))
			require.Equal(t, "lsp.type.function", categoryOf(t, doc, `command "ReserveRoom"`, "ReserveRoom"))
			require.Equal(t, "lsp.type.namespace", categoryOf(t, doc, `context "Notifications"`, "Notifications"))
			require.Equal(t, "comment", categoryOf(t, doc, "# ", "#"))
			require.Equal(t, "number", categoryOf(t, doc, "emod = ", "1"))
			require.Equal(t, "string", categoryOf(t, doc, `slice "Never Closed"`, "Never Closed"))
			require.Equal(t, constructNamesInSemanticTokens(lsp.GetSemanticTokens(doc).Data), constructNamesInTokens(doc, lsp.GetTokens(doc)))
		})

		texts := []struct {
			name     string
			doc      string
			text     string
			category string
		}{
			{"an unterminated string", "x = \"abc\ny = 1\n", "abc", "string"},
			{"an unterminated heredoc", "x = <<EOF\nabc\n", "abc", "string"},
			{"an unterminated block comment", "x = 1\n/* abc\n", "1", "number"},
			{"invalid UTF-8", "x = \"\xff\"\ny = 2\n", "2", "number"},
			{"an unfinished for directive", "x = \"%{ for", "x", "variable.member"},
			{"a payload value still being typed after an earlier one", "model \"M\" {\n  description = \"a ${}\"\n}\n\ncontext \"C\" {\n  aggregate \"A\" {\n    slice \"S\" {\n      spec \"s\" {\n        when = Do({ id = \"b ${}\" })\n      }\n    }\n  }\n}\n", "context", "keyword"},
			{"an unclosed splat", "x = list[*\ny = \"s\"\n", "\"s\"", "string"},
			{"an unfinished interpolation in a description", "model \"M\" {\n  description = \"a ${}\"\n}\n", "description", "variable.member"},
		}
		for _, c := range texts {
			t.Run(c.name+" answers ordered tokens and colours what the lexer reads", func(t *testing.T) {
				requireOrdered(t, c.doc, lsp.GetTokens(c.doc))
				require.Equal(t, c.category, categoryOf(t, c.doc, c.text, c.text))
			})
		}

		t.Run("plain words answer ordered tokens", func(t *testing.T) {
			const doc = "this is not valid emod syntax"
			tokens := lsp.GetTokens(doc)
			require.NotEmpty(t, tokens)
			requireOrdered(t, doc, tokens)
		})
	})

	t.Run("empty document", func(t *testing.T) {
		t.Run("answers no tokens", func(t *testing.T) {
			require.Empty(t, lsp.GetTokens(""))
		})
	})
}

type textToken struct {
	text     string
	category string
}

func textsOf(doc string, tokens []lsp.Token) []textToken {
	units := utf16.Encode([]rune(doc))
	texts := make([]textToken, 0, len(tokens))
	for _, token := range tokens {
		texts = append(texts, textToken{string(utf16.Decode(units[token.Start:token.End])), token.Category})
	}
	return texts
}

func utf16RangeOf(t *testing.T, doc, container, text string) (int, int) {
	t.Helper()
	at := strings.Index(doc, container)
	require.GreaterOrEqual(t, at, 0, "container %q not found", container)
	rel := strings.Index(doc[at:], text)
	require.GreaterOrEqual(t, rel, 0, "text %q not found in container %q", text, container)
	start := len(utf16.Encode([]rune(doc[:at+rel])))
	return start, start + len(utf16.Encode([]rune(text)))
}

func tokensOver(t *testing.T, doc, container, text string) []lsp.Token {
	t.Helper()
	start, end := utf16RangeOf(t, doc, container, text)
	var over []lsp.Token
	for _, token := range lsp.GetTokens(doc) {
		if token.Start < end && token.End > start {
			over = append(over, token)
		}
	}
	return over
}

func categoryOf(t *testing.T, doc, container, text string) string {
	t.Helper()
	start, end := utf16RangeOf(t, doc, container, text)
	over := tokensOver(t, doc, container, text)
	require.NotEmpty(t, over, "no token colours %q", text)

	covered := start
	for _, token := range over {
		require.LessOrEqual(t, token.Start, covered, "a character of %q before %d has no token: %v", text, token.Start, over)
		require.Equal(t, over[0].Category, token.Category, "%q is coloured by more than one category: %v", text, over)
		covered = token.End
	}
	require.GreaterOrEqual(t, covered, end, "the end of %q has no token: %v", text, over)
	return over[0].Category
}

func requireOrdered(t *testing.T, doc string, tokens []lsp.Token) {
	t.Helper()
	limit := len(utf16.Encode([]rune(doc)))
	previousEnd := 0
	for _, token := range tokens {
		require.Less(t, token.Start, token.End, "an empty token: %v", token)
		require.GreaterOrEqual(t, token.Start, previousEnd, "a token out of order or overlapping: %v", token)
		require.LessOrEqual(t, token.End, limit, "a token past the end: %v", token)
		previousEnd = token.End
	}
}

type placedName struct {
	line      int
	character int
	tokenType string
}

func constructNamesInSemanticTokens(data []uint) []placedName {
	legend := lsp.GetSemanticTokensLegend().TokenTypes
	var names []placedName
	for _, token := range decodeTokens(data) {
		names = append(names, placedName{token.line, token.char, legend[token.typeIdx]})
	}
	return names
}

func constructNamesInTokens(doc string, tokens []lsp.Token) []placedName {
	units := utf16.Encode([]rune(doc))
	var names []placedName
	for _, token := range tokens {
		tokenType, ok := strings.CutPrefix(token.Category, "lsp.type.")
		if !ok {
			continue
		}
		before := units[:token.Start]
		line := 0
		lineStart := 0
		for i, unit := range before {
			if unit == '\n' {
				line++
				lineStart = i + 1
			}
		}
		names = append(names, placedName{line, token.Start - lineStart, tokenType})
	}
	sort.SliceStable(names, func(i, j int) bool {
		if names[i].line != names[j].line {
			return names[i].line < names[j].line
		}
		return names[i].character < names[j].character
	})
	return names
}

func examplePaths(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob("../../examples/*.emod")
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	return paths
}
