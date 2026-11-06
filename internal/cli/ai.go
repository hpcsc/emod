package cli

import (
	"errors"
	"fmt"

	"github.com/hpcsc/emod/internal/llm/config"
)

const aiSetup = `AI is not configured, so emod runs without it.
To turn it on, set EMOD_AI_REGION to the AWS region of your Bedrock account:
in the environment, as the line EMOD_AI_REGION=<region> in ~/.config/emod/config, or with --ai-region.
EMOD_AI_MODEL, EMOD_AI_MODEL_CHEAP, EMOD_AI_EFFORT and EMOD_AI_ENDPOINT are optional.`

func RunAI(settings func() (config.Config, error)) error {
	cfg, err := settings()
	if errors.Is(err, config.ErrNotConfigured) {
		fmt.Println(aiSetup)
		return nil
	}
	if err != nil {
		return err
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "the Bedrock endpoint of the region"
	}

	fmt.Printf("AI is configured.\n  Region:      %s\n  Model:       %s\n  Cheap model: %s\n  Effort:      %s\n  Endpoint:    %s\n",
		cfg.Region, cfg.Model, cfg.CheapModel, cfg.Effort, endpoint)
	return nil
}
