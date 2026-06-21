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
			path := writeFile(t, "EMOD_AI_REGION=from-file\n")
			getenv := mapEnv(map[string]string{"EMOD_AI_REGION": "from-env"})

			cfg, err := config.Resolve(config.Flags{Region: "from-flag"}, getenv, path)

			require.NoError(t, err)
			require.Equal(t, "from-flag", cfg.Region)
		})

		t.Run("env overrides file and default when no flag is given", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_REGION=from-file\n")
			getenv := mapEnv(map[string]string{"EMOD_AI_REGION": "from-env"})

			cfg, err := config.Resolve(config.Flags{}, getenv, path)

			require.NoError(t, err)
			require.Equal(t, "from-env", cfg.Region)
		})

		t.Run("file overrides built-in default when neither flag nor env is given", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_REGION=us-east-1\nEMOD_AI_MODEL=file-opus\nEMOD_AI_MODEL_CHEAP=file-haiku\nEMOD_AI_EFFORT=high\n")
			getenv := mapEnv(map[string]string{})

			cfg, err := config.Resolve(config.Flags{}, getenv, path)

			require.NoError(t, err)
			require.Equal(t, config.Config{
				Region:     "us-east-1",
				Model:      "file-opus",
				CheapModel: "file-haiku",
				Effort:     llm.EffortHigh,
				Endpoint:   "",
			}, cfg)
		})
	})

	t.Run("recognises keys", func(t *testing.T) {
		t.Run("reads every documented key from the environment", func(t *testing.T) {
			getenv := mapEnv(map[string]string{
				"EMOD_AI_REGION":      "eu-west-1",
				"EMOD_AI_MODEL":       "env-opus",
				"EMOD_AI_MODEL_CHEAP": "env-haiku",
				"EMOD_AI_EFFORT":      "xhigh",
				"EMOD_AI_ENDPOINT":    "https://env.example",
			})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, config.Config{
				Region:     "eu-west-1",
				Model:      "env-opus",
				CheapModel: "env-haiku",
				Effort:     llm.EffortXHigh,
				Endpoint:   "https://env.example",
			}, cfg)
		})

		t.Run("reads every documented key from the file", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_REGION=ap-southeast-2\nEMOD_AI_MODEL=file-opus\nEMOD_AI_MODEL_CHEAP=file-haiku\nEMOD_AI_EFFORT=low\nEMOD_AI_ENDPOINT=https://file.example\n")
			getenv := mapEnv(map[string]string{})

			cfg, err := config.Resolve(config.Flags{}, getenv, path)

			require.NoError(t, err)
			require.Equal(t, config.Config{
				Region:     "ap-southeast-2",
				Model:      "file-opus",
				CheapModel: "file-haiku",
				Effort:     llm.EffortLow,
				Endpoint:   "https://file.example",
			}, cfg)
		})
	})

	t.Run("required region", func(t *testing.T) {
		t.Run("returns an error naming EMOD_AI_REGION when absent from all sources", func(t *testing.T) {
			cfg, err := config.Resolve(config.Flags{}, mapEnv(map[string]string{}), missingFile(t))

			require.Error(t, err)
			require.Contains(t, err.Error(), "EMOD_AI_REGION")
			require.Equal(t, config.Config{}, cfg)
		})
	})

	t.Run("effort validation", func(t *testing.T) {
		t.Run("rejects an unaccepted effort naming the key and accepted levels", func(t *testing.T) {
			getenv := mapEnv(map[string]string{
				"EMOD_AI_REGION": "us-east-1",
				"EMOD_AI_EFFORT": "ultra",
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
		t.Run("applies default models and medium effort when only region is supplied", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_REGION": "us-east-1"})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "us-east-1", cfg.Region)
			require.Equal(t, "anthropic.claude-opus-4-8", cfg.Model)
			require.Equal(t, "anthropic.claude-haiku-4-5", cfg.CheapModel)
			require.Equal(t, "medium", cfg.Effort.String())
			require.Equal(t, "", cfg.Endpoint)
		})
	})

	t.Run("endpoint", func(t *testing.T) {
		t.Run("carries the endpoint through when set", func(t *testing.T) {
			getenv := mapEnv(map[string]string{
				"EMOD_AI_REGION":   "us-east-1",
				"EMOD_AI_ENDPOINT": "https://bedrock.internal",
			})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "https://bedrock.internal", cfg.Endpoint)
		})

		t.Run("leaves the endpoint empty when no source provides it", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_REGION": "us-east-1"})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "", cfg.Endpoint)
		})
	})

	t.Run("file", func(t *testing.T) {
		t.Run("treats a missing file as no file values", func(t *testing.T) {
			getenv := mapEnv(map[string]string{"EMOD_AI_REGION": "us-east-1"})

			cfg, err := config.Resolve(config.Flags{}, getenv, missingFile(t))

			require.NoError(t, err)
			require.Equal(t, "us-east-1", cfg.Region)
		})

		t.Run("returns a clear error when the file is malformed", func(t *testing.T) {
			path := writeFile(t, "EMOD_AI_REGION=us-east-1\nthis line has no equals sign\n")
			getenv := mapEnv(map[string]string{})

			_, err := config.Resolve(config.Flags{}, getenv, path)

			require.Error(t, err)
			require.Contains(t, err.Error(), path)
		})
	})
}
