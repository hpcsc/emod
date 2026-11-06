//go:build unit

package cli_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/config"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
	urfave "github.com/urfave/cli/v3"
)

func withoutAISettings(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range []string{"EMOD_AI_PROVIDER", "EMOD_AI_API_KEY", "EMOD_AI_MODEL", "EMOD_AI_MODEL_CHEAP", "EMOD_AI_EFFORT", "EMOD_AI_ENDPOINT", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"} {
		t.Setenv(key, "")
	}

	return home
}

func requireExitCode(t *testing.T, expected int, err error) {
	t.Helper()

	var exit urfave.ExitCoder
	require.ErrorAs(t, err, &exit)
	require.Equal(t, expected, exit.ExitCode())
}

func writeAIConfig(t *testing.T, home, contents string) string {
	t.Helper()

	path := filepath.Join(home, ".config", "emod", "config")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}

func wireText(t *testing.T, blocks any) string {
	t.Helper()

	list, ok := blocks.([]any)
	require.True(t, ok, "content must be a list of blocks, got %T", blocks)
	require.Len(t, list, 1)
	block, ok := list[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "text", block["type"])
	text, ok := block["text"].(string)
	require.True(t, ok)

	return text
}

func wireMessages(t *testing.T, body map[string]any) []llm.Message {
	t.Helper()

	raw, ok := body["messages"].([]any)
	require.True(t, ok, "the request must carry messages")
	messages := make([]llm.Message, 0, len(raw))
	for _, entry := range raw {
		message, ok := entry.(map[string]any)
		require.True(t, ok)
		role, _ := message["role"].(string)
		messages = append(messages, llm.Message{Role: role, Content: wireText(t, message["content"])})
	}

	return messages
}

func requireSetting(t *testing.T, output, label, value string) {
	t.Helper()

	require.Regexp(t, `(?m)^\s*`+regexp.QuoteMeta(label)+`:\s+`+regexp.QuoteMeta(value)+`$`, output)
}

func configLoader(cfg config.Config, err error) func() (config.Config, error) {
	return func() (config.Config, error) { return cfg, err }
}

func connectTo(model llm.Model) func(context.Context, config.Config) (llm.Model, error) {
	return func(context.Context, config.Config) (llm.Model, error) { return model, nil }
}

func usedAnswer(text string, inputTokens, outputTokens int) llm.Response {
	return llm.Response{Text: text, Usage: llm.Usage{InputTokens: inputTokens, OutputTokens: outputTokens}}
}

type apiGateway struct {
	server *httptest.Server

	mu      sync.Mutex
	answers []string
	bodies  []map[string]any
	apiKeys []string
}

func newAPIGateway(t *testing.T, answers ...string) *apiGateway {
	t.Helper()

	gateway := &apiGateway{answers: answers}
	gateway.server = httptest.NewServer(http.HandlerFunc(gateway.answer))
	t.Cleanup(gateway.server.Close)

	return gateway
}

func (g *apiGateway) answer(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()

	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	g.bodies = append(g.bodies, body)
	g.apiKeys = append(g.apiKeys, r.Header.Get("X-Api-Key"))

	if len(g.answers) == 0 {
		http.Error(w, `{"type":"error","error":{"type":"api_error","message":"no answer queued"}}`, http.StatusInternalServerError)
		return
	}
	text := g.answers[0]
	g.answers = g.answers[1:]

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":          "msg_test",
		"type":        "message",
		"role":        "assistant",
		"model":       body["model"],
		"content":     []map[string]any{{"type": "text", "text": text}},
		"stop_reason": "end_turn",
		"usage":       map[string]any{"input_tokens": 1000, "output_tokens": 200},
	})
}

type chatGateway struct {
	server *httptest.Server

	mu             sync.Mutex
	answers        []string
	paths          []string
	bodies         []map[string]any
	authorizations []string
}

func newChatGateway(t *testing.T, answers ...string) *chatGateway {
	t.Helper()

	gateway := &chatGateway{answers: answers}
	gateway.server = httptest.NewServer(http.HandlerFunc(gateway.answer))
	t.Cleanup(gateway.server.Close)

	return gateway
}

func (g *chatGateway) answer(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()

	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	g.paths = append(g.paths, r.URL.Path)
	g.bodies = append(g.bodies, body)
	g.authorizations = append(g.authorizations, r.Header.Get("Authorization"))

	if len(g.answers) == 0 {
		http.Error(w, `{"error":{"message":"no answer queued"}}`, http.StatusInternalServerError)
		return
	}
	text := g.answers[0]
	g.answers = g.answers[1:]

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": text}, "finish_reason": "stop"}},
		"usage":   map[string]any{"prompt_tokens": 700, "completion_tokens": 150},
	})
}

func TestAI(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		t.Run("prints how to turn AI on and succeeds without connecting", func(t *testing.T) {
			connected := false
			connect := func(context.Context, config.Config) (llm.Model, error) {
				connected = true
				return llm.NewFake(), nil
			}

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunAI(context.Background(), configLoader(config.Config{}, config.ErrNotConfigured), connect))
			})

			require.Contains(t, output, "AI is not configured")
			require.Contains(t, output, "EMOD_AI_API_KEY")
			require.False(t, connected)
		})

		t.Run("emod ai with no AI setting in the environment or the file prints the setup message", func(t *testing.T) {
			withoutAISettings(t)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai"))
			})

			require.Contains(t, output, "AI is not configured")
		})

		t.Run("emod ai reports that AI is not configured beside the files of the desktop app", func(t *testing.T) {
			home := withoutAISettings(t)
			require.NoError(t, os.MkdirAll(filepath.Join(home, ".config", "emod"), 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(home, ".config", "emod", "recent-files.json"), []byte("[]"), 0o600))

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai"))
			})

			require.Contains(t, output, "AI is not configured")
		})
	})

	t.Run("invalid configuration", func(t *testing.T) {
		t.Run("an AI config file that cannot be read stops emod ai but not the commands that do not use AI", func(t *testing.T) {
			home := withoutAISettings(t)
			configFile := writeAIConfig(t, home, "not a setting\n")
			path := writeTemp(t, "model.emod", validEmod)

			var err error
			stderr := captureStderr(t, func() {
				err = runCommandLine(t, "emod", "ai")
			})
			requireExitCode(t, 1, err)
			require.Contains(t, stderr, "config file "+configFile+" could not be parsed")

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "export", path, "--format", "cue"))
			})
			require.Contains(t, output, "name: ")
		})

		t.Run("returns the configuration error and does not connect", func(t *testing.T) {
			invalid := errors.New(`EMOD_AI_EFFORT has an invalid value "highest"`)
			connected := false
			connect := func(context.Context, config.Config) (llm.Model, error) {
				connected = true
				return llm.NewFake(), nil
			}

			err := cli.RunAI(context.Background(), configLoader(config.Config{}, invalid), connect)

			require.ErrorIs(t, err, invalid)
			require.False(t, connected)
		})

		t.Run("emod ai with an AI setting but no API key exits 1 and names the key variables", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_MODEL", "claude-opus-4-8")

			var err error
			stderr := captureStderr(t, func() {
				err = runCommandLine(t, "emod", "ai")
			})

			requireExitCode(t, 1, err)
			require.Contains(t, stderr, "an API key is required: set EMOD_AI_API_KEY or ANTHROPIC_API_KEY in the environment")
		})

		t.Run("emod ai with an effort flag that is not a level exits 1 and names the effort", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_API_KEY", "test-api-key")

			var err error
			stderr := captureStderr(t, func() {
				err = runCommandLine(t, "emod", "ai", "--ai-effort", "highest")
			})

			requireExitCode(t, 1, err)
			require.Contains(t, stderr, `EMOD_AI_EFFORT has an invalid value "highest"`)
		})

		t.Run("emod ai with the openai provider and no model exits 1 and names the model", func(t *testing.T) {
			withoutAISettings(t)

			var err error
			stderr := captureStderr(t, func() {
				err = runCommandLine(t, "emod", "ai", "--ai-provider", "openai")
			})

			requireExitCode(t, 1, err)
			require.Contains(t, stderr, "EMOD_AI_MODEL is required for the openai provider")
		})
	})

	t.Run("settings", func(t *testing.T) {
		t.Run("emod ai reports each setting that its flags give and sends the call with the key through the gateway", func(t *testing.T) {
			withoutAISettings(t)
			gateway := newAPIGateway(t, test.BillingPayments)
			t.Setenv("EMOD_AI_API_KEY", "test-api-key")

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai",
					"--ai-provider", "anthropic",
					"--ai-model", "claude-opus-5-5",
					"--ai-model-cheap", "claude-sonnet-5-5",
					"--ai-effort", "high",
					"--ai-endpoint", gateway.server.URL,
				))
			})

			requireSetting(t, output, "Provider", "anthropic")
			requireSetting(t, output, "Model", "claude-opus-5-5")
			requireSetting(t, output, "Cheap model", "claude-sonnet-5-5")
			requireSetting(t, output, "Effort", "high")
			requireSetting(t, output, "Endpoint", gateway.server.URL)
			require.Len(t, gateway.bodies, 1)
			require.Equal(t, "claude-opus-5-5", gateway.bodies[0]["model"])
			require.Equal(t, map[string]any{"effort": "high"}, gateway.bodies[0]["output_config"])
			require.Equal(t, []string{"test-api-key"}, gateway.apiKeys)
			require.NotContains(t, output, "Notice:")
		})

		t.Run("emod ai sends the call to an OpenAI-compatible server with no key, and says that the model is outside the tested tier", func(t *testing.T) {
			withoutAISettings(t)
			gateway := newChatGateway(t, test.BillingPayments)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai",
					"--ai-provider", "openai",
					"--ai-model", "llama3.1",
					"--ai-endpoint", gateway.server.URL+"/v1",
				))
			})

			requireSetting(t, output, "Provider", "openai")
			requireSetting(t, output, "Cheap model", "llama3.1")
			requireSetting(t, output, "API key", "none")
			require.Contains(t, output, "Notice: emod tests its AI features on Claude through the anthropic provider.")
			require.Contains(t, output, "The source passed the oracle on attempt 1 of 3.\n")
			require.Contains(t, output, "Tokens: 700 input, 150 output\n")
			require.Equal(t, []string{"/v1/chat/completions"}, gateway.paths)
			require.Equal(t, "llama3.1", gateway.bodies[0]["model"])
			require.Equal(t, []string{""}, gateway.authorizations)
		})

		t.Run("emod ai reads its settings from ~/.config/emod/config and the key from ANTHROPIC_API_KEY", func(t *testing.T) {
			home := withoutAISettings(t)
			gateway := newAPIGateway(t, test.BillingPayments)
			t.Setenv("ANTHROPIC_API_KEY", "anthropic-api-key")
			writeAIConfig(t, home, "EMOD_AI_PROVIDER=anthropic\nEMOD_AI_ENDPOINT="+gateway.server.URL+"\n")

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai"))
			})

			requireSetting(t, output, "Provider", "anthropic")
			requireSetting(t, output, "API key", "from ANTHROPIC_API_KEY")
			require.Contains(t, output, "The source passed the oracle on attempt 1 of 3.\n")
			require.Equal(t, []string{"anthropic-api-key"}, gateway.apiKeys)
		})

		t.Run("names the defaults and the Anthropic API when no gateway is set", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_API_KEY", "test-api-key")
			loadConfig := func() (config.Config, error) { return config.Load(config.Flags{}) }

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunAI(context.Background(), loadConfig, connectTo(llm.NewFake(usedAnswer(test.BillingPayments, 1, 1)))))
			})

			requireSetting(t, output, "Provider", "anthropic")
			requireSetting(t, output, "Model", "claude-opus-4-8")
			requireSetting(t, output, "Cheap model", "claude-haiku-4-5")
			requireSetting(t, output, "Effort", "medium")
			requireSetting(t, output, "Endpoint", "the Anthropic API")
			requireSetting(t, output, "API key", "from EMOD_AI_API_KEY")
		})
	})

	t.Run("status", func(t *testing.T) {
		t.Run("emod ai status prints how to turn AI on when AI is not configured", func(t *testing.T) {
			withoutAISettings(t)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai", "status"))
			})

			require.Contains(t, output, "AI is not configured")
		})

		t.Run("emod ai status names the variable that holds the API key, hides the key and calls no model", func(t *testing.T) {
			withoutAISettings(t)
			gateway := newAPIGateway(t, test.BillingPayments)
			t.Setenv("EMOD_AI_API_KEY", "secret-api-key")

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai", "status", "--ai-endpoint", gateway.server.URL))
			})

			requireSetting(t, output, "API key", "from EMOD_AI_API_KEY")
			require.NotContains(t, output, "secret-api-key")
			require.Empty(t, gateway.bodies)
		})

		t.Run("emod ai status names ANTHROPIC_API_KEY when it gives the key", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_PROVIDER", "anthropic")
			t.Setenv("ANTHROPIC_API_KEY", "anthropic-api-key")

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai", "status"))
			})

			requireSetting(t, output, "API key", "from ANTHROPIC_API_KEY")
		})

		t.Run("emod ai status names OPENAI_API_KEY and the OpenAI API for the openai provider", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_PROVIDER", "openai")
			t.Setenv("EMOD_AI_MODEL", "gpt-test")
			t.Setenv("OPENAI_API_KEY", "openai-api-key")

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai", "status"))
			})

			requireSetting(t, output, "Endpoint", "the OpenAI API")
			requireSetting(t, output, "API key", "from OPENAI_API_KEY")
			require.NotContains(t, output, "openai-api-key")
		})

		t.Run("emod ai status exits 1 and names the key variables when AI is on but no key is set", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_PROVIDER", "anthropic")

			var err error
			stderr := captureStderr(t, func() {
				captureStdout(t, func() {
					err = runCommandLine(t, "emod", "ai", "status")
				})
			})

			requireExitCode(t, 1, err)
			require.Contains(t, stderr, "an API key is required: set EMOD_AI_API_KEY or ANTHROPIC_API_KEY in the environment")
		})
	})

	t.Run("round trip", func(t *testing.T) {
		configured := config.Config{Provider: "anthropic", Model: "claude-opus-4-8", CheapModel: "claude-haiku-4-5", Effort: llm.EffortMedium, APIKey: "test-api-key", APIKeySource: "EMOD_AI_API_KEY"}

		t.Run("reports an answer that passes the oracle with its tokens", func(t *testing.T) {
			model := llm.NewFake(usedAnswer(test.BillingPayments, 1200, 300))

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunAI(context.Background(), configLoader(configured, nil), connectTo(model)))
			})

			require.Contains(t, output, "The source passed the oracle on attempt 1 of 3.\n")
			require.Contains(t, output, "Tokens: 1200 input, 300 output\n")
		})

		t.Run("reports a repaired answer with the tokens of every attempt", func(t *testing.T) {
			model := llm.NewFake(usedAnswer(test.Unparseable, 1000, 200), usedAnswer(test.BillingPayments, 500, 100))

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunAI(context.Background(), configLoader(configured, nil), connectTo(model)))
			})

			require.Contains(t, output, "The source passed the oracle on attempt 2 of 3.\n")
			require.Contains(t, output, "Tokens: 1500 input, 300 output\n")
		})

		t.Run("emod ai asks again with no blank turn when the model answers with no source", func(t *testing.T) {
			withoutAISettings(t)
			gateway := newAPIGateway(t, "", test.BillingPayments)
			t.Setenv("EMOD_AI_API_KEY", "test-api-key")
			t.Setenv("EMOD_AI_ENDPOINT", gateway.server.URL)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai"))
			})

			require.Contains(t, output, "The source passed the oracle on attempt 2 of 3.\n")
			require.Len(t, gateway.bodies, 2)
			retry := wireMessages(t, gateway.bodies[1])
			require.Len(t, retry, 3)
			for _, message := range retry {
				require.NotEmpty(t, strings.TrimSpace(message.Content), "the API rejects a blank %s turn", message.Role)
			}
		})

		t.Run("reports an answer that did not converge with its last diagnostics and exits 1", func(t *testing.T) {
			invalid := strings.Replace(test.BillingPayments, "TakePayment -> PaymentTaken", "TakePayment -> PaymentRefunded", 1)
			model := llm.NewFake(usedAnswer(test.Unparseable, 1, 1), usedAnswer(test.Unparseable, 1, 1), usedAnswer(invalid, 1, 1))

			var err error
			output := captureStdout(t, func() {
				err = cli.RunAI(context.Background(), configLoader(configured, nil), connectTo(model))
			})

			require.Contains(t, output, "The source did not pass the oracle in 3 attempts. The last answer has these diagnostics:\n")
			diagnostics := oracle.Check(invalid, "generated.emod")
			require.Len(t, diagnostics, 2)
			for _, diagnostic := range diagnostics {
				require.Contains(t, output, "  "+diagnostic.String()+"\n")
			}
			require.Contains(t, output, "Tokens: 3 input, 3 output\n")
			var lintErr *cli.LintError
			require.ErrorAs(t, err, &lintErr)
			require.Equal(t, 1, lintErr.ExitCode)
			require.Contains(t, lintErr.Message, "did not converge")
		})

		t.Run("emod ai teaches the model with an example that passes the oracle", func(t *testing.T) {
			withoutAISettings(t)
			gateway := newAPIGateway(t, test.BillingPayments)
			t.Setenv("EMOD_AI_API_KEY", "test-api-key")
			t.Setenv("EMOD_AI_ENDPOINT", gateway.server.URL)

			captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai"))
			})

			require.Len(t, gateway.bodies, 1)
			system := wireText(t, gateway.bodies[0]["system"])
			example := regexp.MustCompile("(?s)```hcl\n(.*?)```").FindStringSubmatch(system)
			require.Len(t, example, 2, "the system prompt must hold one fenced emod example")
			require.Empty(t, oracle.Check(example[1], "example.emod"))
		})

		t.Run("asks the model for a small model of its own", func(t *testing.T) {
			model := llm.NewFake(usedAnswer(test.BillingPayments, 1, 1))

			captureStdout(t, func() {
				require.NoError(t, cli.RunAI(context.Background(), configLoader(configured, nil), connectTo(model)))
			})

			request := model.Requests()[0]
			exampleNames := regexp.MustCompile(`(?:command|event) "(\w+)"`).FindAllStringSubmatch(request.System, -1)
			require.NotEmpty(t, exampleNames, "the system prompt must hold an example with commands and events")
			require.Len(t, request.Messages, 1)
			require.Equal(t, "user", request.Messages[0].Role)
			require.NotEmpty(t, strings.TrimSpace(request.Messages[0].Content))
			for _, name := range exampleNames {
				require.NotContains(t, request.Messages[0].Content, name[1], "the prompt must not ask for the example again")
			}
		})

		t.Run("reports the tokens spent before a model call fails", func(t *testing.T) {
			model := llm.NewFake(usedAnswer(test.Unparseable, 1000, 200))

			var err error
			output := captureStdout(t, func() {
				err = cli.RunAI(context.Background(), configLoader(configured, nil), connectTo(model))
			})

			require.ErrorIs(t, err, llm.ErrNoMoreResponses)
			require.Contains(t, output, "Tokens: 1000 input, 200 output\n")
		})

		t.Run("returns the error of a connection that fails", func(t *testing.T) {
			unreachable := errors.New("the provider is unreachable")
			connect := func(context.Context, config.Config) (llm.Model, error) { return nil, unreachable }

			var err error
			captureStdout(t, func() {
				err = cli.RunAI(context.Background(), configLoader(configured, nil), connect)
			})

			require.ErrorIs(t, err, unreachable)
		})

		t.Run("emod ai repairs an answer through the gateway and reports the run", func(t *testing.T) {
			withoutAISettings(t)
			gateway := newAPIGateway(t, test.Unparseable, test.BillingPayments)
			t.Setenv("EMOD_AI_API_KEY", "test-api-key")
			t.Setenv("EMOD_AI_ENDPOINT", gateway.server.URL)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai"))
			})

			require.Contains(t, output, "The source passed the oracle on attempt 2 of 3.\n")
			require.Contains(t, output, "Tokens: 2000 input, 400 output\n")
			require.Len(t, gateway.bodies, 2)
			retry := wireMessages(t, gateway.bodies[1])
			require.Len(t, retry, 3)
			require.Equal(t, []string{"user", "assistant", "user"}, []string{retry[0].Role, retry[1].Role, retry[2].Role})
			require.Equal(t, test.Unparseable, retry[1].Content)
			diagnostics := oracle.Check(test.Unparseable, "generated.emod")
			require.NotEmpty(t, diagnostics)
			for _, diagnostic := range diagnostics {
				require.Contains(t, retry[2].Content, diagnostic.String())
			}
		})
	})
}
