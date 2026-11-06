//go:build unit

package openai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/openai"
	"github.com/stretchr/testify/require"
)

const nameSchema = `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}`

type reply struct {
	status int
	body   string
}

func answer(text string, promptTokens, completionTokens int) reply {
	encoded, err := json.Marshal(map[string]any{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": text}, "finish_reason": "stop"}},
		"usage":   map[string]any{"prompt_tokens": promptTokens, "completion_tokens": completionTokens},
	})
	if err != nil {
		panic(err)
	}
	return reply{status: http.StatusOK, body: string(encoded)}
}

func refusal(status int, message string) reply {
	encoded, err := json.Marshal(map[string]any{"error": map[string]any{"message": message, "type": "invalid_request_error"}})
	if err != nil {
		panic(err)
	}
	return reply{status: status, body: string(encoded)}
}

type chatServer struct {
	server *httptest.Server

	mu             sync.Mutex
	replies        []reply
	paths          []string
	bodies         []map[string]any
	authorizations []string
}

func newChatServer(t *testing.T, replies ...reply) *chatServer {
	t.Helper()

	s := &chatServer{replies: replies}
	s.server = httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(s.server.Close)
	return s
}

func (s *chatServer) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	s.paths = append(s.paths, r.URL.Path)
	s.bodies = append(s.bodies, body)
	s.authorizations = append(s.authorizations, r.Header.Get("Authorization"))

	next := reply{status: http.StatusInternalServerError, body: `{"error":{"message":"no reply queued"}}`}
	if len(s.replies) > 0 {
		next, s.replies = s.replies[0], s.replies[1:]
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(next.status)
	_, _ = io.WriteString(w, next.body)
}

func settingsFor(s *chatServer) openai.Settings {
	return openai.Settings{
		APIKey:        "test-key",
		Endpoint:      s.server.URL + "/v1",
		DefaultModel:  "test-model",
		CheapModel:    "test-cheap-model",
		DefaultEffort: llm.EffortMedium,
	}
}

func newAdapter(t *testing.T, settings openai.Settings) *openai.Adapter {
	t.Helper()

	adapter, err := openai.New(settings)
	require.NoError(t, err)
	return adapter
}

func hello() llm.GenerateRequest {
	return llm.GenerateRequest{Messages: []llm.Message{{Role: "user", Content: "hello"}}}
}

type urlRecorder struct {
	url string
}

func (u *urlRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	u.url = r.URL.String()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(answer("hi", 1, 1).body)),
		Request:    r,
	}, nil
}

func TestAdapter(t *testing.T) {
	t.Run("generate", func(t *testing.T) {
		t.Run("posts the model, the system prompt and the messages to chat/completions under the endpoint", func(t *testing.T) {
			server := newChatServer(t, answer("hi", 1, 1))

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), llm.GenerateRequest{
				System: "You write event models.",
				Messages: []llm.Message{
					{Role: "user", Content: "Model a library."},
					{Role: "assistant", Content: "emod = 1"},
					{Role: "user", Content: "Fix the diagnostics."},
				},
			})

			require.NoError(t, err)
			require.Equal(t, []string{"/v1/chat/completions"}, server.paths)
			require.Equal(t, "test-model", server.bodies[0]["model"])
			require.Equal(t, []any{
				map[string]any{"role": "system", "content": "You write event models."},
				map[string]any{"role": "user", "content": "Model a library."},
				map[string]any{"role": "assistant", "content": "emod = 1"},
				map[string]any{"role": "user", "content": "Fix the diagnostics."},
			}, server.bodies[0]["messages"])
		})

		t.Run("sends the API key as a bearer token", func(t *testing.T) {
			server := newChatServer(t, answer("hi", 1, 1))

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.NoError(t, err)
			require.Equal(t, []string{"Bearer test-key"}, server.authorizations)
		})

		t.Run("sends no Authorization header when there is no API key, as a local server needs", func(t *testing.T) {
			server := newChatServer(t, answer("hi", 1, 1))
			settings := settingsFor(server)
			settings.APIKey = ""

			_, err := newAdapter(t, settings).Generate(context.Background(), hello())

			require.NoError(t, err)
			require.Equal(t, []string{""}, server.authorizations)
		})

		t.Run("returns the text and the token usage", func(t *testing.T) {
			server := newChatServer(t, answer("emod = 1", 1200, 300))

			response, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.NoError(t, err)
			require.Equal(t, llm.Response{Text: "emod = 1", Usage: llm.Usage{InputTokens: 1200, OutputTokens: 300}}, response)
		})

		t.Run("reports an answer with no choice as an error", func(t *testing.T) {
			server := newChatServer(t, reply{status: http.StatusOK, body: `{"choices":[],"usage":{"prompt_tokens":3,"completion_tokens":0}}`})

			response, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.ErrorContains(t, err, "answered with no choice")
			require.Equal(t, llm.Usage{InputTokens: 3}, response.Usage)
		})

		t.Run("reports an error status with the message of the server", func(t *testing.T) {
			server := newChatServer(t, refusal(http.StatusUnauthorized, "Incorrect API key provided"))

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.ErrorContains(t, err, "401 Unauthorized: Incorrect API key provided")
		})

		t.Run("reports an error that the server gives as a plain string", func(t *testing.T) {
			server := newChatServer(t, reply{status: http.StatusNotFound, body: `{"error":"model \"llama9\" not found"}`})

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.ErrorContains(t, err, `404 Not Found: model "llama9" not found`)
		})
	})

	t.Run("model selection", func(t *testing.T) {
		t.Run("the cheap adapter asks for the cheap model", func(t *testing.T) {
			server := newChatServer(t, answer("hi", 1, 1))

			_, err := newAdapter(t, settingsFor(server)).Cheap().Generate(context.Background(), hello())

			require.NoError(t, err)
			require.Equal(t, "test-cheap-model", server.bodies[0]["model"])
		})

		t.Run("the cheap adapter asks for the default model when no cheap model is set", func(t *testing.T) {
			server := newChatServer(t, answer("hi", 1, 1))
			settings := settingsFor(server)
			settings.CheapModel = ""

			_, err := newAdapter(t, settings).Cheap().Generate(context.Background(), hello())

			require.NoError(t, err)
			require.Equal(t, "test-model", server.bodies[0]["model"])
		})
	})

	t.Run("effort", func(t *testing.T) {
		cases := []struct {
			name      string
			requested llm.Effort
			sent      string
		}{
			{name: "an unset effort sends the default effort", requested: llm.EffortUnset, sent: "medium"},
			{name: "low sends low", requested: llm.EffortLow, sent: "low"},
			{name: "high sends high", requested: llm.EffortHigh, sent: "high"},
			{name: "xhigh sends high, the highest common level", requested: llm.EffortXHigh, sent: "high"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				server := newChatServer(t, answer("hi", 1, 1))
				request := hello()
				request.Effort = tc.requested

				_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), request)

				require.NoError(t, err)
				require.Equal(t, tc.sent, server.bodies[0]["reasoning_effort"])
			})
		}

		t.Run("drops reasoning_effort and asks again when the server names it in a 400", func(t *testing.T) {
			server := newChatServer(t,
				refusal(http.StatusBadRequest, "Unsupported parameter: 'reasoning_effort' is not supported with this model."),
				answer("hi", 1, 1),
			)

			response, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.NoError(t, err)
			require.Equal(t, "hi", response.Text)
			require.Len(t, server.bodies, 2)
			require.NotContains(t, server.bodies[1], "reasoning_effort")
		})

		t.Run("asks only one more time when the server refuses again", func(t *testing.T) {
			server := newChatServer(t,
				refusal(http.StatusBadRequest, "Unsupported parameter: 'reasoning_effort'"),
				refusal(http.StatusBadRequest, "Unsupported parameter: 'reasoning_effort'"),
			)

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.ErrorContains(t, err, "refused the request")
			require.Len(t, server.bodies, 2)
		})

		t.Run("does not ask again after a 400 that names no optional field", func(t *testing.T) {
			server := newChatServer(t, refusal(http.StatusBadRequest, "This model's maximum context length is 8192 tokens."))

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.ErrorContains(t, err, "maximum context length")
			require.Len(t, server.bodies, 1)
		})
	})

	t.Run("structured output", func(t *testing.T) {
		withSchema := func() llm.GenerateRequest {
			request := hello()
			request.Schema = json.RawMessage(nameSchema)
			return request
		}

		t.Run("asks for the schema as a strict json_schema response format", func(t *testing.T) {
			server := newChatServer(t, answer(`{"name":"Loan"}`, 1, 1))

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), withSchema())

			require.NoError(t, err)
			var schema map[string]any
			require.NoError(t, json.Unmarshal([]byte(nameSchema), &schema))
			require.Equal(t, map[string]any{
				"type":        "json_schema",
				"json_schema": map[string]any{"name": "answer", "schema": schema, "strict": true},
			}, server.bodies[0]["response_format"])
		})

		t.Run("returns output that conforms to the schema", func(t *testing.T) {
			server := newChatServer(t, answer(`{"name":"Loan"}`, 1, 1))

			response, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), withSchema())

			require.NoError(t, err)
			require.Equal(t, `{"name":"Loan"}`, response.Text)
		})

		t.Run("reports output that does not conform as an error and keeps the usage", func(t *testing.T) {
			server := newChatServer(t, answer(`{"title":"Loan"}`, 40, 9))

			response, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), withSchema())

			require.ErrorContains(t, err, "does not conform to the request schema")
			require.Equal(t, llm.Usage{InputTokens: 40, OutputTokens: 9}, response.Usage)
		})

		t.Run("rejects an unusable schema without a call", func(t *testing.T) {
			server := newChatServer(t)
			request := hello()
			request.Schema = json.RawMessage(`not json`)

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), request)

			require.ErrorContains(t, err, "not a valid JSON Schema")
			require.Empty(t, server.bodies)
		})

		t.Run("keeps a request without a schema free of response_format", func(t *testing.T) {
			server := newChatServer(t, answer("hi", 1, 1))

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), hello())

			require.NoError(t, err)
			require.NotContains(t, server.bodies[0], "response_format")
		})

		t.Run("puts the schema in the system prompt and asks again when the server names response_format", func(t *testing.T) {
			server := newChatServer(t,
				refusal(http.StatusBadRequest, "Invalid parameter: 'response_format' of type 'json_schema' is not supported with this model."),
				answer(`{"name":"Loan"}`, 1, 1),
			)
			request := withSchema()
			request.System = "You name aggregates."

			response, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), request)

			require.NoError(t, err)
			require.Equal(t, `{"name":"Loan"}`, response.Text)
			require.Len(t, server.bodies, 2)
			require.NotContains(t, server.bodies[1], "response_format")
			messages, ok := server.bodies[1]["messages"].([]any)
			require.True(t, ok)
			system, ok := messages[0].(map[string]any)
			require.True(t, ok)
			require.Equal(t, "system", system["role"])
			require.Contains(t, system["content"], "You name aggregates.")
			require.Contains(t, system["content"], `"required":["name"]`)
		})

		t.Run("takes the JSON out of a fenced block before the check", func(t *testing.T) {
			server := newChatServer(t,
				refusal(http.StatusBadRequest, "response_format is not supported"),
				answer("```json\n{\"name\":\"Loan\"}\n```", 1, 1),
			)

			response, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), withSchema())

			require.NoError(t, err)
			require.Equal(t, `{"name":"Loan"}`, response.Text)
		})

		t.Run("checks the answer that the system prompt asked for against the schema", func(t *testing.T) {
			server := newChatServer(t,
				refusal(http.StatusBadRequest, "response_format is not supported"),
				answer(`{"title":"Loan"}`, 1, 1),
			)

			_, err := newAdapter(t, settingsFor(server)).Generate(context.Background(), withSchema())

			require.ErrorContains(t, err, "does not conform to the request schema")
		})
	})

	t.Run("endpoint", func(t *testing.T) {
		t.Run("targets the OpenAI API when no endpoint is set", func(t *testing.T) {
			recorder := &urlRecorder{}
			adapter, err := openai.New(openai.Settings{DefaultModel: "gpt-test"}, openai.WithHTTPClient(&http.Client{Transport: recorder}))
			require.NoError(t, err)

			_, err = adapter.Generate(context.Background(), hello())

			require.NoError(t, err)
			require.Equal(t, "https://api.openai.com/v1/chat/completions", recorder.url)
		})

		t.Run("adds no second slash after an endpoint that ends with a slash", func(t *testing.T) {
			server := newChatServer(t, answer("hi", 1, 1))
			settings := settingsFor(server)
			settings.Endpoint = fmt.Sprintf("%s/v1/", server.server.URL)

			_, err := newAdapter(t, settings).Generate(context.Background(), hello())

			require.NoError(t, err)
			require.Equal(t, []string{"/v1/chat/completions"}, server.paths)
		})
	})

	t.Run("settings", func(t *testing.T) {
		t.Run("refuses settings with no model", func(t *testing.T) {
			_, err := openai.New(openai.Settings{Endpoint: "http://localhost:11434/v1"})

			require.ErrorContains(t, err, "no model")
		})
	})
}
