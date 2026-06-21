package bedrock

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/hpcsc/emod/internal/llm"
)

// defaultMaxTokens caps a non-streaming generation so a long response cannot stall on an HTTP timeout.
const defaultMaxTokens int64 = 16384

// Settings holds the resolved adapter configuration. Model IDs must be the
// Bedrock-prefixed strings (e.g. "anthropic.claude-opus-4-8"); the bare SDK
// constants are rejected by Bedrock.
type Settings struct {
	Region        string
	DefaultModel  string
	CheapModel    string
	DefaultEffort llm.Effort
	Endpoint      string
}

type Option func(*config)

type config struct {
	httpClient option.HTTPClient
	maxTokens  int64
}

func WithHTTPClient(client option.HTTPClient) Option {
	return func(c *config) {
		c.httpClient = client
	}
}

func WithMaxTokens(maxTokens int64) Option {
	return func(c *config) {
		c.maxTokens = maxTokens
	}
}

type Adapter struct {
	messages   anthropic.MessageService
	model      string
	cheapModel string
	effort     llm.Effort
	maxTokens  int64
}

func New(ctx context.Context, settings Settings, opts ...Option) (*Adapter, error) {
	cfg := config{maxTokens: defaultMaxTokens}
	for _, opt := range opts {
		opt(&cfg)
	}

	var requestOptions []option.RequestOption
	if cfg.httpClient != nil {
		requestOptions = append(requestOptions, option.WithHTTPClient(cfg.httpClient))
	}

	client, err := bedrock.NewMantleClient(ctx, bedrock.MantleClientConfig{
		AWSRegion: settings.Region,
		BaseURL:   settings.Endpoint,
	}, requestOptions...)
	if err != nil {
		return nil, err
	}

	return &Adapter{
		messages:   client.Messages,
		model:      settings.DefaultModel,
		cheapModel: settings.CheapModel,
		effort:     settings.DefaultEffort,
		maxTokens:  cfg.maxTokens,
	}, nil
}

func (a *Adapter) Cheap() *Adapter {
	cheap := *a
	cheap.model = a.cheapModel
	return &cheap
}

func (a *Adapter) Generate(ctx context.Context, request llm.GenerateRequest) (llm.Response, error) {
	params := anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: a.maxTokens,
		Messages:  toMessageParams(request.Messages),
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
		},
		OutputConfig: anthropic.OutputConfigParam{
			Effort: anthropic.OutputConfigEffort(resolveEffort(request.Effort, a.effort).String()),
		},
	}

	if request.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: request.System}}
	}

	// request.Schema is intentionally unwired: structured output is not supported yet, so the request returns plain text.

	message, err := a.messages.New(ctx, params)
	if err != nil {
		return llm.Response{}, err
	}

	return llm.Response{
		Text: extractText(message),
		Usage: llm.Usage{
			InputTokens:  int(message.Usage.InputTokens),
			OutputTokens: int(message.Usage.OutputTokens),
		},
	}, nil
}

func resolveEffort(requested, fallback llm.Effort) llm.Effort {
	if requested == llm.EffortUnset {
		return fallback
	}
	return requested
}

func toMessageParams(messages []llm.Message) []anthropic.MessageParam {
	params := make([]anthropic.MessageParam, 0, len(messages))
	for _, message := range messages {
		block := anthropic.NewTextBlock(message.Content)
		if message.Role == "assistant" {
			params = append(params, anthropic.NewAssistantMessage(block))
			continue
		}
		params = append(params, anthropic.NewUserMessage(block))
	}
	return params
}

func extractText(message *anthropic.Message) string {
	var builder strings.Builder
	for _, block := range message.Content {
		if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
			builder.WriteString(textBlock.Text)
		}
	}
	return builder.String()
}

var _ llm.Model = (*Adapter)(nil)
