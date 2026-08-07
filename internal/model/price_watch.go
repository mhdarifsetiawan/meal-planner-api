package model

import "time"

type PriceWatchCampaign struct {
	ID          int              `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	IsActive    bool             `json:"is_active"`
	CreatedBy   string           `json:"created_by"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Items       []PriceWatchItem `json:"items,omitempty"`
}

type PriceWatchItem struct {
	ID             int     `json:"id"`
	CampaignID     int     `json:"campaign_id"`
	IngredientName string  `json:"ingredient_name"`
	Unit           string  `json:"unit"`
	IconURL        *string `json:"icon_url,omitempty"`
	DisplayOrder   int     `json:"display_order"`
	IsActive       bool    `json:"is_active"`
}
