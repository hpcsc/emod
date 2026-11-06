package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/anthropic"
	"github.com/hpcsc/emod/internal/llm/config"
	"github.com/hpcsc/emod/internal/llm/openai"
	"github.com/hpcsc/emod/internal/llm/repair"
)

const aiSetup = `AI is not configured, so emod runs without it.
To turn it on, set EMOD_AI_API_KEY to your Anthropic API key in the environment.
If ANTHROPIC_API_KEY already holds the key, set EMOD_AI_PROVIDER=anthropic instead.
For OpenAI or a server with the OpenAI-compatible API, such as Ollama, set EMOD_AI_PROVIDER=openai
and EMOD_AI_MODEL, and set EMOD_AI_ENDPOINT for a server other than OpenAI.
Set these keys in the environment, as lines in ~/.config/emod/config, or with the --ai-* flags.
The API key goes only in the environment. EMOD_AI_MODEL_CHEAP and EMOD_AI_EFFORT are optional.`

const outsideTestedTier = `Notice: emod tests its AI features on Claude through the anthropic provider. With this provider,
emod checks the structure of each answer, but it does not measure the quality of the judgement.`

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
	if cfg.Provider != config.ProviderAnthropic {
		fmt.Println(outsideTestedTier)
	}

	model, err := connect(ctx, cfg)
	if err != nil {
		return err
	}

	fmt.Println("emod asks the model for an emod source. Each attempt can take many seconds.")
	outcome, err := repair.Generate(ctx, model, llm.GenerateRequest{
		System:   smokeSystem,
		Messages: []llm.Message{{Role: "user", Content: smokePrompt}},
	}, smokeAttempts, repair.WithProgress(printAttempt))
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

func printAttempt(attempt repair.Attempt) {
	tokens := fmt.Sprintf("Tokens: %d input, %d output.", attempt.Usage.InputTokens, attempt.Usage.OutputTokens)
	count := len(attempt.Diagnostics)
	switch {
	case count == 0:
		fmt.Printf("Attempt %d of %d: the source passes the oracle. %s\n", attempt.Number, attempt.MaxAttempts, tokens)
	case attempt.Number < attempt.MaxAttempts:
		fmt.Printf("Attempt %d of %d: the source has %s. emod asks the model again. %s\n", attempt.Number, attempt.MaxAttempts, diagnosticCount(count), tokens)
	default:
		fmt.Printf("Attempt %d of %d: the source has %s. %s\n", attempt.Number, attempt.MaxAttempts, diagnosticCount(count), tokens)
	}
}

func diagnosticCount(count int) string {
	if count == 1 {
		return "1 diagnostic"
	}
	return fmt.Sprintf("%d diagnostics", count)
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
		if cfg.Provider == config.ProviderOpenAI {
			endpoint = "the OpenAI API"
		}
	}
	apiKey := "none"
	if cfg.APIKeySource != "" {
		apiKey = "from " + cfg.APIKeySource
	}
	fmt.Printf("AI is configured.\n  Provider:    %s\n  Model:       %s\n  Cheap model: %s\n  Effort:      %s\n  Endpoint:    %s\n  API key:     %s\n",
		cfg.Provider, cfg.Model, cfg.CheapModel, cfg.Effort, endpoint, apiKey)
}

func printTokens(usage llm.Usage) {
	fmt.Printf("Tokens: %d input, %d output\n", usage.InputTokens, usage.OutputTokens)
}

func connectModel(_ context.Context, cfg config.Config) (llm.Model, error) {
	if cfg.Provider == config.ProviderOpenAI {
		adapter, err := openai.New(openai.Settings{
			APIKey:        cfg.APIKey,
			Endpoint:      cfg.Endpoint,
			DefaultModel:  cfg.Model,
			CheapModel:    cfg.CheapModel,
			DefaultEffort: cfg.Effort,
		})
		if err != nil {
			return nil, err
		}
		return adapter, nil
	}
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
