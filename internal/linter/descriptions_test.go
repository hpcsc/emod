//go:build unit

package linter_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hpcsc/emod/internal/linter"
)

func TestRuleDescription(t *testing.T) {
	t.Run("list rules", func(t *testing.T) {
		t.Run("lists the rules in the order of their names, with the dcb, spec and validator rules among them", func(t *testing.T) {
			names := namesOf(linter.Rules())

			require.True(t, slices.IsSorted(names), "the names must be in order: %v", names)
			require.Subset(t, names, []string{
				"dcb/orphan-tag-key",
				"orphan-command",
				"orphan-event",
				"spec/given-outside-boundary",
				"state-obsession",
			})
		})

		t.Run("gives each rule the description that RuleDescription gives for its name", func(t *testing.T) {
			for _, rule := range linter.Rules() {
				description, err := linter.RuleDescription(rule.Name)

				require.NoError(t, err, "rule %s", rule.Name)
				require.NotEmpty(t, rule.Description, "rule %s", rule.Name)
				require.Equal(t, description, rule.Description, "rule %s", rule.Name)
			}
		})

		t.Run("lists the rules that the linter and the validator report, once each, and no other rule", func(t *testing.T) {
			reported := append(
				reportedRules(t, ".", map[string]int{"info": 1, "warning": 1, "errorEntry": 1}),
				reportedRules(t, "../validator", map[string]int{"orphanDiagnostics": 2})...,
			)
			slices.Sort(reported)

			require.Equal(t, slices.Compact(reported), namesOf(linter.Rules()))
		})
	})

	t.Run("describe", func(t *testing.T) {
		t.Run("a rule name gives the description of the rule", func(t *testing.T) {
			description, err := linter.RuleDescription("state-obsession")

			require.NoError(t, err)
			require.Contains(t, description, `"OrderShipped" instead of "OrderUpdated"`)
		})

		t.Run("an unknown rule name gives an error that names the rule", func(t *testing.T) {
			_, err := linter.RuleDescription("dcb/nonexistent")

			require.EqualError(t, err, `unknown rule "dcb/nonexistent"`)
		})
	})
}

func namesOf(rules []linter.Rule) []string {
	names := make([]string, 0, len(rules))
	for _, rule := range rules {
		names = append(names, rule.Name)
	}
	return names
}

func reportedRules(t *testing.T, dir string, ruleIndexOf map[string]int) []string {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	files := token.NewFileSet()
	var names []string
	calls := map[string]int{}
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(files, path, nil, 0)
		require.NoError(t, err)
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			ast.Inspect(function, func(node ast.Node) bool {
				if setsRuleName(node) {
					_, isHelper := ruleIndexOf[function.Name.Name]
					require.True(t, isHelper, "only %v can set a rule name, but %s sets one at %s",
						slices.Sorted(maps.Keys(ruleIndexOf)), function.Name.Name, files.Position(node.Pos()))
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee, ok := call.Fun.(*ast.Ident)
				if !ok {
					return true
				}
				index, ok := ruleIndexOf[callee.Name]
				if !ok {
					return true
				}
				calls[callee.Name]++
				literal, ok := call.Args[index].(*ast.BasicLit)
				require.True(t, ok && literal.Kind == token.STRING,
					"the rule name of the call to %s at %s must be a string literal", callee.Name, files.Position(call.Pos()))
				name, err := strconv.Unquote(literal.Value)
				require.NoError(t, err)
				names = append(names, name)
				return true
			})
		}
	}
	for helper := range ruleIndexOf {
		require.Positive(t, calls[helper], "the package in %s must call %s at least once", dir, helper)
	}
	return names
}

func setsRuleName(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.KeyValueExpr:
		key, ok := node.Key.(*ast.Ident)
		return ok && key.Name == "RuleName"
	case *ast.AssignStmt:
		for _, target := range node.Lhs {
			if selector, ok := target.(*ast.SelectorExpr); ok && selector.Sel.Name == "RuleName" {
				return true
			}
		}
	}
	return false
}
