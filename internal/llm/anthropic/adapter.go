package anthropic

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/hpcsc/emod/internal/llm"
	"github.com/hpcsc/emod/internal/llm/schema"
)

// defaultMaxTokens caps a non-streaming generation so a long response cannot stall on an HTTP timeout.
const defaultMaxTokens int64 = 16384

type Settings struct {
	APIKey        string
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

func New(settings Settings, opts ...Option) (*Adapter, error) {
	if settings.APIKey == "" {
		return nil, errors.New("no API key for the Anthropic API")
	}

	cfg := config{maxTokens: defaultMaxTokens}
	for _, opt := range opts {
		opt(&cfg)
	}

	requestOptions := []option.RequestOption{
		option.WithoutEnvironmentDefaults(),
		option.WithAPIKey(settings.APIKey),
	}
	if settings.Endpoint != "" {
		requestOptions = append(requestOptions, option.WithBaseURL(settings.Endpoint))
	}
	if cfg.httpClient != nil {
		requestOptions = append(requestOptions, option.WithHTTPClient(cfg.httpClient))
	}

	client := anthropic.NewClient(requestOptions...)

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

	var output *schema.Output
	if len(request.Schema) > 0 {
		var err error
		output, err = schema.Parse(request.Schema)
		if err != nil {
			return llm.Response{}, err
		}
		params.OutputConfig.Format = anthropic.JSONOutputFormatParam{Schema: output.Document()}
	}

	message, err := a.messages.New(ctx, params)
	if err != nil {
		return llm.Response{}, err
	}

	text := extractText(message)
	usage := llm.Usage{
		InputTokens:  int(message.Usage.InputTokens),
		OutputTokens: int(message.Usage.OutputTokens),
	}

	if output != nil {
		if err := output.Check(text); err != nil {
			return llm.Response{Usage: usage}, err
		}
	}

	return llm.Response{Text: text, Usage: usage}, nil
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
		content := message.Content
		// the Messages API rejects a text block that is empty or holds only whitespace
		if strings.TrimSpace(content) == "" {
			content = "(empty)"
		}
		block := anthropic.NewTextBlock(content)
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
