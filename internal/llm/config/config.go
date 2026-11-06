package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hpcsc/emod/internal/llm"
)

const (
	keyProvider   = "EMOD_AI_PROVIDER"
	keyAPIKey     = "EMOD_AI_API_KEY"
	keyModel      = "EMOD_AI_MODEL"
	keyCheapModel = "EMOD_AI_MODEL_CHEAP"
	keyEffort     = "EMOD_AI_EFFORT"
	keyEndpoint   = "EMOD_AI_ENDPOINT"

	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"

	defaultAnthropicModel      = "claude-opus-4-8"
	defaultAnthropicCheapModel = "claude-haiku-4-5"
)

var apiKeyVariables = map[string]string{
	ProviderAnthropic: "ANTHROPIC_API_KEY",
	ProviderOpenAI:    "OPENAI_API_KEY",
}

var ErrNotConfigured = errors.New("AI is not configured")

type Config struct {
	Provider     string
	Model        string
	CheapModel   string
	Effort       llm.Effort
	Endpoint     string
	APIKey       string
	APIKeySource string
}

// Flags carries string overrides supplied on the command line. An empty string
// means the flag was not provided and the next source in precedence is used.
type Flags struct {
	Provider   string
	Model      string
	CheapModel string
	Effort     string
	Endpoint   string
}

// Resolve merges sources in order default -> file -> env -> flags so the net
// precedence is flags > env > file > built-in default. The API key comes only
// from the environment.
func Resolve(flags Flags, getenv func(string) string, filePath string) (Config, error) {
	fileValues, err := readFile(filePath)
	if err != nil {
		return Config{}, err
	}
	if _, ok := fileValues[keyAPIKey]; ok {
		return Config{}, fmt.Errorf("config file %s holds %s: remove the line, and set the key in the environment", filePath, keyAPIKey)
	}

	pick := func(flag, key string) string {
		if flag != "" {
			return flag
		}
		if v := getenv(key); v != "" {
			return v
		}
		return fileValues[key]
	}

	provider := pick(flags.Provider, keyProvider)
	model := pick(flags.Model, keyModel)
	cheapModel := pick(flags.CheapModel, keyCheapModel)
	effortLevel := pick(flags.Effort, keyEffort)
	endpoint := pick(flags.Endpoint, keyEndpoint)
	apiKey := getenv(keyAPIKey)

	if provider+model+cheapModel+effortLevel+endpoint+apiKey == "" {
		return Config{}, ErrNotConfigured
	}

	if provider == "" {
		provider = ProviderAnthropic
	}
	providerAPIKey, known := apiKeyVariables[provider]
	if !known {
		return Config{}, fmt.Errorf("%s has an invalid value %q: accepted providers are %s and %s", keyProvider, provider, ProviderAnthropic, ProviderOpenAI)
	}

	apiKeySource := keyAPIKey
	if apiKey == "" {
		apiKey, apiKeySource = getenv(providerAPIKey), providerAPIKey
	}
	if apiKey == "" {
		apiKeySource = ""
	}
	if apiKey == "" && provider == ProviderAnthropic {
		return Config{}, fmt.Errorf("an API key is required: set %s or %s in the environment", keyAPIKey, providerAPIKey)
	}

	effort, err := parseEffort(effortLevel)
	if err != nil {
		return Config{}, err
	}

	switch provider {
	case ProviderAnthropic:
		if model == "" {
			model = defaultAnthropicModel
		}
		if cheapModel == "" {
			cheapModel = defaultAnthropicCheapModel
		}
	case ProviderOpenAI:
		if model == "" {
			return Config{}, fmt.Errorf("%s is required for the %s provider: set it with the --ai-model flag, the %s environment variable, or the ~/.config/emod/config file", keyModel, ProviderOpenAI, keyModel)
		}
		if cheapModel == "" {
			cheapModel = model
		}
	}

	return Config{
		Provider:     provider,
		Model:        model,
		CheapModel:   cheapModel,
		Effort:       effort,
		Endpoint:     endpoint,
		APIKey:       apiKey,
		APIKeySource: apiKeySource,
	}, nil
}

func Load(flags Flags) (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("determining home directory for ~/.config/emod/config: %w", err)
	}
	return Resolve(flags, os.Getenv, filepath.Join(home, ".config", "emod", "config"))
}

func parseEffort(value string) (llm.Effort, error) {
	switch value {
	case "":
		return llm.EffortMedium, nil
	case "low":
		return llm.EffortLow, nil
	case "medium":
		return llm.EffortMedium, nil
	case "high":
		return llm.EffortHigh, nil
	case "xhigh":
		return llm.EffortXHigh, nil
	default:
		return llm.EffortUnset, fmt.Errorf("%s has an invalid value %q: accepted levels are low, medium, high, xhigh", keyEffort, value)
	}
}

func readFile(filePath string) (map[string]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", filePath, err)
	}
	defer f.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(raw, "=")
		if !ok {
			return nil, fmt.Errorf("config file %s could not be parsed: line %d %q is missing a '=' separator", filePath, lineNo, raw)
		}
		values[strings.TrimSpace(key)] = strings.TrimRight(value, " \t\r\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", filePath, err)
	}

	return values, nil
}
