package model

import "time"

type MasterIngredient struct {
	ID          int       `json:"id"`
	Category    string    `json:"category"`
	Name        string    `json:"name"`
	DefaultUnit string    `json:"default_unit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type IngredientAlias struct {
	ID                 int       `json:"id"`
	MasterIngredientID int       `json:"master_ingredient_id"`
	AliasName          string    `json:"alias_name"`
	CreatedAt          time.Time `json:"created_at"`
}

type MasterIngredientWithAliases struct {
	MasterIngredient
	Aliases []IngredientAlias `json:"aliases"`
}

type CreateMasterIngredientRequest struct {
	Category    string   `json:"category"`
	Name        string   `json:"name"`
	DefaultUnit string   `json:"default_unit"`
	Aliases     []string `json:"aliases,omitempty"`
}

type UpdateMasterIngredientRequest struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	DefaultUnit string `json:"default_unit"`
}

type AddAliasRequest struct {
	AliasName string `json:"alias_name"`
}
