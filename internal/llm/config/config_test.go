//go:build unit

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/config"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	mapEnv := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	writeFile := func(t *testing.T, contents string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "emod")
		require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
		return path
	}

	missingFile := func(t *testing.T) string {
		t.Helper()
		return filepath.Join(t.TempDir(), "does-not-exist")
	}

	t.Run("precedence", func(t *testing.T) {
		t.Run("flag overrides env, file, and default for the same key", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_MODEL=from-file\n")
			getenv := mapEnv(map[string]string{"EMOD_AI_MODEL": "from-env", "EMOD_AI_API_KEY": "test-key"})

			cfg, err := config.Resolve(config.Flags{Model: "from-flag"}, getenv, path)

			require.NoError(t, err)
			require.Equal(t, "from-flag", cfg.Model)
		})

		t.Run("env overrides file and default when no flag is given", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_MODEL=from-file\n")
			getenv := mapEnv(map[string]string{"EMOD_AI_MODEL": "from-env", "EMOD_AI_API_KEY": "test-key"})

			cfg, err := config.Resolve(config.Flags{}, getenv, path)

			require.NoError(t, err)
			require.Equal(t, "from-env", cfg.Model)
		})

		t.Run("file overrides built-in default when neither flag nor env is given", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_MODEL=file-opus\nEMOD_AI_MODEL_CHEAP=file-haiku\nEMOD_AI_EFFORT=high\n")
			getenv := mapEnv(map[string]string{"EMOD_AI_API_KEY": "test-key"})

			cfg, err := config.Resolve(config.Flags{}, getenv, path)

			require.NoError(t, err)
			require.Equal(t, config.Config{
				Provider:     "anthropic",
				Model:        "file-opus",
				CheapModel:   "file-haiku",
				Effort:       llm.EffortHigh,
				Endpoint:     "",
				APIKey:       "test-key",
				APIKeySource: "EMOD_AI_API_KEY",
			}, cfg)
		})
	})

	t.Run("recognises keys", func(t *testing.T) {
		t.Run("reads every documented key from the environment", func(t *testing.T) {
			getenv := mapEnv(map[string]string{
				"EMOD_AI_PROVIDER":    "anthropic",
				"EMOD_AI_API_KEY":     "env-key",
				"EMOD_AI_MODEL":       "env-opus",
				"EMOD_AI_MODEL_CHEAP": "env-haiku",
				"EMOD_AI_EFFORT":      "xhigh",
				"EMOD_AI_ENDPOINT":    "https://env.example",
			})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, config.Config{
				Provider:     "anthropic",
				Model:        "env-opus",
				CheapModel:   "env-haiku",
				Effort:       llm.EffortXHigh,
				Endpoint:     "https://env.example",
				APIKey:       "env-key",
				APIKeySource: "EMOD_AI_API_KEY",
			}, cfg)
		})

		t.Run("reads every documented key except the API key from the file", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_PROVIDER=anthropic\nEMOD_AI_MODEL=file-opus\nEMOD_AI_MODEL_CHEAP=file-haiku\nEMOD_AI_EFFORT=low\nEMOD_AI_ENDPOINT=https://file.example\n")
			getenv := mapEnv(map[string]string{"EMOD_AI_API_KEY": "env-key"})

			cfg, err := config.Resolve(config.Flags{}, getenv, path)

			require.NoError(t, err)
			require.Equal(t, config.Config{
				Provider:     "anthropic",
				Model:        "file-opus",
				CheapModel:   "file-haiku",
				Effort:       llm.EffortLow,
				Endpoint:     "https://file.example",
				APIKey:       "env-key",
				APIKeySource: "EMOD_AI_API_KEY",
			}, cfg)
		})
	})

	t.Run("opt-in", func(t *testing.T) {
		t.Run("reports that AI is not configured when no source sets any AI setting", func(t *testing.T) {
			cfg, err := config.Resolve(config.Flags{}, mapEnv(map[string]string{}), missingFile(t))

			require.ErrorIs(t, err, config.ErrNotConfigured)
			require.Equal(t, config.Config{}, cfg)
		})

		t.Run("reports that AI is not configured when the file sets nothing", func(t *testing.T) {
			path := writeFile(t, "# AI settings\n\n")

			_, err := config.Resolve(config.Flags{}, mapEnv(map[string]string{}), path)

			require.ErrorIs(t, err, config.ErrNotConfigured)
		})

		t.Run("ANTHROPIC_API_KEY alone does not turn AI on", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"ANTHROPIC_API_KEY": "key-for-another-tool"})

			_, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.ErrorIs(t, err, config.ErrNotConfigured)
		})

		t.Run("any one AI setting from any source turns AI on and then asks for the API key", func(t *testing.T) {
			cases := []struct {
				name  string
				flags config.Flags
				env   map[string]string
				file  string
			}{
				{name: "the provider flag", flags: config.Flags{Provider: "anthropic"}},
				{name: "the model flag", flags: config.Flags{Model: "flag-opus"}},
				{name: "the cheap model flag", flags: config.Flags{CheapModel: "flag-haiku"}},
				{name: "the effort flag", flags: config.Flags{Effort: "high"}},
				{name: "the endpoint flag", flags: config.Flags{Endpoint: "https://flag.example"}},
				{name: "the provider variable", env: map[string]string{"EMOD_AI_PROVIDER": "anthropic"}},
				{name: "the model variable", env: map[string]string{"EMOD_AI_MODEL": "env-opus"}},
				{name: "the cheap model variable", env: map[string]string{"EMOD_AI_MODEL_CHEAP": "env-haiku"}},
				{name: "the effort variable", env: map[string]string{"EMOD_AI_EFFORT": "low"}},
				{name: "the endpoint variable", env: map[string]string{"EMOD_AI_ENDPOINT": "https://env.example"}},
				{name: "the provider in the file", file: "EMOD_AI_PROVIDER=anthropic\n"},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					path := missingFile(t)
					if tc.file != "" {
						path = writeFile(t, tc.file)
					}

					_, err := config.Resolve(tc.flags, mapEnv(tc.env), path)

					require.Error(t, err)
					require.NotErrorIs(t, err, config.ErrNotConfigured)
					require.ErrorContains(t, err, "an API key is required: set EMOD_AI_API_KEY or ANTHROPIC_API_KEY in the environment")
				})
			}
		})
	})

	t.Run("API key", func(t *testing.T) {
		t.Run("takes the key from EMOD_AI_API_KEY before ANTHROPIC_API_KEY", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_API_KEY": "emod-key", "ANTHROPIC_API_KEY": "anthropic-key"})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "emod-key", cfg.APIKey)
			require.Equal(t, "EMOD_AI_API_KEY", cfg.APIKeySource)
		})

		t.Run("takes the key from ANTHROPIC_API_KEY when another setting turns AI on", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_PROVIDER": "anthropic", "ANTHROPIC_API_KEY": "anthropic-key"})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "anthropic-key", cfg.APIKey)
			require.Equal(t, "ANTHROPIC_API_KEY", cfg.APIKeySource)
		})

		t.Run("refuses a config file that holds the key, and does not repeat the key", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_API_KEY=secret-in-the-file\n")

			_, err := config.Resolve(config.Flags{}, mapEnv(map[string]string{}), path)

			require.ErrorContains(t, err, "config file "+path+" holds EMOD_AI_API_KEY")
			require.NotContains(t, err.Error(), "secret-in-the-file")
		})
	})

	t.Run("provider validation", func(t *testing.T) {
		t.Run("rejects a provider other than anthropic and names the accepted provider", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_PROVIDER": "openai", "EMOD_AI_API_KEY": "test-key"})

			_, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.ErrorContains(t, err, `EMOD_AI_PROVIDER has an invalid value "openai": the accepted provider is anthropic`)
		})
	})

	t.Run("effort validation", func(t *testing.T) {
		t.Run("rejects an unaccepted effort naming the key and accepted levels", func(t *testing.T) {
			getenv := mapEnv(map[string]string{
				"EMOD_AI_API_KEY": "test-key",
				"EMOD_AI_EFFORT":  "ultra",
			})

			_, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.Error(t, err)
			msg := err.Error()
			require.Contains(t, msg, "EMOD_AI_EFFORT")
			require.Contains(t, msg, "low")
			require.Contains(t, msg, "medium")
			require.Contains(t, msg, "high")
			require.Contains(t, msg, "xhigh")
		})
	})

	t.Run("defaults", func(t *testing.T) {
		t.Run("applies the provider, the default models and medium effort when only the API key is supplied", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_API_KEY": "test-key"})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, config.Config{
				Provider:     "anthropic",
				Model:        "claude-opus-4-8",
				CheapModel:   "claude-haiku-4-5",
				Effort:       llm.EffortMedium,
				Endpoint:     "",
				APIKey:       "test-key",
				APIKeySource: "EMOD_AI_API_KEY",
			}, cfg)
		})
	})

	t.Run("endpoint", func(t *testing.T) {
		t.Run("carries the endpoint through when set", func(t *testing.T) {
			getenv := mapEnv(map[string]string{
				"EMOD_AI_API_KEY":  "test-key",
				"EMOD_AI_ENDPOINT": "https://gateway.internal",
			})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "https://gateway.internal", cfg.Endpoint)
		})

		t.Run("leaves the endpoint empty when no source provides it", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_API_KEY": "test-key"})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "", cfg.Endpoint)
		})
	})

	t.Run("file", func(t *testing.T) {
		t.Run("treats a missing file as no file values", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_API_KEY": "test-key"})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "claude-opus-4-8", cfg.Model)
		})

		t.Run("returns a clear error when the file is malformed", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_MODEL=file-opus\nthis line has no equals sign\n")
			getenv := mapEnv(map[string]string{})

			_, err := config.Resolve(config.Flags{}, getenv, path)

			require.Error(t, err)
			require.Contains(t, err.Error(), path)
		})
	})
}

func TestLoad(t *testing.T) {
	t.Run("home", func(t *testing.T) {
		t.Run("reports that the config file cannot be found without a home directory", func(t *testing.T) {
			t.Setenv("HOME", "")

			_, err := config.Load(config.Flags{Model: "claude-opus-4-8"})

			require.ErrorContains(t, err, "determining home directory for ~/.config/emod/config")
		})
	})
}
