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

type PriceSubmission struct {
	ID             int        `json:"id"`
	WatchItemID    int        `json:"watch_item_id"`
	UserID         string     `json:"user_id"`
	CityID         int        `json:"city_id"`
	SubmittedPrice int        `json:"submitted_price"`
	Status         string     `json:"status"` // "pending" | "validated" | "rejected"
	ValidatedAt    *time.Time `json:"validated_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type UserPriceSubmissionDetail struct {
	ID             int        `json:"id"`
	WatchItemID    int        `json:"watch_item_id"`
	IngredientName string     `json:"ingredient_name"`
	Unit           string     `json:"unit"`
	CampaignTitle  string     `json:"campaign_title"`
	CityID         int        `json:"city_id"`
	SubmittedPrice int        `json:"submitted_price"`
	Status         string     `json:"status"`
	ValidatedAt    *time.Time `json:"validated_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type AdminSubmissionDetail struct {
	ID             int       `json:"id"`
	UserID         string    `json:"user_id"`
	IngredientName string    `json:"ingredient_name"`
	Unit           string    `json:"unit"`
	CityName       string    `json:"city_name"`
	CityID         int       `json:"city_id"`
	SubmittedPrice int       `json:"submitted_price"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}
