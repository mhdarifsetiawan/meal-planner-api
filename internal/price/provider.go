package price

import "context"

type PriceSource string

const (
	SourceCrowdsource PriceSource = "crowdsource"
	SourceAIEstimate  PriceSource = "ai_estimate"
	SourceScrape      PriceSource = "scrape"
	SourceAPI         PriceSource = "api"
)

type IngredientPrice struct {
	IngredientName  string      `json:"ingredient_name"`
	CityID          *int        `json:"city_id,omitempty"`
	Price           int         `json:"price"` // Rupiah integer
	Source          PriceSource `json:"source"`
	ConfidenceScore float64     `json:"confidence_score"`
}

type PriceProvider interface {
	GetIngredientPrice(ctx context.Context, name string, cityID *int) (*IngredientPrice, error)
	GetIngredientPricesBatch(ctx context.Context, names []string, cityID *int) (map[string]*IngredientPrice, error)
}
