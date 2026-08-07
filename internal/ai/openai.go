package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

var (
	ErrMissingAPIKey = errors.New("OpenAI API key is missing or not configured")
	ErrEmptyResponse = errors.New("received empty response from OpenAI")
)

type OpenAIProvider struct {
	client *openai.Client
	model  string
}

// NewOpenAIProvider creates an instance of OpenAIProvider using the specified API key and model.
// If apiKey is empty, it resolves from the default environment variable AI_PROVIDER_API_KEY_OPENAI.
func NewOpenAIProvider(apiKey string, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		apiKey = os.Getenv("AI_PROVIDER_API_KEY_OPENAI")
	}

	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	if model == "" {
		model = openai.GPT4oMini
	}

	client := openai.NewClient(apiKey)
	return &OpenAIProvider{
		client: client,
		model:  model,
	}, nil
}

// NewOpenAIProviderWithClient creates an instance using a custom Client (useful for unit testing).
func NewOpenAIProviderWithClient(client *openai.Client, model string) *OpenAIProvider {
	if model == "" {
		model = openai.GPT4oMini
	}
	return &OpenAIProvider{
		client: client,
		model:  model,
	}
}

func (p *OpenAIProvider) GenerateMenu(ctx context.Context, params MenuGenerateParams) (*MenuOptions, error) {
	systemPrompt := `Anda adalah ahli gizi dan koki profesional Indonesia.
Tugas Anda adalah merekomendasikan 3 variasi menu masak harian khas Indonesia dalam bentuk JSON yang terstruktur.

ATURAN STRICT:
1. Kembalikan HANYA format JSON valid tanpa teks markdown pembuka/penutup.
2. Setiap estimasi harga bahan dan total harga HARUS dalam integer Rupiah (tanpa desimal/koma).
3. Sesuaikan dengan batasan pantangan/alergi dan target budget user.

FORMAT JSON OUTPUT YANG WAJIB DIIKUTI:
{
  "options": [
    {
      "recipe_name": "Nama Masakan",
      "description": "Deskripsi singkat masakan",
      "estimated_total_price": 45000,
      "goal_tags": ["hemat", "sehat"],
      "ingredients": [
        {
          "name": "Bahan",
          "quantity": "2",
          "unit": "potong",
          "estimated_price": 5000
        }
      ]
    }
  ]
}`

	userPrompt := fmt.Sprintf(`Target Goal: %s
Budget Amount: Rp %d
Budget Period: %s
Jumlah Anggota Keluarga: %d orang
Pantangan / Alergi: %s

Berikan 3 opsi menu rekomendasi masakan harian Indonesia sesuai parameter di atas!`,
		params.Goal,
		params.BudgetAmount,
		params.BudgetPeriod,
		params.HouseholdSize,
		formatRestrictions(params.Restrictions),
	)

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
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, ErrEmptyResponse
	}

	content := resp.Choices[0].Message.Content

	var menuOpts MenuOptions
	if err := json.Unmarshal([]byte(content), &menuOpts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAI response JSON: %w", err)
	}

	return &menuOpts, nil
}

func formatRestrictions(restrictions []string) string {
	if len(restrictions) == 0 {
		return "Tidak ada"
	}
	return strings.Join(restrictions, ", ")
}
