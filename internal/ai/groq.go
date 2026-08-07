package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type GroqProvider struct {
	client *openai.Client
	model  string
}

// NewGroqProvider creates a new Groq API provider instance (OpenAI-compatible client pointing to Groq endpoint).
func NewGroqProvider(apiKey string, model string) (*GroqProvider, error) {
	if apiKey == "" {
		apiKey = os.Getenv("AI_PROVIDER_API_KEY_GROQ")
	}

	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.groq.com/openai/v1"

	client := openai.NewClientWithConfig(config)
	return &GroqProvider{
		client: client,
		model:  model,
	}, nil
}

func (p *GroqProvider) GenerateMenu(ctx context.Context, params MenuGenerateParams) (*MenuOptions, error) {
	systemPrompt := BuildSystemPrompt()
	userPrompt := BuildUserPrompt(params)

	req := openai.ChatCompletionRequest{
		Model: p.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Temperature: 0.7,
	}

	resp, err := p.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("Groq API call failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, ErrEmptyResponse
	}

	content := resp.Choices[0].Message.Content

	var menuOpts MenuOptions
	if err := json.Unmarshal([]byte(content), &menuOpts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Groq response JSON: %w", err)
	}

	if err := ValidateMenuOptions(&menuOpts); err != nil {
		return nil, fmt.Errorf("Groq AI response failed schema validation: %w", err)
	}

	return &menuOpts, nil
}
