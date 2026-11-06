//go:build unit

package bedrock_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/bedrock"
	"github.com/stretchr/testify/require"
)

const (
	defaultModelID = "anthropic.claude-opus-4-8"
	cheapModelID   = "anthropic.claude-haiku-4-5"
)

// capturingTransport short-circuits the network: it records the outgoing
// request (so tests can assert on the real serialized body and URL the
// adapter builds) and replies with a single canned Messages response. This is
// the injection seam the adapter exposes for offline testing; it is not a mock
// of the SDK.
type capturingTransport struct {
	url  string
	body []byte
	resp string
}

func (c *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.url = req.URL.String()
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		c.body = body
	}

	resp := c.resp
	if resp == "" {
		resp = cannedResponse("text", 1, 1)
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(resp)),
		Request:    req,
	}, nil
}

func cannedResponse(text string, inputTokens, outputTokens int) string {
	payload := map[string]any{
		"id":    "msg_test",
		"type":  "message",
		"role":  "assistant",
		"model": defaultModelID,
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"stop_reason": "end_turn",
		"usage": map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func newSettings() bedrock.Settings {
	return bedrock.Settings{
		Region:        "us-east-1",
		DefaultModel:  defaultModelID,
		CheapModel:    cheapModelID,
		DefaultEffort: llm.EffortMedium,
	}
}

func newAdapter(t *testing.T, settings bedrock.Settings, transport http.RoundTripper) *bedrock.Adapter {
	t.Helper()

	// The Mantle client signs with SigV4; static fake credentials keep signing
	// offline and credential-free while the injected transport blocks the call.
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_REGION", settings.Region)

	adapter, err := bedrock.New(context.Background(), settings, bedrock.WithHTTPClient(&http.Client{Transport: transport}))
	require.NoError(t, err)

	return adapter
}

func decodeBody(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(body, &decoded))

	return decoded
}

func TestAdapter(t *testing.T) {
	t.Run("generate", func(t *testing.T) {
		t.Run("issues a request for the configured default model", func(t *testing.T) {
			transport := &capturingTransport{}
			adapter := newAdapter(t, newSettings(), transport)

			_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "hello"}},
			})
			require.NoError(t, err)

			require.Equal(t, defaultModelID, decodeBody(t, transport.body)["model"])
		})

		t.Run("sends a blank message as a text block that is not blank", func(t *testing.T) {
			transport := &capturingTransport{}
			adapter := newAdapter(t, newSettings(), transport)

			_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{
					{Role: "user", Content: "describe the event"},
					{Role: "assistant", Content: "  \n"},
					{Role: "user", Content: "the answer holds no source"},
				},
			})
			require.NoError(t, err)

			messages, ok := decodeBody(t, transport.body)["messages"].([]any)
			require.True(t, ok)
			require.Len(t, messages, 3)
			blank := messages[1].(map[string]any)
			require.Equal(t, "assistant", blank["role"])
			text := blank["content"].([]any)[0].(map[string]any)["text"].(string)
			require.NotEmpty(t, strings.TrimSpace(text), "the Messages API rejects a blank text block")
		})

		t.Run("parses canned response text and token usage", func(t *testing.T) {
			transport := &capturingTransport{resp: cannedResponse("generated answer", 137, 42)}
			adapter := newAdapter(t, newSettings(), transport)

			response, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "hello"}},
			})
			require.NoError(t, err)

			require.Equal(t, llm.Response{
				Text: "generated answer",
				Usage: llm.Usage{
					InputTokens:  137,
					OutputTokens: 42,
				},
			}, response)
		})

		t.Run("requests adaptive thinking", func(t *testing.T) {
			transport := &capturingTransport{}
			adapter := newAdapter(t, newSettings(), transport)

			_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "hello"}},
			})
			require.NoError(t, err)

			thinking, ok := decodeBody(t, transport.body)["thinking"].(map[string]any)
			require.True(t, ok, "request must carry a thinking config")
			require.Equal(t, "adaptive", thinking["type"])
		})

		t.Run("omits parameters rejected by the Opus-4.8 generation", func(t *testing.T) {
			transport := &capturingTransport{}
			adapter := newAdapter(t, newSettings(), transport)

			_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "hello"}},
				Effort:   llm.EffortHigh,
			})
			require.NoError(t, err)

			body := decodeBody(t, transport.body)
			for _, key := range []string{"temperature", "top_p", "top_k"} {
				_, present := body[key]
				require.False(t, present, "request must not contain %q", key)
			}

			thinking, _ := body["thinking"].(map[string]any)
			_, budgetPresent := thinking["budget_tokens"]
			require.False(t, budgetPresent, "adaptive thinking must not carry budget_tokens")
		})
	})

	t.Run("model selection", func(t *testing.T) {
		t.Run("cheap path issues a request for the cheap model without altering the interface", func(t *testing.T) {
			transport := &capturingTransport{}
			adapter := newAdapter(t, newSettings(), transport)

			var cheap llm.Model = adapter.Cheap()

			_, err := cheap.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "hello"}},
			})
			require.NoError(t, err)

			require.Equal(t, cheapModelID, decodeBody(t, transport.body)["model"])
		})
	})

	t.Run("effort mapping", func(t *testing.T) {
		cases := []struct {
			name     string
			effort   llm.Effort
			expected string
		}{
			{name: "low maps to low", effort: llm.EffortLow, expected: "low"},
			{name: "medium maps to medium", effort: llm.EffortMedium, expected: "medium"},
			{name: "high maps to high", effort: llm.EffortHigh, expected: "high"},
			{name: "xhigh maps to xhigh", effort: llm.EffortXHigh, expected: "xhigh"},
			{name: "unset defaults to medium", effort: llm.EffortUnset, expected: "medium"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				transport := &capturingTransport{}
				adapter := newAdapter(t, newSettings(), transport)

				_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
					Messages: []llm.Message{{Role: "user", Content: "hello"}},
					Effort:   tc.effort,
				})
				require.NoError(t, err)

				outputConfig, ok := decodeBody(t, transport.body)["output_config"].(map[string]any)
				require.True(t, ok, "request must carry an output_config")
				require.Equal(t, tc.expected, outputConfig["effort"])
			})
		}
	})

	t.Run("structured output", func(t *testing.T) {
		eventSchema := `{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"fields": {"type": "array", "items": {"type": "string"}}
			},
			"required": ["name", "fields"],
			"additionalProperties": false
		}`

		t.Run("asks for the request schema as the JSON output format", func(t *testing.T) {
			transport := &capturingTransport{resp: cannedResponse(`{"name":"OrderPlaced","fields":[]}`, 1, 1)}
			adapter := newAdapter(t, newSettings(), transport)

			_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "describe the event"}},
				Schema:   json.RawMessage(eventSchema),
			})
			require.NoError(t, err)

			var expectedSchema map[string]any
			require.NoError(t, json.Unmarshal([]byte(eventSchema), &expectedSchema))
			outputConfig, ok := decodeBody(t, transport.body)["output_config"].(map[string]any)
			require.True(t, ok, "request must carry an output_config")
			require.Equal(t, map[string]any{"type": "json_schema", "schema": expectedSchema}, outputConfig["format"])
		})

		t.Run("returns conforming output that a feature decodes into its own type", func(t *testing.T) {
			transport := &capturingTransport{resp: cannedResponse(`{"name":"OrderPlaced","fields":["orderId","total"]}`, 1, 1)}
			adapter := newAdapter(t, newSettings(), transport)

			response, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "describe the event"}},
				Schema:   json.RawMessage(eventSchema),
			})
			require.NoError(t, err)

			var event struct {
				Name   string   `json:"name"`
				Fields []string `json:"fields"`
			}
			require.NoError(t, json.Unmarshal([]byte(response.Text), &event))
			require.Equal(t, "OrderPlaced", event.Name)
			require.Equal(t, []string{"orderId", "total"}, event.Fields)
		})

		t.Run("reports output that does not conform to the schema as an error", func(t *testing.T) {
			transport := &capturingTransport{resp: cannedResponse(`{"name":"OrderPlaced"}`, 1, 1)}
			adapter := newAdapter(t, newSettings(), transport)

			response, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "describe the event"}},
				Schema:   json.RawMessage(eventSchema),
			})

			require.ErrorContains(t, err, "does not conform to the request schema")
			require.ErrorContains(t, err, "fields")
			require.Empty(t, response.Text)
		})

		t.Run("reports output that is not JSON as an error", func(t *testing.T) {
			transport := &capturingTransport{resp: cannedResponse("OrderPlaced has two fields", 1, 1)}
			adapter := newAdapter(t, newSettings(), transport)

			response, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "describe the event"}},
				Schema:   json.RawMessage(eventSchema),
			})

			require.ErrorContains(t, err, "is not JSON")
			require.Empty(t, response.Text)
		})

		t.Run("still reports the tokens that a non-conforming answer spent", func(t *testing.T) {
			transport := &capturingTransport{resp: cannedResponse(`{"name":"OrderPlaced"}`, 120, 30)}
			adapter := newAdapter(t, newSettings(), transport)

			response, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "describe the event"}},
				Schema:   json.RawMessage(eventSchema),
			})

			require.Error(t, err)
			require.Equal(t, llm.Usage{InputTokens: 120, OutputTokens: 30}, response.Usage)
		})

		t.Run("rejects an unusable schema without calling the model", func(t *testing.T) {
			cases := []struct {
				name   string
				schema string
			}{
				{name: "a schema that is not JSON", schema: `{"type": "object"`},
				{name: "a schema that is a bare number", schema: `1`},
				{name: "a schema that is a bare boolean", schema: `true`},
				{name: "a schema that is null", schema: `null`},
				{name: "a keyword with a value of the wrong type", schema: `{"type": 12}`},
				{name: "a reference to a definition that does not exist", schema: `{"$ref": "#/$defs/missing"}`},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					transport := &capturingTransport{}
					adapter := newAdapter(t, newSettings(), transport)

					_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
						Messages: []llm.Message{{Role: "user", Content: "describe the event"}},
						Schema:   json.RawMessage(tc.schema),
					})

					require.ErrorContains(t, err, "not a valid JSON Schema")
					require.Empty(t, transport.url, "no request must reach the model")
				})
			}
		})

		t.Run("keeps a request without a schema on the plain-text path", func(t *testing.T) {
			transport := &capturingTransport{resp: cannedResponse("OrderPlaced has two fields", 1, 1)}
			adapter := newAdapter(t, newSettings(), transport)

			response, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "describe the event"}},
			})
			require.NoError(t, err)

			require.Equal(t, "OrderPlaced has two fields", response.Text)
			outputConfig, _ := decodeBody(t, transport.body)["output_config"].(map[string]any)
			_, formatPresent := outputConfig["format"]
			require.False(t, formatPresent, "a request without a schema must not ask for a JSON output format")
		})
	})

	t.Run("endpoint", func(t *testing.T) {
		t.Run("routes through the configured gateway base URL", func(t *testing.T) {
			settings := newSettings()
			settings.Endpoint = "https://gateway.internal.example/anthropic"
			transport := &capturingTransport{}
			adapter := newAdapter(t, settings, transport)

			_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "hello"}},
			})
			require.NoError(t, err)

			require.True(t, strings.HasPrefix(transport.url, "https://gateway.internal.example/anthropic"),
				"expected request to target the gateway, got %q", transport.url)
		})

		t.Run("targets the default Bedrock host when no endpoint is configured", func(t *testing.T) {
			transport := &capturingTransport{}
			adapter := newAdapter(t, newSettings(), transport)

			_, err := adapter.Generate(context.Background(), llm.GenerateRequest{
				Messages: []llm.Message{{Role: "user", Content: "hello"}},
			})
			require.NoError(t, err)

			require.Contains(t, transport.url, "bedrock-mantle.us-east-1.api.aws")
		})
	})
}
