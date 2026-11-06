//go:build unit

package llm_test

import (
	"testing"

	"github.com/hpcsc/emod/internal/llm"
	"github.com/stretchr/testify/require"
)

func TestEffort(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		t.Run("unset value resolves to the documented default", func(t *testing.T) {
			var unset llm.Effort

			require.Equal(t, "medium", unset.String())
		})

		t.Run("each documented level reports its own canonical name", func(t *testing.T) {
			require.Equal(t, "low", llm.EffortLow.String())
			require.Equal(t, "medium", llm.EffortMedium.String())
			require.Equal(t, "high", llm.EffortHigh.String())
			require.Equal(t, "xhigh", llm.EffortXHigh.String())
		})
	})
}

func TestUsage(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		t.Run("sums input and output tokens separately", func(t *testing.T) {
			first := llm.Usage{InputTokens: 100, OutputTokens: 20}
			second := llm.Usage{InputTokens: 30, OutputTokens: 7}

			require.Equal(t, llm.Usage{InputTokens: 130, OutputTokens: 27}, first.Add(second))
		})
	})
}
