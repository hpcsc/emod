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
	keyRegion     = "EMOD_AI_REGION"
	keyModel      = "EMOD_AI_MODEL"
	keyCheapModel = "EMOD_AI_MODEL_CHEAP"
	keyEffort     = "EMOD_AI_EFFORT"
	keyEndpoint   = "EMOD_AI_ENDPOINT"

	defaultModel      = "anthropic.claude-opus-4-8"
	defaultCheapModel = "anthropic.claude-haiku-4-5"
)

var ErrNotConfigured = errors.New("AI is not configured")

type Config struct {
	Region     string
	Model      string
	CheapModel string
	Effort     llm.Effort
	Endpoint   string
}

// Flags carries string overrides supplied on the command line. An empty string
// means the flag was not provided and the next source in precedence is used.
type Flags struct {
	Region     string
	Model      string
	CheapModel string
	Effort     string
	Endpoint   string
}

// Resolve merges sources in order default -> file -> env -> flags so the net
// precedence is flags > env > file > built-in default.
func Resolve(flags Flags, getenv func(string) string, filePath string) (Config, error) {
	fileValues, err := readFile(filePath)
	if err != nil {
		return Config{}, err
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

	region := pick(flags.Region, keyRegion)
	model := pick(flags.Model, keyModel)
	cheapModel := pick(flags.CheapModel, keyCheapModel)
	effortLevel := pick(flags.Effort, keyEffort)
	endpoint := pick(flags.Endpoint, keyEndpoint)

	if region+model+cheapModel+effortLevel+endpoint == "" {
		return Config{}, ErrNotConfigured
	}

	if region == "" {
		return Config{}, fmt.Errorf("%s is required: set it via the --ai-region flag, the %s environment variable, or the ~/.config/emod/config file", keyRegion, keyRegion)
	}

	effort, err := parseEffort(effortLevel)
	if err != nil {
		return Config{}, err
	}

	if model == "" {
		model = defaultModel
	}

	if cheapModel == "" {
		cheapModel = defaultCheapModel
	}

	return Config{
		Region:     region,
		Model:      model,
		CheapModel: cheapModel,
		Effort:     effort,
		Endpoint:   endpoint,
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
