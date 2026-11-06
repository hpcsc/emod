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
	for _, key := range []string{"EMOD_AI_REGION", "EMOD_AI_MODEL", "EMOD_AI_MODEL_CHEAP", "EMOD_AI_EFFORT", "EMOD_AI_ENDPOINT"} {
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

type bedrockGateway struct {
	server *httptest.Server

	mu             sync.Mutex
	answers        []string
	bodies         []map[string]any
	authorizations []string
}

func newBedrockGateway(t *testing.T, answers ...string) *bedrockGateway {
	t.Helper()

	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_PROFILE", "")

	gateway := &bedrockGateway{answers: answers}
	gateway.server = httptest.NewServer(http.HandlerFunc(gateway.answer))
	t.Cleanup(gateway.server.Close)

	return gateway
}

func (g *bedrockGateway) answer(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()

	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	g.bodies = append(g.bodies, body)
	g.authorizations = append(g.authorizations, r.Header.Get("Authorization"))

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
			require.Contains(t, output, "EMOD_AI_REGION")
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

		t.Run("emod ai with an AI setting but no region exits 1 and names the region", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_MODEL", "anthropic.claude-opus-4-8")

			var err error
			stderr := captureStderr(t, func() {
				err = runCommandLine(t, "emod", "ai")
			})

			requireExitCode(t, 1, err)
			require.Contains(t, stderr, "EMOD_AI_REGION is required")
		})

		t.Run("emod ai with an effort flag that is not a level exits 1 and names the effort", func(t *testing.T) {
			withoutAISettings(t)

			var err error
			stderr := captureStderr(t, func() {
				err = runCommandLine(t, "emod", "ai", "--ai-region", "eu-west-1", "--ai-effort", "highest")
			})

			requireExitCode(t, 1, err)
			require.Contains(t, stderr, `EMOD_AI_EFFORT has an invalid value "highest"`)
		})
	})

	t.Run("settings", func(t *testing.T) {
		t.Run("emod ai reports each setting that its flags give and sends the call through the gateway", func(t *testing.T) {
			withoutAISettings(t)
			gateway := newBedrockGateway(t, test.BillingPayments)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai",
					"--ai-region", "eu-west-1",
					"--ai-model", "anthropic.claude-opus-5-5",
					"--ai-model-cheap", "anthropic.claude-sonnet-5-5",
					"--ai-effort", "high",
					"--ai-endpoint", gateway.server.URL,
				))
			})

			requireSetting(t, output, "Region", "eu-west-1")
			requireSetting(t, output, "Model", "anthropic.claude-opus-5-5")
			requireSetting(t, output, "Cheap model", "anthropic.claude-sonnet-5-5")
			requireSetting(t, output, "Effort", "high")
			requireSetting(t, output, "Endpoint", gateway.server.URL)
			require.Len(t, gateway.bodies, 1)
			require.Equal(t, "anthropic.claude-opus-5-5", gateway.bodies[0]["model"])
			require.Equal(t, map[string]any{"effort": "high"}, gateway.bodies[0]["output_config"])
			require.Contains(t, gateway.authorizations[0], "/eu-west-1/")
		})

		t.Run("emod ai reads its settings from ~/.config/emod/config", func(t *testing.T) {
			home := withoutAISettings(t)
			gateway := newBedrockGateway(t, test.BillingPayments)
			writeAIConfig(t, home, "EMOD_AI_REGION=ap-southeast-2\nEMOD_AI_ENDPOINT="+gateway.server.URL+"\n")

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai"))
			})

			requireSetting(t, output, "Region", "ap-southeast-2")
			require.Contains(t, output, "The source passed the oracle on attempt 1 of 3.\n")
			require.Len(t, gateway.bodies, 1)
		})

		t.Run("names the defaults and the Bedrock endpoint of the region when no gateway is set", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_REGION", "us-east-1")
			loadConfig := func() (config.Config, error) { return config.Load(config.Flags{}) }

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunAI(context.Background(), loadConfig, connectTo(llm.NewFake(usedAnswer(test.BillingPayments, 1, 1)))))
			})

			requireSetting(t, output, "Model", "anthropic.claude-opus-4-8")
			requireSetting(t, output, "Cheap model", "anthropic.claude-haiku-4-5")
			requireSetting(t, output, "Effort", "medium")
			requireSetting(t, output, "Endpoint", "the Bedrock endpoint of the region")
		})
	})

	t.Run("round trip", func(t *testing.T) {
		configured := config.Config{Region: "us-east-1", Model: "anthropic.claude-opus-4-8", CheapModel: "anthropic.claude-haiku-4-5", Effort: llm.EffortMedium}

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
			gateway := newBedrockGateway(t, "", test.BillingPayments)
			t.Setenv("EMOD_AI_REGION", "us-east-1")
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
			gateway := newBedrockGateway(t, test.BillingPayments)
			t.Setenv("EMOD_AI_REGION", "us-east-1")
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
			unreachable := errors.New("no AWS credentials")
			connect := func(context.Context, config.Config) (llm.Model, error) { return nil, unreachable }

			var err error
			captureStdout(t, func() {
				err = cli.RunAI(context.Background(), configLoader(configured, nil), connect)
			})

			require.ErrorIs(t, err, unreachable)
		})

		t.Run("emod ai exits 1 when the Bedrock connection cannot be set up", func(t *testing.T) {
			home := withoutAISettings(t)
			gateway := newBedrockGateway(t)
			t.Setenv("EMOD_AI_REGION", "us-east-1")
			t.Setenv("EMOD_AI_ENDPOINT", gateway.server.URL)
			t.Setenv("AWS_ACCESS_KEY_ID", "")
			t.Setenv("AWS_SECRET_ACCESS_KEY", "")
			t.Setenv("AWS_PROFILE", "emod-missing-profile")
			t.Setenv("AWS_CONFIG_FILE", filepath.Join(home, "aws-config"))
			t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(home, "aws-credentials"))

			var err error
			stderr := captureStderr(t, func() {
				captureStdout(t, func() {
					err = runCommandLine(t, "emod", "ai")
				})
			})

			requireExitCode(t, 1, err)
			require.Contains(t, stderr, "emod-missing-profile")
			require.Empty(t, gateway.bodies)
		})

		t.Run("emod ai repairs an answer through the gateway and reports the run", func(t *testing.T) {
			withoutAISettings(t)
			gateway := newBedrockGateway(t, test.Unparseable, test.BillingPayments)
			t.Setenv("EMOD_AI_REGION", "us-east-1")
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
