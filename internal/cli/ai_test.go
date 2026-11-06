//go:build unit

package cli_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/cli"
	"github.com/hpcsc/emod/internal/llm/config"
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

func TestAI(t *testing.T) {
	t.Run("not configured", func(t *testing.T) {
		t.Run("prints how to turn AI on and succeeds", func(t *testing.T) {
			settings := func() (config.Config, error) { return config.Config{}, config.ErrNotConfigured }

			output := captureStdout(t, func() {
				require.NoError(t, cli.RunAI(settings))
			})

			require.Contains(t, output, "AI is not configured")
			require.Contains(t, output, "EMOD_AI_REGION")
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

		t.Run("returns the configuration error", func(t *testing.T) {
			invalid := errors.New(`EMOD_AI_EFFORT has an invalid value "highest"`)
			settings := func() (config.Config, error) { return config.Config{}, invalid }

			err := cli.RunAI(settings)

			require.ErrorIs(t, err, invalid)
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

	t.Run("configured", func(t *testing.T) {
		t.Run("emod ai reports each setting that its flags give", func(t *testing.T) {
			withoutAISettings(t)

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai",
					"--ai-region", "eu-west-1",
					"--ai-model", "anthropic.claude-opus-5-5",
					"--ai-model-cheap", "anthropic.claude-sonnet-5-5",
					"--ai-effort", "high",
					"--ai-endpoint", "https://gateway.example",
				))
			})

			require.Contains(t, output, "AI is configured.")
			require.Contains(t, output, "Region:      eu-west-1\n")
			require.Contains(t, output, "Model:       anthropic.claude-opus-5-5\n")
			require.Contains(t, output, "Cheap model: anthropic.claude-sonnet-5-5\n")
			require.Contains(t, output, "Effort:      high\n")
			require.Contains(t, output, "Endpoint:    https://gateway.example\n")
		})

		t.Run("emod ai names the defaults and the Bedrock endpoint when only the region is set", func(t *testing.T) {
			withoutAISettings(t)
			t.Setenv("EMOD_AI_REGION", "us-east-1")

			output := captureStdout(t, func() {
				require.NoError(t, runCommandLine(t, "emod", "ai"))
			})

			require.Contains(t, output, "Model:       anthropic.claude-opus-4-8\n")
			require.Contains(t, output, "Cheap model: anthropic.claude-haiku-4-5\n")
			require.Contains(t, output, "Effort:      medium\n")
			require.Contains(t, output, "Endpoint:    the Bedrock endpoint of the region\n")
		})
	})
}
