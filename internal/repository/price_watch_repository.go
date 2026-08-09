package repository

import (
	"context"
	"errors"
	"fmt"

	"meal-planner-api/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCampaignNotFound   = errors.New("price watch campaign not found")
	ErrItemNotFound       = errors.New("price watch item not found")
	ErrSubmissionNotFound = errors.New("price submission not found")
)

type PriceWatchRepository interface {
	CreateCampaign(ctx context.Context, campaign *model.PriceWatchCampaign) error
	GetCampaigns(ctx context.Context, includeInactive bool) ([]model.PriceWatchCampaign, error)
	GetCampaignByID(ctx context.Context, id int) (*model.PriceWatchCampaign, error)
	UpdateCampaign(ctx context.Context, campaign *model.PriceWatchCampaign) error
	DeleteCampaign(ctx context.Context, id int) error

	CreateItem(ctx context.Context, item *model.PriceWatchItem) error
	GetItemByID(ctx context.Context, id int) (*model.PriceWatchItem, error)
	UpdateItem(ctx context.Context, item *model.PriceWatchItem) error
	DeleteItem(ctx context.Context, id int) error

	CreateSubmission(ctx context.Context, sub *model.PriceSubmission) error
	GetActiveCampaignsWithItems(ctx context.Context) ([]model.PriceWatchCampaign, error)
	GetUserSubmissions(ctx context.Context, userID string) ([]model.UserPriceSubmissionDetail, error)
	GetAllSubmissions(ctx context.Context, cityID *int, status *string) ([]model.AdminSubmissionDetail, error)
	UpdateSubmissionStatus(ctx context.Context, id int, status string) error
	DeleteSubmission(ctx context.Context, id int) error
	HasRecentSubmission(ctx context.Context, userID string, watchItemID int, cityID int) (bool, error)
}

type pgxPriceWatchRepository struct {
	db *pgxpool.Pool
}

func NewPriceWatchRepository(db *pgxpool.Pool) PriceWatchRepository {
	return &pgxPriceWatchRepository{db: db}
}

func (r *pgxPriceWatchRepository) CreateCampaign(ctx context.Context, campaign *model.PriceWatchCampaign) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		INSERT INTO price_watch_campaigns (title, description, is_active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, campaign.Title, campaign.Description, campaign.IsActive, campaign.CreatedBy).
		Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert campaign: %w", err)
	}

	return nil
}

func (r *pgxPriceWatchRepository) GetCampaigns(ctx context.Context, includeInactive bool) ([]model.PriceWatchCampaign, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id, title, COALESCE(description, ''), is_active, created_by, created_at, updated_at
		FROM price_watch_campaigns
	`
	if !includeInactive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY id DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []model.PriceWatchCampaign
	for rows.Next() {
		var c model.PriceWatchCampaign
		err := rows.Scan(&c.ID, &c.Title, &c.Description, &c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan campaign: %w", err)
		}
		campaigns = append(campaigns, c)
	}

	if campaigns == nil {
		campaigns = []model.PriceWatchCampaign{}
	}

	return campaigns, nil
}

func (r *pgxPriceWatchRepository) GetCampaignByID(ctx context.Context, id int) (*model.PriceWatchCampaign, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id, title, COALESCE(description, ''), is_active, created_by, created_at, updated_at
		FROM price_watch_campaigns
		WHERE id = $1
	`

	var c model.PriceWatchCampaign
	err := r.db.QueryRow(ctx, query, id).
		Scan(&c.ID, &c.Title, &c.Description, &c.IsActive, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCampaignNotFound
		}
		return nil, fmt.Errorf("failed to get campaign by id: %w", err)
	}

	// Fetch items
	itemQuery := `
		SELECT id, campaign_id, ingredient_name, unit, icon_url, display_order, is_active
		FROM price_watch_items
		WHERE campaign_id = $1
		ORDER BY display_order ASC, id ASC
	`
	rows, err := r.db.Query(ctx, itemQuery, id)
	if err == nil {
		defer rows.Close()
		var items []model.PriceWatchItem
		for rows.Next() {
			var item model.PriceWatchItem
			_ = rows.Scan(&item.ID, &item.CampaignID, &item.IngredientName, &item.Unit, &item.IconURL, &item.DisplayOrder, &item.IsActive)
			items = append(items, item)
		}
		if items == nil {
			items = []model.PriceWatchItem{}
		}
		c.Items = items
	}

	return &c, nil
}

func (r *pgxPriceWatchRepository) UpdateCampaign(ctx context.Context, campaign *model.PriceWatchCampaign) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE price_watch_campaigns
		SET title = $1, description = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
	`

	cmdTag, err := r.db.Exec(ctx, query, campaign.Title, campaign.Description, campaign.IsActive, campaign.ID)
	if err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}

	return nil
}

func (r *pgxPriceWatchRepository) DeleteCampaign(ctx context.Context, id int) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM price_watch_campaigns WHERE id = $1`
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete campaign: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}

	return nil
}

func (r *pgxPriceWatchRepository) CreateItem(ctx context.Context, item *model.PriceWatchItem) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		INSERT INTO price_watch_items (campaign_id, ingredient_name, unit, icon_url, display_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query, item.CampaignID, item.IngredientName, item.Unit, item.IconURL, item.DisplayOrder, item.IsActive).
		Scan(&item.ID)
	if err != nil {
		return fmt.Errorf("failed to insert item: %w", err)
	}

	return nil
}

func (r *pgxPriceWatchRepository) UpdateItem(ctx context.Context, item *model.PriceWatchItem) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE price_watch_items
		SET ingredient_name = $1, unit = $2, icon_url = $3, display_order = $4, is_active = $5
		WHERE id = $6
	`

	cmdTag, err := r.db.Exec(ctx, query, item.IngredientName, item.Unit, item.IconURL, item.DisplayOrder, item.IsActive, item.ID)
	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrItemNotFound
	}

	return nil
}

func (r *pgxPriceWatchRepository) GetItemByID(ctx context.Context, id int) (*model.PriceWatchItem, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT id, campaign_id, ingredient_name, unit, icon_url, display_order, is_active
		FROM price_watch_items
		WHERE id = $1
	`

	var item model.PriceWatchItem
	err := r.db.QueryRow(ctx, query, id).
		Scan(&item.ID, &item.CampaignID, &item.IngredientName, &item.Unit, &item.IconURL, &item.DisplayOrder, &item.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get price watch item: %w", err)
	}

	return &item, nil
}

func (r *pgxPriceWatchRepository) DeleteItem(ctx context.Context, id int) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM price_watch_items WHERE id = $1`
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrItemNotFound
	}

	return nil
}

func (r *pgxPriceWatchRepository) CreateSubmission(ctx context.Context, sub *model.PriceSubmission) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		INSERT INTO price_submissions (watch_item_id, user_id, city_id, submitted_price, status, created_at)
		VALUES ($1, $2, $3, $4, 'pending', NOW())
		RETURNING id, created_at
	`

	err := r.db.QueryRow(ctx, query, sub.WatchItemID, sub.UserID, sub.CityID, sub.SubmittedPrice).
		Scan(&sub.ID, &sub.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert price submission: %w", err)
	}
	sub.Status = "pending"

	return nil
}

func (r *pgxPriceWatchRepository) GetActiveCampaignsWithItems(ctx context.Context) ([]model.PriceWatchCampaign, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	campaigns, err := r.GetCampaigns(ctx, false) // is_active = true only
	if err != nil {
		return nil, err
	}

	for i := range campaigns {
		c := &campaigns[i]
		itemQuery := `
			SELECT id, campaign_id, ingredient_name, unit, icon_url, display_order, is_active
			FROM price_watch_items
			WHERE campaign_id = $1 AND is_active = true
			ORDER BY display_order ASC, id ASC
		`
		rows, err := r.db.Query(ctx, itemQuery, c.ID)
		if err == nil {
			var items []model.PriceWatchItem
			for rows.Next() {
				var item model.PriceWatchItem
				_ = rows.Scan(&item.ID, &item.CampaignID, &item.IngredientName, &item.Unit, &item.IconURL, &item.DisplayOrder, &item.IsActive)
				items = append(items, item)
			}
			rows.Close()
			if items == nil {
				items = []model.PriceWatchItem{}
			}
			c.Items = items
		}
	}

	return campaigns, nil
}

func (r *pgxPriceWatchRepository) GetUserSubmissions(ctx context.Context, userID string) ([]model.UserPriceSubmissionDetail, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT ps.id, ps.watch_item_id, pwi.ingredient_name, pwi.unit, pwc.title,
		       ps.city_id, ps.submitted_price, ps.status, ps.validated_at, ps.created_at
		FROM price_submissions ps
		JOIN price_watch_items pwi ON pwi.id = ps.watch_item_id
		JOIN price_watch_campaigns pwc ON pwc.id = pwi.campaign_id
		WHERE ps.user_id = $1
		ORDER BY ps.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user submissions: %w", err)
	}
	defer rows.Close()

	var details []model.UserPriceSubmissionDetail
	for rows.Next() {
		var d model.UserPriceSubmissionDetail
		err := rows.Scan(
			&d.ID,
			&d.WatchItemID,
			&d.IngredientName,
			&d.Unit,
			&d.CampaignTitle,
			&d.CityID,
			&d.SubmittedPrice,
			&d.Status,
			&d.ValidatedAt,
			&d.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission detail: %w", err)
		}
		details = append(details, d)
	}

	if details == nil {
		details = []model.UserPriceSubmissionDetail{}
	}

	return details, nil
}

func (r *pgxPriceWatchRepository) GetAllSubmissions(ctx context.Context, cityID *int, status *string) ([]model.AdminSubmissionDetail, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT ps.id, ps.user_id, pwi.ingredient_name, pwi.unit,
		       c.name AS city_name, ps.city_id, ps.submitted_price, ps.status, ps.created_at
		FROM price_submissions ps
		JOIN price_watch_items pwi ON pwi.id = ps.watch_item_id
		JOIN cities c ON c.id = ps.city_id
		WHERE ($1::int IS NULL OR ps.city_id = $1)
		  AND ($2::text IS NULL OR ps.status = $2)
		ORDER BY ps.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, cityID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query all submissions: %w", err)
	}
	defer rows.Close()

	var details []model.AdminSubmissionDetail
	for rows.Next() {
		var d model.AdminSubmissionDetail
		err := rows.Scan(
			&d.ID,
			&d.UserID,
			&d.IngredientName,
			&d.Unit,
			&d.CityName,
			&d.CityID,
			&d.SubmittedPrice,
			&d.Status,
			&d.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin submission detail: %w", err)
		}
		details = append(details, d)
	}

	if details == nil {
		details = []model.AdminSubmissionDetail{}
	}

	return details, nil
}

func (r *pgxPriceWatchRepository) HasRecentSubmission(ctx context.Context, userID string, watchItemID int, cityID int) (bool, error) {
	if r.db == nil {
		return false, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT EXISTS (
			SELECT 1 FROM price_submissions
			WHERE user_id = $1 AND watch_item_id = $2 AND city_id = $3
			  AND created_at >= NOW() - INTERVAL '24 hours'
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID, watchItemID, cityID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check recent submission: %w", err)
	}

	return exists, nil
}

func (r *pgxPriceWatchRepository) UpdateSubmissionStatus(ctx context.Context, id int, status string) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE price_submissions
		SET status = $1,
		    validated_at = CASE WHEN $1 = 'validated' THEN NOW() ELSE validated_at END
		WHERE id = $2
	`

	cmdTag, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}

	return nil
}

func (r *pgxPriceWatchRepository) DeleteSubmission(ctx context.Context, id int) error {
	if r.db == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `DELETE FROM price_submissions WHERE id = $1`

	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete price submission: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrSubmissionNotFound
	}

	return nil
}
