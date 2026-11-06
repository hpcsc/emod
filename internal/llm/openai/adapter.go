package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/schema"
)

const (
	defaultEndpoint = "https://api.openai.com/v1"
	defaultTimeout  = 10 * time.Minute
	maxErrorBody    = 64 << 10
)

type Settings struct {
	APIKey        string
	Endpoint      string
	DefaultModel  string
	CheapModel    string
	DefaultEffort llm.Effort
}

type Option func(*config)

type config struct {
	httpClient *http.Client
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *config) {
		c.httpClient = client
	}
}

type Adapter struct {
	client     *http.Client
	url        string
	apiKey     string
	model      string
	cheapModel string
	effort     llm.Effort
}

func New(settings Settings, opts ...Option) (*Adapter, error) {
	if settings.DefaultModel == "" {
		return nil, errors.New("no model for the OpenAI-compatible API")
	}

	cfg := config{httpClient: &http.Client{Timeout: defaultTimeout}}
	for _, opt := range opts {
		opt(&cfg)
	}

	endpoint := settings.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	cheapModel := settings.CheapModel
	if cheapModel == "" {
		cheapModel = settings.DefaultModel
	}

	return &Adapter{
		client:     cfg.httpClient,
		url:        strings.TrimSuffix(endpoint, "/") + "/chat/completions",
		apiKey:     settings.APIKey,
		model:      settings.DefaultModel,
		cheapModel: cheapModel,
		effort:     settings.DefaultEffort,
	}, nil
}

func (a *Adapter) Cheap() *Adapter {
	cheap := *a
	cheap.model = a.cheapModel
	return &cheap
}

func (a *Adapter) Generate(ctx context.Context, request llm.GenerateRequest) (llm.Response, error) {
	var output *schema.Output
	if len(request.Schema) > 0 {
		var err error
		output, err = schema.Parse(request.Schema)
		if err != nil {
			return llm.Response{}, err
		}
	}

	body := chatRequest{
		Model:           a.model,
		Messages:        toMessages(request.System, request.Messages),
		ReasoningEffort: reasoningEffort(request.Effort, a.effort),
	}
	if output != nil {
		body.ResponseFormat = &responseFormat{
			Type:       "json_schema",
			JSONSchema: jsonSchemaFormat{Name: "answer", Schema: output.Document(), Strict: true},
		}
	}

	answer, err := a.post(ctx, body)
	var rejected *rejectedError
	for errors.As(err, &rejected) {
		switch {
		case body.ReasoningEffort != "" && rejected.names("reasoning_effort"):
			body.ReasoningEffort = ""
		case body.ResponseFormat != nil && (rejected.names("response_format") || rejected.names("json_schema")):
			body.ResponseFormat = nil
			body.Messages = withSchemaInstruction(body.Messages, output)
		default:
			return llm.Response{}, err
		}
		answer, err = a.post(ctx, body)
	}
	if err != nil {
		return llm.Response{}, err
	}

	usage := llm.Usage{InputTokens: answer.Usage.PromptTokens, OutputTokens: answer.Usage.CompletionTokens}
	if len(answer.Choices) == 0 {
		return llm.Response{Usage: usage}, fmt.Errorf("%s answered with no choice", a.url)
	}
	text := answer.Choices[0].Message.Content

	if output != nil {
		text = withoutFence(text)
		if err := output.Check(text); err != nil {
			return llm.Response{Usage: usage}, err
		}
	}

	return llm.Response{Text: text, Usage: usage}, nil
}

func (a *Adapter) post(ctx context.Context, body chatRequest) (chatResponse, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return chatResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url, bytes.NewReader(encoded))
	if err != nil {
		return chatResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if a.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+a.apiKey)
	}

	response, err := a.client.Do(httpRequest)
	if err != nil {
		return chatResponse{}, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
		message := errorMessage(raw)
		if response.StatusCode == http.StatusBadRequest {
			return chatResponse{}, &rejectedError{url: a.url, message: message}
		}
		return chatResponse{}, fmt.Errorf("%s answered %s: %s", a.url, response.Status, message)
	}

	var answer chatResponse
	if err := json.NewDecoder(response.Body).Decode(&answer); err != nil {
		return chatResponse{}, fmt.Errorf("%s answered with a body that is not a chat completion: %w", a.url, err)
	}
	return answer, nil
}

func toMessages(system string, messages []llm.Message) []chatMessage {
	converted := make([]chatMessage, 0, len(messages)+1)
	if system != "" {
		converted = append(converted, chatMessage{Role: "system", Content: system})
	}
	for _, message := range messages {
		converted = append(converted, chatMessage{Role: message.Role, Content: message.Content})
	}
	return converted
}

func reasoningEffort(requested, fallback llm.Effort) string {
	effort := requested
	if effort == llm.EffortUnset {
		effort = fallback
	}
	if effort == llm.EffortXHigh {
		effort = llm.EffortHigh
	}
	return effort.String()
}

func withSchemaInstruction(messages []chatMessage, output *schema.Output) []chatMessage {
	document, _ := json.Marshal(output.Document())
	instruction := "Answer with one JSON value and nothing else. The value must conform to this JSON Schema:\n" + string(document)

	changed := append([]chatMessage(nil), messages...)
	if len(changed) > 0 && changed[0].Role == "system" {
		changed[0].Content += "\n\n" + instruction
		return changed
	}
	return append([]chatMessage{{Role: "system", Content: instruction}}, changed...)
}

func withoutFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") || !strings.HasSuffix(trimmed, "```") {
		return text
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "```"), "```")
	if newline := strings.IndexByte(inner, '\n'); newline >= 0 {
		inner = inner[newline+1:]
	}
	return strings.TrimSpace(inner)
}

func errorMessage(raw []byte) string {
	var envelope struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && len(envelope.Error) > 0 {
		var detail struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(envelope.Error, &detail); err == nil && detail.Message != "" {
			return detail.Message
		}
		var text string
		if err := json.Unmarshal(envelope.Error, &text); err == nil && text != "" {
			return text
		}
	}
	return strings.TrimSpace(string(raw))
}

type rejectedError struct {
	url     string
	message string
}

func (e *rejectedError) Error() string {
	return fmt.Sprintf("%s refused the request: %s", e.url, e.message)
}

func (e *rejectedError) names(field string) bool {
	return strings.Contains(strings.ToLower(e.message), field)
}

type chatRequest struct {
	Model           string          `json:"model"`
	Messages        []chatMessage   `json:"messages"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
	ResponseFormat  *responseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type       string           `json:"type"`
	JSONSchema jsonSchemaFormat `json:"json_schema"`
}

type jsonSchemaFormat struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
	Strict bool           `json:"strict"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

var _ llm.Model = (*Adapter)(nil)
