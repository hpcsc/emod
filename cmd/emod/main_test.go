//go:build integration

package main_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

var emodBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "emod-binary-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	emodBinary = filepath.Join(dir, "emod")

	build := exec.Command("go", "build", "-o", emodBinary, ".")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "building emod:", err)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}

func withoutAISettings(t *testing.T) []string {
	t.Helper()

	var env []string
	for _, variable := range os.Environ() {
		key, _, _ := strings.Cut(variable, "=")
		if strings.HasPrefix(key, "EMOD_AI_") || key == "ANTHROPIC_API_KEY" || key == "OPENAI_API_KEY" ||
			key == "HOME" || key == "XDG_CONFIG_HOME" {
			continue
		}
		env = append(env, variable)
	}
	return append(env, "HOME="+t.TempDir())
}

func startServer(t *testing.T) (*sdk.ClientSession, *exec.Cmd) {
	t.Helper()

	return startServerIn(t, repoRoot(t))
}

func startServerIn(t *testing.T, dir string, flags ...string) (*sdk.ClientSession, *exec.Cmd) {
	t.Helper()

	server := exec.Command(emodBinary, append([]string{"mcp"}, flags...)...)
	server.Dir = dir
	server.Env = withoutAISettings(t)
	server.Stderr = os.Stderr

	client := sdk.NewClient(&sdk.Implementation{Name: "test-host", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &sdk.CommandTransport{Command: server, TerminateDuration: 10 * time.Second}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session, server
}

func validateJSON(t *testing.T, path string) string {
	t.Helper()

	validate := exec.Command(emodBinary, "validate", "--format", "json", path)
	validate.Dir = repoRoot(t)
	validate.Env = withoutAISettings(t)
	output, err := validate.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		require.LessOrEqual(t, exitErr.ExitCode(), 2, "emod validate failed: %s", exitErr.Stderr)
	} else {
		require.NoError(t, err)
	}
	return string(output)
}

func explainRule(t *testing.T, rule string) string {
	t.Helper()

	explain := exec.Command(emodBinary, "lint", "--explain", rule)
	explain.Env = withoutAISettings(t)
	output, err := explain.Output()
	require.NoError(t, err)
	return strings.TrimSuffix(string(output), "\n")
}

func listSlicesCLI(t *testing.T, path string) (listed string, parseDiagnostics string) {
	t.Helper()

	list := exec.Command(emodBinary, "slices", "list", "--format", "json", path)
	list.Dir = repoRoot(t)
	list.Env = withoutAISettings(t)
	var stdout, stderr strings.Builder
	list.Stdout = &stdout
	list.Stderr = &stderr
	err := list.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		require.Equal(t, 1, exitErr.ExitCode(), "emod slices list failed: %s", stderr.String())
		require.NotEmpty(t, stderr.String())
		return "", strings.TrimSpace(stderr.String())
	}
	require.NoError(t, err)
	return stdout.String(), ""
}

func modelPaths(t *testing.T) []string {
	t.Helper()

	root := repoRoot(t)
	var paths []string
	for _, pattern := range []string{"examples/*.emod", "internal/parser/testdata/*.emod"} {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		require.NoError(t, err)
		for _, match := range matches {
			relative, err := filepath.Rel(root, match)
			require.NoError(t, err)
			paths = append(paths, filepath.ToSlash(relative))
		}
	}
	require.NotEmpty(t, paths)
	return paths
}

func exportCLI(t *testing.T, path, format string) (string, int) {
	t.Helper()

	command := exec.Command(emodBinary, "export", "--format", format, path)
	command.Dir = repoRoot(t)
	command.Env = withoutAISettings(t)
	output, err := command.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		require.LessOrEqual(t, exitErr.ExitCode(), 2, "emod export failed: %s", exitErr.Stderr)
		return string(output), exitErr.ExitCode()
	}
	require.NoError(t, err)
	return string(output), 0
}

func glossaryCLI(t *testing.T, path, format string) (printed string, parseDiagnostics string) {
	t.Helper()

	command := exec.Command(emodBinary, "glossary", "--format", format, path)
	command.Dir = repoRoot(t)
	command.Env = withoutAISettings(t)
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		require.Equal(t, 1, exitErr.ExitCode(), "emod glossary failed: %s", stderr.String())
		require.NotEmpty(t, stderr.String())
		return "", strings.TrimSpace(stderr.String())
	}
	require.NoError(t, err)
	return stdout.String(), ""
}

func fmtCLI(t *testing.T, path string) (formatted string, changed bool, parseDiagnostics string) {
	t.Helper()

	check := exec.Command(emodBinary, "fmt", "--check", path)
	check.Dir = repoRoot(t)
	check.Env = withoutAISettings(t)
	var stderr strings.Builder
	check.Stderr = &stderr
	err := check.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		require.Equal(t, 1, exitErr.ExitCode(), "emod fmt --check failed: %s", stderr.String())
		if stderr.String() != path+" is not formatted\n" {
			require.NotEmpty(t, stderr.String())
			return "", false, strings.TrimSpace(stderr.String())
		}
		changed = true
	} else {
		require.NoError(t, err)
	}

	source, err := os.ReadFile(filepath.Join(repoRoot(t), path))
	require.NoError(t, err)
	copied := filepath.Join(t.TempDir(), filepath.Base(path))
	require.NoError(t, os.WriteFile(copied, source, 0o644))
	write := exec.Command(emodBinary, "fmt", copied)
	write.Env = withoutAISettings(t)
	output, err := write.CombinedOutput()
	require.NoError(t, err, "emod fmt failed: %s", output)
	written, err := os.ReadFile(copied)
	require.NoError(t, err)
	return string(written), changed, ""
}

var (
	movedLine    = regexp.MustCompile(`^moved (\d+) slices?; backward references (\d+) -> (\d+)$`)
	unmovedLine  = regexp.MustCompile(`^already arranged; (\d+) backward references?$`)
	backwardLine = regexp.MustCompile(`^  \[([a-z]+)\] (.+) \((.+) -> (.+)\)$`)
)

func arrangeCLI(t *testing.T, path string) (fromCLI map[string]any, parseDiagnostics string) {
	t.Helper()

	check := exec.Command(emodBinary, "slices", "arrange", "--check", path)
	check.Dir = repoRoot(t)
	check.Env = withoutAISettings(t)
	var stderr strings.Builder
	check.Stderr = &stderr
	err := check.Run()
	changed := false
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		require.Equal(t, 1, exitErr.ExitCode(), "emod slices arrange --check failed: %s", stderr.String())
		if stderr.String() != path+" is not arranged\n" {
			require.NotEmpty(t, stderr.String())
			return nil, strings.TrimSpace(stderr.String())
		}
		changed = true
	} else {
		require.NoError(t, err)
	}

	source, err := os.ReadFile(filepath.Join(repoRoot(t), path))
	require.NoError(t, err)
	copied := filepath.Join(t.TempDir(), filepath.Base(path))
	require.NoError(t, os.WriteFile(copied, source, 0o644))
	write := exec.Command(emodBinary, "slices", "arrange", copied)
	write.Env = withoutAISettings(t)
	printed, err := write.Output()
	require.NoError(t, err, "emod slices arrange failed")
	written, err := os.ReadFile(copied)
	require.NoError(t, err)

	fromCLI = map[string]any{"arranged": string(written), "changed": changed, "backward": []any{}}
	lines := strings.Split(strings.TrimSuffix(string(printed), "\n"), "\n")
	if counts := movedLine.FindStringSubmatch(lines[0]); counts != nil {
		fromCLI["moved"], fromCLI["backwardBefore"], fromCLI["backwardAfter"] = number(t, counts[1]), number(t, counts[2]), number(t, counts[3])
	} else {
		counts := unmovedLine.FindStringSubmatch(lines[0])
		require.NotNil(t, counts, "the first line of the report must give the counts: %q", lines[0])
		unchangedCount := number(t, counts[1])
		fromCLI["moved"], fromCLI["backwardBefore"], fromCLI["backwardAfter"] = float64(0), unchangedCount, unchangedCount
	}
	var backward []any
	for _, line := range lines[1:] {
		reference := backwardLine.FindStringSubmatch(line)
		require.NotNil(t, reference, "each line after the counts must name a backward reference: %q", line)
		backward = append(backward, map[string]any{"kind": reference[1], "label": reference[2], "from": reference[3], "to": reference[4]})
	}
	if backward != nil {
		fromCLI["backward"] = backward
	}
	return fromCLI, ""
}

func number(t *testing.T, digits string) float64 {
	t.Helper()

	n, err := strconv.Atoi(digits)
	require.NoError(t, err)
	return float64(n)
}

func copyModel(t *testing.T, path, dir string) string {
	t.Helper()

	source, err := os.ReadFile(filepath.Join(repoRoot(t), path))
	require.NoError(t, err)
	copied := filepath.Join(dir, filepath.Base(path))
	require.NoError(t, os.WriteFile(copied, source, 0o644))
	return copied
}

func fileText(t *testing.T, path string) string {
	t.Helper()

	text, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(text)
}

func diagnosticLines(t *testing.T, diagnostics any) string {
	t.Helper()

	encoded, err := json.Marshal(diagnostics)
	require.NoError(t, err)
	var records []struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(encoded, &records))
	require.NotEmpty(t, records)
	lines := make([]string, 0, len(records))
	for _, record := range records {
		lines = append(lines, fmt.Sprintf("%s:%d: %s", record.File, record.Line, record.Message))
	}
	return strings.Join(lines, "\n")
}

func requireDiagramParity(t *testing.T, path string, options map[string]any) {
	t.Helper()

	session, _ := startServer(t)
	written, exitCode := diagramCLI(t, path, options)
	arguments := map[string]any{"path": path}
	maps.Copy(arguments, options)

	result := callTool(t, session, "emod_diagram", arguments)

	require.False(t, result.IsError, "the call must succeed: %v", result.Content)
	structured, ok := result.StructuredContent.(map[string]any)
	require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
	if exitCode == 2 {
		require.NotContains(t, structured, "diagram")
		records, err := json.Marshal(structured["diagnostics"])
		require.NoError(t, err)
		require.JSONEq(t, validateJSON(t, path), string(records))
		return
	}
	require.Equal(t, written, structured["diagram"])
	require.Equal(t, written, textOf(t, &sdk.CallToolResult{Content: result.Content[:1]}))
}

func diagramCLI(t *testing.T, path string, options map[string]any) (string, int) {
	t.Helper()

	format, _ := options["format"].(string)
	file := filepath.Join(t.TempDir(), "diagram")
	arguments := []string{"diagram", "--format", format, "-o", file}
	if style, ok := options["style"].(string); ok {
		arguments = append(arguments, "--style", style)
	}
	if specs, _ := options["specs"].(bool); specs {
		arguments = append(arguments, "--specs")
	}
	command := exec.Command(emodBinary, append(arguments, path)...)
	command.Dir = repoRoot(t)
	command.Env = withoutAISettings(t)
	output, err := command.CombinedOutput()
	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	} else {
		require.NoError(t, err)
	}
	require.LessOrEqual(t, exitCode, 2, "emod diagram failed: %s", output)
	if exitCode == 2 {
		return "", exitCode
	}
	drawn, err := os.ReadFile(file)
	require.NoError(t, err, "emod diagram must write the file: %s", output)
	return string(drawn), exitCode
}

func callTool(t *testing.T, session *sdk.ClientSession, name string, arguments map[string]any) *sdk.CallToolResult {
	t.Helper()

	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: arguments})
	require.NoError(t, err)
	return result
}

func textOf(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()

	require.Len(t, result.Content, 1)
	text, ok := result.Content[0].(*sdk.TextContent)
	require.True(t, ok, "the content must be a text block, but it is %T", result.Content[0])
	return text.Text
}

func printedSchema(t *testing.T) string {
	t.Helper()

	schema := exec.Command(emodBinary, "schema")
	schema.Env = withoutAISettings(t)
	output, err := schema.Output()
	require.NoError(t, err)
	require.NotEmpty(t, output)
	return string(output)
}

func TestMCP(t *testing.T) {
	t.Run("handshake", func(t *testing.T) {
		t.Run("the binary names itself emod, with the version that emod version prints", func(t *testing.T) {
			printed, err := exec.Command(emodBinary, "version").Output()
			require.NoError(t, err)
			session, _ := startServer(t)

			info := session.InitializeResult().ServerInfo

			require.Equal(t, &sdk.Implementation{Name: "emod", Version: strings.TrimSpace(string(printed))}, info)
		})

		t.Run("the binary lists each tool", func(t *testing.T) {
			session, _ := startServer(t)

			listed, err := session.ListTools(context.Background(), nil)

			require.NoError(t, err)
			names := make([]string, 0, len(listed.Tools))
			for _, tool := range listed.Tools {
				names = append(names, tool.Name)
			}
			require.ElementsMatch(t, []string{"emod_check", "emod_diagram", "emod_explain_rule", "emod_export", "emod_fmt", "emod_glossary", "emod_schema", "emod_slices_arrange", "emod_slices_list"}, names)
		})
	})

	t.Run("explain rule", func(t *testing.T) {
		t.Run("the binary gives the description that emod lint --explain prints", func(t *testing.T) {
			session, _ := startServer(t)

			result, err := session.CallTool(context.Background(), &sdk.CallToolParams{
				Name:      "emod_explain_rule",
				Arguments: map[string]any{"rule": "state-obsession"},
			})

			require.NoError(t, err)
			require.False(t, result.IsError)
			require.Equal(t, map[string]any{"rule": "state-obsession", "description": explainRule(t, "state-obsession")},
				result.StructuredContent)
		})
	})

	t.Run("schema", func(t *testing.T) {
		t.Run("the binary gives the text that emod schema prints", func(t *testing.T) {
			session, _ := startServer(t)

			result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "emod_schema", Arguments: map[string]any{}})

			require.NoError(t, err)
			require.False(t, result.IsError)
			require.Len(t, result.Content, 1)
			text, ok := result.Content[0].(*sdk.TextContent)
			require.True(t, ok)
			require.Equal(t, printedSchema(t), text.Text)
		})
	})

	t.Run("read grammar", func(t *testing.T) {
		t.Run("the binary serves the schema, each example and the DSL reference from an empty working directory", func(t *testing.T) {
			root := repoRoot(t)
			expected := map[string]string{"emod://schema": printedSchema(t)}
			for uri, path := range map[string]string{
				"emod://examples/all_patterns": "examples/all_patterns.emod",
				"emod://examples/dcb_model":    "examples/dcb_model.emod",
				"emod://examples/specs_hotel":  "examples/specs_hotel.emod",
				"emod://docs/dsl-reference":    "docs/dsl-reference.md",
			} {
				text, err := os.ReadFile(filepath.Join(root, path))
				require.NoError(t, err)
				expected[uri] = string(text)
			}
			session, _ := startServerIn(t, t.TempDir())

			for uri, text := range expected {
				read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: uri})

				require.NoError(t, err, "resource %s", uri)
				require.Len(t, read.Contents, 1, "resource %s", uri)
				require.Equal(t, text, read.Contents[0].Text, "resource %s", uri)
			}
		})
	})

	t.Run("read rules", func(t *testing.T) {
		t.Run("the binary lists each rule with the description that emod lint --explain prints for it", func(t *testing.T) {
			session, _ := startServer(t)

			read, err := session.ReadResource(context.Background(), &sdk.ReadResourceParams{URI: "emod://rules"})

			require.NoError(t, err)
			require.Len(t, read.Contents, 1)
			var listed struct {
				Rules []struct {
					Rule        string `json:"rule"`
					Description string `json:"description"`
				} `json:"rules"`
			}
			require.NoError(t, json.Unmarshal([]byte(read.Contents[0].Text), &listed))
			require.Len(t, listed.Rules, 24)
			for _, rule := range listed.Rules {
				require.Equal(t, explainRule(t, rule.Rule), rule.Description, "rule %s", rule.Rule)
			}
		})
	})

	t.Run("check", func(t *testing.T) {
		t.Run("the binary refuses a path outside its working directory, and names the root", func(t *testing.T) {
			session, _ := startServer(t)

			result, err := session.CallTool(context.Background(), &sdk.CallToolParams{
				Name:      "emod_check",
				Arguments: map[string]any{"path": "../outside.emod"},
			})

			require.NoError(t, err)
			require.True(t, result.IsError)
			require.Len(t, result.Content, 1)
			text, ok := result.Content[0].(*sdk.TextContent)
			require.True(t, ok)
			require.Equal(t, `path "../outside.emod" is outside the root `+repoRoot(t), text.Text)
		})

		paths := modelPaths(t)
		require.Contains(t, paths, "examples/error_diagnostics_test.emod", "the files must include one with diagnostics")

		for _, path := range paths {
			t.Run("gives the records of emod validate for the path "+path, func(t *testing.T) {
				session, _ := startServer(t)

				result, err := session.CallTool(context.Background(), &sdk.CallToolParams{
					Name:      "emod_check",
					Arguments: map[string]any{"path": path},
				})

				require.NoError(t, err)
				require.False(t, result.IsError, "the call must succeed: %v", result.Content)
				structured, ok := result.StructuredContent.(map[string]any)
				require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
				records, err := json.Marshal(structured["diagnostics"])
				require.NoError(t, err)
				require.JSONEq(t, validateJSON(t, path), string(records))
			})

			t.Run("gives the records of emod validate for the content of "+path, func(t *testing.T) {
				source, err := os.ReadFile(filepath.Join(repoRoot(t), path))
				require.NoError(t, err)
				session, _ := startServer(t)

				result, err := session.CallTool(context.Background(), &sdk.CallToolParams{
					Name:      "emod_check",
					Arguments: map[string]any{"content": string(source), "name": path},
				})

				require.NoError(t, err)
				require.False(t, result.IsError)
				structured, ok := result.StructuredContent.(map[string]any)
				require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
				records, err := json.Marshal(structured["diagnostics"])
				require.NoError(t, err)
				require.JSONEq(t, validateJSON(t, path), string(records))
			})
		}
	})

	t.Run("list slices", func(t *testing.T) {
		paths := modelPaths(t)
		require.Contains(t, paths, "internal/parser/testdata/invalid.emod", "the files must include one that does not parse")

		for _, path := range paths {
			t.Run("gives the slices or the parse diagnostics that emod slices list gives for the path "+path, func(t *testing.T) {
				session, _ := startServer(t)

				result, err := session.CallTool(context.Background(), &sdk.CallToolParams{
					Name:      "emod_slices_list",
					Arguments: map[string]any{"path": path},
				})

				require.NoError(t, err)
				require.False(t, result.IsError, "the call must succeed: %v", result.Content)
				structured, ok := result.StructuredContent.(map[string]any)
				require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
				listed, parseDiagnostics := listSlicesCLI(t, path)
				if parseDiagnostics == "" {
					require.NotContains(t, structured, "diagnostics")
					slices, err := json.Marshal(structured["slices"])
					require.NoError(t, err)
					require.JSONEq(t, listed, string(slices))
					return
				}
				require.NotContains(t, structured, "slices")
				require.Equal(t, parseDiagnostics, diagnosticLines(t, structured["diagnostics"]))
			})
		}
	})

	t.Run("export", func(t *testing.T) {
		for _, path := range modelPaths(t) {
			for _, format := range []string{"json", "diagram-json"} {
				t.Run("gives the "+format+" document of emod export for the path "+path, func(t *testing.T) {
					session, _ := startServer(t)
					printed, _ := exportCLI(t, path, format)

					result := callTool(t, session, "emod_export", map[string]any{"path": path, "format": format})

					require.False(t, result.IsError, "the call must succeed: %v", result.Content)
					document, err := json.Marshal(result.StructuredContent)
					require.NoError(t, err)
					require.JSONEq(t, printed, string(document))
				})
			}

			t.Run("gives the cue of emod export, or the records of emod validate, for the path "+path, func(t *testing.T) {
				session, _ := startServer(t)
				printed, exitCode := exportCLI(t, path, "cue")

				result := callTool(t, session, "emod_export", map[string]any{"path": path, "format": "cue"})

				require.False(t, result.IsError, "the call must succeed: %v", result.Content)
				if exitCode == 0 {
					require.Nil(t, result.StructuredContent)
					require.Equal(t, printed, textOf(t, result)+"\n")
					return
				}
				require.Empty(t, printed)
				structured, ok := result.StructuredContent.(map[string]any)
				require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
				records, err := json.Marshal(structured["diagnostics"])
				require.NoError(t, err)
				require.JSONEq(t, validateJSON(t, path), string(records))
			})
		}
	})

	t.Run("glossary", func(t *testing.T) {
		for _, path := range modelPaths(t) {
			for _, format := range []string{"markdown", "json"} {
				t.Run("gives the "+format+" glossary or the parse diagnostics that emod glossary gives for the path "+path, func(t *testing.T) {
					session, _ := startServer(t)
					printed, parseDiagnostics := glossaryCLI(t, path, format)

					result := callTool(t, session, "emod_glossary", map[string]any{"path": path, "format": format})

					require.False(t, result.IsError, "the call must succeed: %v", result.Content)
					switch {
					case parseDiagnostics != "":
						structured, ok := result.StructuredContent.(map[string]any)
						require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
						require.Equal(t, parseDiagnostics, diagnosticLines(t, structured["diagnostics"]))
					case format == "json":
						document, err := json.Marshal(result.StructuredContent)
						require.NoError(t, err)
						require.JSONEq(t, printed, string(document))
					default:
						require.Nil(t, result.StructuredContent)
						require.Equal(t, printed, textOf(t, result))
					}
				})
			}
		}
	})

	t.Run("diagram", func(t *testing.T) {
		for _, path := range modelPaths(t) {
			for _, format := range []string{"svg", "mermaid", "ascii", "event-flow", "event-flow-mermaid", "event-flow-mermaid-no-icons"} {
				t.Run("gives the "+format+" of emod diagram, or the records of emod validate, for the path "+path, func(t *testing.T) {
					requireDiagramParity(t, path, map[string]any{"format": format})
				})
			}
		}

		for _, style := range []string{"auto", "projected", "dcb"} {
			t.Run("gives the mermaid of emod diagram --style "+style+" for a dcb model", func(t *testing.T) {
				requireDiagramParity(t, "examples/dcb_model.emod", map[string]any{"format": "mermaid", "style": style})
			})
		}

		t.Run("gives the svg of emod diagram --specs for a model with specs", func(t *testing.T) {
			requireDiagramParity(t, "examples/specs_hotel.emod", map[string]any{"format": "svg", "specs": true})
		})
	})

	t.Run("fmt", func(t *testing.T) {
		paths := modelPaths(t)
		require.Contains(t, paths, "internal/parser/testdata/invalid.emod", "the files must include one that does not parse")
		_, changed, _ := fmtCLI(t, "examples/error_diagnostics_test.emod")
		require.True(t, changed, "the files must include one that is not formatted, or no leaf sees changed true")
		_, changed, _ = fmtCLI(t, "examples/all_patterns.emod")
		require.False(t, changed, "the files must include one that is formatted, or no leaf sees changed false")

		for _, path := range paths {
			t.Run("gives the text that emod fmt writes and the answer of emod fmt --check, or the parse diagnostics, for the path "+path, func(t *testing.T) {
				session, _ := startServer(t)
				formatted, changed, parseDiagnostics := fmtCLI(t, path)

				result := callTool(t, session, "emod_fmt", map[string]any{"path": path})

				require.False(t, result.IsError, "the call must succeed: %v", result.Content)
				structured, ok := result.StructuredContent.(map[string]any)
				require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
				if parseDiagnostics != "" {
					require.NotContains(t, structured, "formatted")
					require.Equal(t, parseDiagnostics, diagnosticLines(t, structured["diagnostics"]))
					return
				}
				require.Equal(t, map[string]any{"formatted": formatted, "changed": changed}, structured)
			})
		}
	})

	t.Run("arrange slices", func(t *testing.T) {
		paths := modelPaths(t)
		require.Contains(t, paths, "internal/parser/testdata/invalid.emod", "the files must include one that does not parse")
		movedFile, _ := arrangeCLI(t, "examples/all_patterns.emod")
		require.Equal(t, true, movedFile["changed"], "the files must include one in which a slice moves")
		keptFile, _ := arrangeCLI(t, "examples/specs_hotel.emod")
		require.Equal(t, false, keptFile["changed"], "the files must include one in which no slice moves")
		require.NotEmpty(t, keptFile["backward"], "the files must include one with a reference that stays backward")

		for _, path := range paths {
			t.Run("gives the text that emod slices arrange writes, the report that it prints and the answer of --check, or the parse diagnostics, for the path "+path, func(t *testing.T) {
				session, _ := startServer(t)
				fromCLI, parseDiagnostics := arrangeCLI(t, path)

				result := callTool(t, session, "emod_slices_arrange", map[string]any{"path": path})

				require.False(t, result.IsError, "the call must succeed: %v", result.Content)
				structured, ok := result.StructuredContent.(map[string]any)
				require.True(t, ok, "the structured result must be an object, but it is %T", result.StructuredContent)
				if parseDiagnostics != "" {
					require.NotContains(t, structured, "arranged")
					require.Equal(t, parseDiagnostics, diagnosticLines(t, structured["diagnostics"]))
					return
				}
				require.Equal(t, fromCLI, structured)
			})
		}
	})

	t.Run("write", func(t *testing.T) {
		t.Run("the binary with --allow-write writes the text that emod fmt writes to an unformatted copy of a model", func(t *testing.T) {
			formatted, changed, _ := fmtCLI(t, "examples/error_diagnostics_test.emod")
			require.True(t, changed, "the fixture must not be formatted, or the leaf proves nothing")
			dir := t.TempDir()
			copied := copyModel(t, "examples/error_diagnostics_test.emod", dir)
			session, _ := startServerIn(t, dir, "--allow-write")

			result := callTool(t, session, "emod_fmt", map[string]any{"path": "error_diagnostics_test.emod", "write": true})

			require.False(t, result.IsError, "the call must succeed: %v", result.Content)
			require.Equal(t, formatted, fileText(t, copied))
		})

		t.Run("the binary with --allow-write writes the text that emod slices arrange writes to an unarranged copy of a model", func(t *testing.T) {
			fromCLI, _ := arrangeCLI(t, "examples/all_patterns.emod")
			require.Equal(t, true, fromCLI["changed"], "the fixture must not be arranged, or the leaf proves nothing")
			dir := t.TempDir()
			copied := copyModel(t, "examples/all_patterns.emod", dir)
			session, _ := startServerIn(t, dir, "--allow-write")

			result := callTool(t, session, "emod_slices_arrange", map[string]any{"path": "all_patterns.emod", "write": true})

			require.False(t, result.IsError, "the call must succeed: %v", result.Content)
			require.Equal(t, fromCLI["arranged"], fileText(t, copied))
		})

		for _, call := range []struct{ name, path string }{
			{"emod_fmt", "examples/error_diagnostics_test.emod"},
			{"emod_slices_arrange", "examples/all_patterns.emod"},
		} {
			t.Run("the binary with no flag gives an error result that names --allow-write for a call of "+call.name+" with write on a file that it changes, and the file does not change", func(t *testing.T) {
				dir := t.TempDir()
				copied := copyModel(t, call.path, dir)
				before := fileText(t, copied)
				session, _ := startServerIn(t, dir)

				result := callTool(t, session, call.name, map[string]any{"path": filepath.Base(call.path), "write": true})

				require.True(t, result.IsError)
				require.Contains(t, textOf(t, result), "emod mcp --allow-write")
				require.Equal(t, before, fileText(t, copied))
			})
		}
	})

	t.Run("project file", func(t *testing.T) {
		t.Run("the emod entry of the .mcp.json of the repository starts a server in the repository that checks a model there", func(t *testing.T) {
			var registered struct {
				MCPServers map[string]struct {
					Command string   `json:"command"`
					Args    []string `json:"args"`
				} `json:"mcpServers"`
			}
			require.NoError(t, json.Unmarshal([]byte(fileText(t, filepath.Join(repoRoot(t), ".mcp.json"))), &registered))
			require.Contains(t, registered.MCPServers, "gopls", "the entry of emod must not replace the other servers")
			entry, ok := registered.MCPServers["emod"]
			require.True(t, ok, "the .mcp.json of the repository must register emod")
			require.Equal(t, "emod", entry.Command)
			require.Equal(t, []string{"mcp", "--root", "."}, entry.Args)
			session, _ := startServerIn(t, repoRoot(t), entry.Args[1:]...)

			result := callTool(t, session, "emod_check", map[string]any{"path": "examples/all_patterns.emod"})

			require.False(t, result.IsError, "the call must succeed: %v", result.Content)
			require.Equal(t, map[string]any{"diagnostics": []any{}}, result.StructuredContent)
		})
	})

	t.Run("shutdown", func(t *testing.T) {
		t.Run("the server ends with exit code 0 when the host closes its stdin", func(t *testing.T) {
			session, server := startServer(t)

			err := session.Close()

			require.NoError(t, err)
			require.Equal(t, 0, server.ProcessState.ExitCode())
		})
	})
}
