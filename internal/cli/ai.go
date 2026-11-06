package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/anthropic"
	"github.com/hpcsc/emod/internal/llm/config"
	"github.com/hpcsc/emod/internal/llm/repair"
)

const aiSetup = `AI is not configured, so emod runs without it.
To turn it on, set EMOD_AI_API_KEY to your Anthropic API key in the environment.
If ANTHROPIC_API_KEY already holds the key, set EMOD_AI_PROVIDER=anthropic instead:
in the environment, as a line in ~/.config/emod/config, or with --ai-provider.
EMOD_AI_MODEL, EMOD_AI_MODEL_CHEAP, EMOD_AI_EFFORT and EMOD_AI_ENDPOINT are optional.`

const smokeAttempts = 3

const smokeSystem = "You write event models in the emod language. Answer with one emod source and nothing else.\n" +
	"This is a valid emod source:\n\n```hcl\n" + `emod = 1

model "Billing" {
}

context "Payments" {
  aggregate "Payment" {
    slice "Take Payment" {
      command "TakePayment" {
        fields {
          amount = required(int)
        }
      }

      event "PaymentTaken" {
        fields {
          amount = required(int)
        }
      }

      flow = <<-FLOW
        command -> event:    TakePayment -> PaymentTaken
      FLOW
    }
  }
}
` + "```\n"

const smokePrompt = `Write an emod model "Library" with one context "Lending" and one aggregate "Loan".
The aggregate has one slice "Borrow Book": the command BorrowBook produces the event BookBorrowed.
Both have the required string field bookId.`

func RunAI(ctx context.Context, loadConfig func() (config.Config, error), connect func(context.Context, config.Config) (llm.Model, error)) error {
	cfg, err := loadConfig()
	if errors.Is(err, config.ErrNotConfigured) {
		fmt.Println(aiSetup)
		return nil
	}
	if err != nil {
		return err
	}

	printSettings(cfg)

	model, err := connect(ctx, cfg)
	if err != nil {
		return err
	}

	outcome, err := repair.Generate(ctx, model, llm.GenerateRequest{
		System:   smokeSystem,
		Messages: []llm.Message{{Role: "user", Content: smokePrompt}},
	}, smokeAttempts)
	if err != nil {
		printTokens(outcome.Usage)
		return err
	}

	if outcome.Converged {
		fmt.Printf("The source passed the oracle on attempt %d of %d.\n", outcome.Attempts, smokeAttempts)
	} else {
		fmt.Printf("The source did not pass the oracle in %d attempts. The last answer has these diagnostics:\n", outcome.Attempts)
		for _, entry := range outcome.Diagnostics {
			fmt.Printf("  %s\n", entry)
		}
	}

	printTokens(outcome.Usage)

	if !outcome.Converged {
		return &LintError{Message: "AI did not converge: no answer passed the oracle", ExitCode: 1}
	}
	return nil
}

func RunAIStatus(loadConfig func() (config.Config, error)) error {
	cfg, err := loadConfig()
	if errors.Is(err, config.ErrNotConfigured) {
		fmt.Println(aiSetup)
		return nil
	}
	if err != nil {
		return err
	}

	printSettings(cfg)
	return nil
}

func printSettings(cfg config.Config) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "the Anthropic API"
	}
	fmt.Printf("AI is configured.\n  Provider:    %s\n  Model:       %s\n  Cheap model: %s\n  Effort:      %s\n  Endpoint:    %s\n  API key:     from %s\n",
		cfg.Provider, cfg.Model, cfg.CheapModel, cfg.Effort, endpoint, cfg.APIKeySource)
}

func printTokens(usage llm.Usage) {
	fmt.Printf("Tokens: %d input, %d output\n", usage.InputTokens, usage.OutputTokens)
}

func connectAnthropic(_ context.Context, cfg config.Config) (llm.Model, error) {
	adapter, err := anthropic.New(anthropic.Settings{
		APIKey:        cfg.APIKey,
		DefaultModel:  cfg.Model,
		CheapModel:    cfg.CheapModel,
		DefaultEffort: cfg.Effort,
		Endpoint:      cfg.Endpoint,
	})
	if err != nil {
		return nil, err
	}
	return adapter, nil
}
