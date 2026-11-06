//go:build unit

package repair_test

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/diagnostic"
	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/repair"
	"github.com/hpcsc/emod/internal/oracle"
	"github.com/hpcsc/emod/internal/test"
	"github.com/stretchr/testify/require"
)

var (
	cleanSource       = test.BillingPayments
	unparseableSource = test.Unparseable
	invalidSource     = strings.Replace(test.BillingPayments, "TakePayment -> PaymentTaken", "TakePayment -> PaymentRefunded", 1)
)

func newRequest() llm.GenerateRequest {
	return llm.GenerateRequest{
		System:   "You write emod models.",
		Messages: []llm.Message{{Role: "user", Content: "Model a payment that a customer takes."}},
		Effort:   llm.EffortLow,
	}
}

func answer(text string) llm.Response {
	return llm.Response{Text: text, Usage: llm.Usage{InputTokens: 1, OutputTokens: 1}}
}

func TestGenerate(t *testing.T) {
	t.Run("convergence", func(t *testing.T) {
		t.Run("returns the first answer when it passes the oracle", func(t *testing.T) {
			model := llm.NewFake(answer(cleanSource))

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)
			require.NoError(t, err)

			require.True(t, outcome.Converged)
			require.Equal(t, cleanSource, outcome.Source)
			require.Equal(t, 1, outcome.Attempts)
			require.Empty(t, outcome.Diagnostics)
			require.Len(t, model.Requests(), 1)
		})

		t.Run("returns a repaired answer that passes the oracle on a later attempt", func(t *testing.T) {
			model := llm.NewFake(answer(unparseableSource), answer(invalidSource), answer(cleanSource))

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)
			require.NoError(t, err)

			require.True(t, outcome.Converged)
			require.Equal(t, cleanSource, outcome.Source)
			require.Equal(t, 3, outcome.Attempts)
			require.Empty(t, outcome.Diagnostics)
		})

		t.Run("reads the source out of a fenced code block", func(t *testing.T) {
			model := llm.NewFake(answer("```hcl\n" + cleanSource + "```\n"))

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)
			require.NoError(t, err)

			require.True(t, outcome.Converged)
			require.Equal(t, cleanSource, outcome.Source)
		})

		t.Run("reads the source out of a fenced code block that prose surrounds", func(t *testing.T) {
			model := llm.NewFake(answer("Here is the model:\n```hcl\n" + cleanSource + "```\nIt has one slice.\n"))

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)
			require.NoError(t, err)

			require.True(t, outcome.Converged)
			require.Equal(t, cleanSource, outcome.Source)
		})
	})

	t.Run("repair", func(t *testing.T) {
		t.Run("sends each failing source and its diagnostics back to the model", func(t *testing.T) {
			model := llm.NewFake(answer(unparseableSource), answer(invalidSource), answer(cleanSource))

			_, err := repair.Generate(context.Background(), model, newRequest(), 3)
			require.NoError(t, err)

			messages := model.Requests()[2].Messages
			require.Len(t, messages, 5)
			require.Equal(t, newRequest().Messages[0], messages[0])
			require.Equal(t, llm.Message{Role: "assistant", Content: unparseableSource}, messages[1])
			require.Equal(t, llm.Message{Role: "assistant", Content: invalidSource}, messages[3])
			for _, pair := range []struct {
				feedback llm.Message
				source   string
			}{
				{feedback: messages[2], source: unparseableSource},
				{feedback: messages[4], source: invalidSource},
			} {
				require.Equal(t, "user", pair.feedback.Role)
				diagnostics := oracle.Check(pair.source, "generated.emod")
				require.NotEmpty(t, diagnostics)
				for _, diagnostic := range diagnostics {
					require.Contains(t, pair.feedback.Content, diagnostic.String())
				}
			}
		})

		t.Run("asks again for a source when an answer holds none", func(t *testing.T) {
			for _, blank := range []string{"", "   \n", "```hcl\n```\n"} {
				t.Run(strconv.Quote(blank), func(t *testing.T) {
					model := llm.NewFake(answer(blank), answer(cleanSource))

					outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)
					require.NoError(t, err)

					require.True(t, outcome.Converged)
					require.Equal(t, 2, outcome.Attempts)
					require.Equal(t, cleanSource, outcome.Source)
					require.Contains(t, model.Requests()[1].Messages[2].Content, "generated.emod:1: the answer holds no emod source")
				})
			}
		})

		t.Run("does not write into the messages of the request that the caller gave", func(t *testing.T) {
			base := newRequest()
			base.Messages = append(make([]llm.Message, 0, 8), base.Messages...)
			first := llm.NewFake(answer(unparseableSource), answer(cleanSource))
			second := llm.NewFake(answer(invalidSource), answer(cleanSource))

			_, err := repair.Generate(context.Background(), first, base, 3)
			require.NoError(t, err)
			_, err = repair.Generate(context.Background(), second, base, 3)
			require.NoError(t, err)

			require.Equal(t, unparseableSource, first.Requests()[1].Messages[1].Content)
			require.Equal(t, invalidSource, second.Requests()[1].Messages[1].Content)
		})

		t.Run("keeps the system prompt and effort of the request on every attempt", func(t *testing.T) {
			model := llm.NewFake(answer(unparseableSource), answer(cleanSource))

			_, err := repair.Generate(context.Background(), model, newRequest(), 3)
			require.NoError(t, err)

			retry := model.Requests()[1]
			require.Equal(t, "You write emod models.", retry.System)
			require.Equal(t, llm.EffortLow, retry.Effort)
		})

	})

	t.Run("non-convergence", func(t *testing.T) {
		t.Run("reports that it did not converge when every attempt fails the oracle", func(t *testing.T) {
			model := llm.NewFake(answer(unparseableSource), answer(unparseableSource), answer(invalidSource))

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)
			require.NoError(t, err)

			require.False(t, outcome.Converged)
			require.Equal(t, invalidSource, outcome.Source)
			require.Equal(t, 3, outcome.Attempts)
			require.Equal(t, oracle.Check(invalidSource, "generated.emod"), outcome.Diagnostics)
		})

		t.Run("asks the model no more times than the attempt bound", func(t *testing.T) {
			model := llm.NewFake(answer(unparseableSource), answer(unparseableSource), answer(unparseableSource))

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 2)
			require.NoError(t, err)

			require.False(t, outcome.Converged)
			require.Equal(t, 2, outcome.Attempts)
			require.Len(t, model.Requests(), 2)
		})

		t.Run("asks the model once and repairs nothing with an attempt bound of one", func(t *testing.T) {
			model := llm.NewFake(answer(unparseableSource), answer(cleanSource))

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 1)
			require.NoError(t, err)

			require.False(t, outcome.Converged)
			require.Equal(t, 1, outcome.Attempts)
			require.Len(t, model.Requests(), 1)
		})

		t.Run("reports that it did not converge when no answer holds a source", func(t *testing.T) {
			model := llm.NewFake(answer(""), answer(""))

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 2)
			require.NoError(t, err)

			require.False(t, outcome.Converged)
			require.Len(t, outcome.Diagnostics, 1)
			require.Equal(t, "the answer holds no emod source", outcome.Diagnostics[0].Message)
			require.Equal(t, diagnostic.Error, outcome.Diagnostics[0].Severity)
		})

		t.Run("rejects an attempt bound below one without asking the model", func(t *testing.T) {
			model := llm.NewFake(answer(cleanSource))

			_, err := repair.Generate(context.Background(), model, newRequest(), 0)

			require.ErrorContains(t, err, "at least 1")
			require.Empty(t, model.Requests())
		})
	})

	t.Run("progress", func(t *testing.T) {
		t.Run("reports each attempt with its number, the attempt bound, its diagnostics and its usage", func(t *testing.T) {
			model := llm.NewFake(
				llm.Response{Text: unparseableSource, Usage: llm.Usage{InputTokens: 100, OutputTokens: 20}},
				llm.Response{Text: cleanSource, Usage: llm.Usage{InputTokens: 50, OutputTokens: 7}},
			)
			var attempts []repair.Attempt

			_, err := repair.Generate(context.Background(), model, newRequest(), 3,
				repair.WithProgress(func(attempt repair.Attempt) { attempts = append(attempts, attempt) }))

			require.NoError(t, err)
			require.Equal(t, []repair.Attempt{
				{Number: 1, MaxAttempts: 3, Diagnostics: oracle.Check(unparseableSource, "generated.emod"), Usage: llm.Usage{InputTokens: 100, OutputTokens: 20}},
				{Number: 2, MaxAttempts: 3, Diagnostics: oracle.Check(cleanSource, "generated.emod"), Usage: llm.Usage{InputTokens: 50, OutputTokens: 7}},
			}, attempts)
		})

		t.Run("reports every attempt up to the bound when no answer converges", func(t *testing.T) {
			model := llm.NewFake(answer(unparseableSource), answer(unparseableSource), answer(unparseableSource))
			var numbers []int

			_, err := repair.Generate(context.Background(), model, newRequest(), 3,
				repair.WithProgress(func(attempt repair.Attempt) { numbers = append(numbers, attempt.Number) }))

			require.NoError(t, err)
			require.Equal(t, []int{1, 2, 3}, numbers)
		})

		t.Run("reports no attempt for a model call that fails", func(t *testing.T) {
			model := llm.NewBroken().WithError(errors.New("the provider is unavailable"))
			var attempts []repair.Attempt

			_, err := repair.Generate(context.Background(), model, newRequest(), 3,
				repair.WithProgress(func(attempt repair.Attempt) { attempts = append(attempts, attempt) }))

			require.Error(t, err)
			require.Empty(t, attempts)
		})
	})

	t.Run("usage", func(t *testing.T) {
		t.Run("sums the usage of every attempt", func(t *testing.T) {
			model := llm.NewFake(
				llm.Response{Text: unparseableSource, Usage: llm.Usage{InputTokens: 100, OutputTokens: 20}},
				llm.Response{Text: cleanSource, Usage: llm.Usage{InputTokens: 50, OutputTokens: 7}},
			)

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)
			require.NoError(t, err)

			require.Equal(t, llm.Usage{InputTokens: 150, OutputTokens: 27}, outcome.Usage)
		})

		t.Run("reports the usage that a failed call spends", func(t *testing.T) {
			model := llm.NewBroken().
				WithError(errors.New("the structured output does not conform to the request schema")).
				WithUsage(llm.Usage{InputTokens: 40, OutputTokens: 9})

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)

			require.ErrorContains(t, err, "does not conform")
			require.Equal(t, llm.Usage{InputTokens: 40, OutputTokens: 9}, outcome.Usage)
		})

		t.Run("keeps the usage of earlier attempts when a later call fails", func(t *testing.T) {
			model := llm.NewFake(llm.Response{Text: unparseableSource, Usage: llm.Usage{InputTokens: 100, OutputTokens: 20}})

			outcome, err := repair.Generate(context.Background(), model, newRequest(), 3)

			require.ErrorIs(t, err, llm.ErrNoMoreResponses)
			require.Equal(t, llm.Usage{InputTokens: 100, OutputTokens: 20}, outcome.Usage)
		})
	})

	t.Run("failure", func(t *testing.T) {
		t.Run("returns the error of a failed model call", func(t *testing.T) {
			unavailable := errors.New("the provider is unavailable")
			model := llm.NewBroken().WithError(unavailable)

			_, err := repair.Generate(context.Background(), model, newRequest(), 3)

			require.ErrorIs(t, err, unavailable)
		})
	})
}
