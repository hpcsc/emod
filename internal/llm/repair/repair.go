package repair

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/oracle"
)

const sourceFilename = "generated.emod"

type Outcome struct {
	Source      string
	Converged   bool
	Attempts    int
	Diagnostics []*diagnostic.Entry
	Usage       llm.Usage
}

type Attempt struct {
	Number      int
	MaxAttempts int
	Diagnostics []*diagnostic.Entry
	Usage       llm.Usage
}

type Option func(*options)

type options struct {
	progress func(Attempt)
}

func WithProgress(report func(Attempt)) Option {
	return func(o *options) {
		o.progress = report
	}
}

// On an error, the outcome still carries the usage of every call, the failed one included.
func Generate(ctx context.Context, model llm.Model, request llm.GenerateRequest, maxAttempts int, opts ...Option) (Outcome, error) {
	if maxAttempts < 1 {
		return Outcome{}, fmt.Errorf("the attempt bound must be at least 1, got %d", maxAttempts)
	}

	settings := options{progress: func(Attempt) {}}
	for _, opt := range opts {
		opt(&settings)
	}

	request.Messages = slices.Clip(request.Messages)

	var outcome Outcome
	for outcome.Attempts < maxAttempts {
		response, err := model.Generate(ctx, request)
		outcome.Usage = outcome.Usage.Add(response.Usage)
		if err != nil {
			return outcome, err
		}

		outcome.Attempts++
		outcome.Source = sourceOf(response.Text)
		outcome.Diagnostics = check(outcome.Source)
		settings.progress(Attempt{Number: outcome.Attempts, MaxAttempts: maxAttempts, Diagnostics: outcome.Diagnostics, Usage: response.Usage})
		if len(outcome.Diagnostics) == 0 {
			outcome.Converged = true
			return outcome, nil
		}

		request.Messages = append(request.Messages,
			llm.Message{Role: "assistant", Content: response.Text},
			llm.Message{Role: "user", Content: feedback(outcome.Diagnostics)},
		)
	}

	return outcome, nil
}

func check(source string) []*diagnostic.Entry {
	if strings.TrimSpace(source) == "" {
		return []*diagnostic.Entry{{
			Filename: sourceFilename,
			Line:     1,
			Message:  "the answer holds no emod source",
			Severity: diagnostic.Error,
		}}
	}
	return oracle.Check(source, sourceFilename)
}

func sourceOf(text string) string {
	_, fenced, found := strings.Cut(text, "```")
	if !found {
		return text
	}

	_, body, _ := strings.Cut(fenced, "\n")
	source, _, _ := strings.Cut(body, "```")
	return source
}

func feedback(diagnostics []*diagnostic.Entry) string {
	var builder strings.Builder
	builder.WriteString("The emod source you returned does not pass the checks. They report:\n")
	for _, entry := range diagnostics {
		builder.WriteString(entry.String())
		builder.WriteString("\n")
	}
	builder.WriteString("Return the whole corrected emod source and nothing else.")
	return builder.String()
}
