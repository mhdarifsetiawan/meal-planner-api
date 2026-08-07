package price

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AIEstimateProvider struct {
	db *pgxpool.Pool
}

func NewAIEstimateProvider(db *pgxpool.Pool) *AIEstimateProvider {
	return &AIEstimateProvider{db: db}
}

// Standard fallback price catalog for common Indonesian grocery ingredients (in Rupiah)
var defaultPriceCatalog = map[string]int{
	"ayam":        35000,
	"daging":      120000,
	"daging sapi": 130000,
	"tahu":        5000,
	"tempe":       6000,
	"telur":       28000,
	"beras":       15000,
	"minyak":      18000,
	"cabai":       40000,
	"cabe":        40000,
	"bawang":      30000,
	"bawang merah": 32000,
	"bawang putih": 36000,
	"ikan":        35000,
	"udang":       80000,
	"cumi":        75000,
	"sayur":       5000,
	"bayam":       3000,
	"kangkung":    3000,
	"wortel":      10000,
	"kentang":     18000,
	"tomat":       12000,
}

func (p *AIEstimateProvider) GetIngredientPrice(ctx context.Context, name string, cityID *int) (*IngredientPrice, error) {
	cleanName := strings.TrimSpace(name)

	// 1. Query DB log if database connection pool is available
	if p.db != nil {
		var price int
		var sourceStr string
		var confidence float64

		query := `
			SELECT price, source, COALESCE(confidence_score, 0.8)
			FROM ingredient_price_logs
			WHERE LOWER(ingredient_name) = LOWER($1)
			  AND (city_id = $2 OR city_id IS NULL OR $2 IS NULL)
			ORDER BY recorded_at DESC
			LIMIT 1
		`
		err := p.db.QueryRow(ctx, query, cleanName, cityID).Scan(&price, &sourceStr, &confidence)
		if err == nil && price > 0 {
			return &IngredientPrice{
				IngredientName:  cleanName,
				CityID:          cityID,
				Price:           price,
				Source:          PriceSource(sourceStr),
				ConfidenceScore: confidence,
			}, nil
		}
	}

	// 2. Fallback estimation based on catalog lookup
	lowerName := strings.ToLower(cleanName)
	estimatedPrice := 10000 // default fallback
	confidence := 0.70

	for key, val := range defaultPriceCatalog {
		if strings.Contains(lowerName, key) {
			estimatedPrice = val
			confidence = 0.85
			break
		}
	}

	// 3. Log newly estimated price into DB if available
	if p.db != nil {
		insertQuery := `
			INSERT INTO ingredient_price_logs (ingredient_name, city_id, price, source, confidence_score, recorded_at)
			VALUES ($1, $2, $3, 'ai_estimate', $4, NOW())
		`
		_, _ = p.db.Exec(ctx, insertQuery, cleanName, cityID, estimatedPrice, confidence)
	}

	return &IngredientPrice{
		IngredientName:  cleanName,
		CityID:          cityID,
		Price:           estimatedPrice,
		Source:          SourceAIEstimate,
		ConfidenceScore: confidence,
	}, nil
}
