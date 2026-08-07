package job

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConsensusSummary struct {
	GroupsProcessed int `json:"groups_processed"`
	ValidatedCount  int `json:"validated_count"`
	RejectedCount   int `json:"rejected_count"`
}

type ConsensusJob struct {
	db *pgxpool.Pool
}

func NewConsensusJob(db *pgxpool.Pool) *ConsensusJob {
	return &ConsensusJob{db: db}
}

type pendingSub struct {
	id             int
	userID         string
	submittedPrice int
}

func (j *ConsensusJob) RunConsensusValidation(ctx context.Context, minSubmissions int, tolerancePercent float64) (*ConsensusSummary, error) {
	if j.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if minSubmissions <= 0 {
		minSubmissions = 3
	}
	if tolerancePercent <= 0 {
		tolerancePercent = 15.0 // default 15% tolerance
	}

	// 1. Find groups with enough pending submissions
	groupQuery := `
		SELECT watch_item_id, city_id, COUNT(*)
		FROM price_submissions
		WHERE status = 'pending'
		GROUP BY watch_item_id, city_id
		HAVING COUNT(*) >= $1
	`

	rows, err := j.db.Query(ctx, groupQuery, minSubmissions)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending submission groups: %w", err)
	}

	type groupKey struct {
		watchItemID int
		cityID      int
	}

	var targetGroups []groupKey
	for rows.Next() {
		var gk groupKey
		var count int
		if err := rows.Scan(&gk.watchItemID, &gk.cityID, &count); err == nil {
			targetGroups = append(targetGroups, gk)
		}
	}
	rows.Close()

	summary := &ConsensusSummary{}

	// 2. Process each qualified group
	for _, gk := range targetGroups {
		// Fetch ingredient_name from price_watch_items
		var ingredientName string
		err := j.db.QueryRow(ctx, "SELECT ingredient_name FROM price_watch_items WHERE id = $1", gk.watchItemID).Scan(&ingredientName)
		if err != nil {
			continue
		}

		// Fetch all pending submissions in group
		subQuery := `
			SELECT id, user_id, submitted_price
			FROM price_submissions
			WHERE watch_item_id = $1 AND city_id = $2 AND status = 'pending'
			ORDER BY submitted_price ASC
		`
		subRows, err := j.db.Query(ctx, subQuery, gk.watchItemID, gk.cityID)
		if err != nil {
			continue
		}

		var subs []pendingSub
		for subRows.Next() {
			var ps pendingSub
			if err := subRows.Scan(&ps.id, &ps.userID, &ps.submittedPrice); err == nil {
				subs = append(subs, ps)
			}
		}
		subRows.Close()

		if len(subs) < minSubmissions {
			continue
		}

		summary.GroupsProcessed++

		// Calculate Median Price
		prices := make([]int, len(subs))
		for i, s := range subs {
			prices[i] = s.submittedPrice
		}
		sort.Ints(prices)

		var median float64
		n := len(prices)
		if n%2 == 1 {
			median = float64(prices[n/2])
		} else {
			median = float64(prices[n/2-1]+prices[n/2]) / 2.0
		}

		lowerBound := median * (1.0 - (tolerancePercent / 100.0))
		upperBound := median * (1.0 + (tolerancePercent / 100.0))

		// Evaluate each submission
		for _, s := range subs {
			val := float64(s.submittedPrice)
			if val >= lowerBound && val <= upperBound {
				// Mark validated
				if _, err := j.db.Exec(ctx, "UPDATE price_submissions SET status = 'validated', validated_at = NOW() WHERE id = $1", s.id); err != nil {
					fmt.Printf("Err update: %v\n", err)
				}

				// Log to ingredient_price_log
				if _, err := j.db.Exec(ctx, `
					INSERT INTO ingredient_price_log (ingredient_name, city_id, price, source, confidence_score)
					VALUES ($1, $2, $3, 'crowdsource', 0.9)
				`, ingredientName, gk.cityID, s.submittedPrice); err != nil {
					fmt.Printf("Err log: %v\n", err)
				}

				// Award credit
				if _, err := j.db.Exec(ctx, `
					INSERT INTO user_credits (user_id, balance, updated_at)
					VALUES ($1, 1, NOW())
					ON CONFLICT (user_id) DO UPDATE SET balance = user_credits.balance + 1, updated_at = NOW()
				`, s.userID); err != nil {
					fmt.Printf("Err credit: %v\n", err)
				}

				// Record credit transaction
				if _, err := j.db.Exec(ctx, `
					INSERT INTO credit_transactions (user_id, amount, type, reference_id)
					VALUES ($1, 1, 'earn_submission', $2)
				`, s.userID, s.id); err != nil {
					fmt.Printf("Err credit_tx: %v\n", err)
				}

				summary.ValidatedCount++
			} else {
				// Mark rejected
				_, _ = j.db.Exec(ctx, "UPDATE price_submissions SET status = 'rejected' WHERE id = $1", s.id)
				summary.RejectedCount++
			}
		}
	}

	return summary, nil
}
