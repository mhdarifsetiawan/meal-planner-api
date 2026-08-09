package model

import "time"

type AIProviderConfig struct {
	ID           int       `json:"id"`
	ProviderName string    `json:"provider_name"`
	ModelName    string    `json:"model_name"`
	IsActive     bool      `json:"is_active"`
	APIKeyRef    string    `json:"api_key_ref"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
	Description  string    `json:"description,omitempty"`
	Icon         string    `json:"icon,omitempty"`
}
