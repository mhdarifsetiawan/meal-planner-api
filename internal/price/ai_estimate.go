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

// Fallback catalog map for offline unit tests when DB pool is nil
var defaultPriceCatalog = map[string]int{
	"ayam":   35000,
	"daging": 120000,
	"tahu":   5000,
	"tempe":  6000,
	"telur":  28000,
	"beras":  15000,
	"minyak": 18000,
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
			FROM ingredient_price_log
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

	// 2. Fallback estimation based on master_ingredients baseline_price in DB
	lowerName := strings.ToLower(cleanName)
	estimatedPrice := 10000 // default fallback if ingredient is unknown
	confidence := 0.70

	if p.db != nil {
		masterQuery := `
			SELECT mi.baseline_price
			FROM master_ingredients mi
			LEFT JOIN ingredient_aliases ia ON ia.master_ingredient_id = mi.id
			WHERE LOWER(mi.name) = $1 OR LOWER(ia.alias_name) = $1
			   OR $1 LIKE '%' || LOWER(mi.name) || '%'
			LIMIT 1
		`
		var dbBaselinePrice int
		if err := p.db.QueryRow(ctx, masterQuery, lowerName).Scan(&dbBaselinePrice); err == nil && dbBaselinePrice > 0 {
			estimatedPrice = dbBaselinePrice
			confidence = 0.85
		}
	} else {
		// Offline catalog fallback
		for key, val := range defaultPriceCatalog {
			if strings.Contains(lowerName, key) {
				estimatedPrice = val
				confidence = 0.85
				break
			}
		}
	}

	// 3. Log newly estimated price into DB if available
	if p.db != nil {
		insertQuery := `
			INSERT INTO ingredient_price_log (ingredient_name, city_id, price, source, confidence_score, recorded_at)
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

func (p *AIEstimateProvider) GetIngredientPricesBatch(ctx context.Context, names []string, cityID *int) (map[string]*IngredientPrice, error) {
	result := make(map[string]*IngredientPrice)
	if len(names) == 0 {
		return result, nil
	}

	// Deduplicate names
	for _, name := range names {
		clean := strings.TrimSpace(name)
		if clean == "" {
			continue
		}
		if _, exists := result[strings.ToLower(clean)]; !exists {
			pRes, err := p.GetIngredientPrice(ctx, clean, cityID)
			if err == nil && pRes != nil {
				result[strings.ToLower(clean)] = pRes
			}
		}
	}

	return result, nil
}
