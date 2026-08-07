package model

import (
	"encoding/json"
	"time"
)

type UserPreference struct {
	ID            int             `json:"id"`
	UserID        string          `json:"user_id"`
	Goal          string          `json:"goal"`           // "hemat" | "sehat" | "diet" | "bebas"
	BudgetAmount  int             `json:"budget_amount"`  // rupiah integer
	BudgetPeriod  string          `json:"budget_period"`  // "harian" | "mingguan"
	HouseholdSize int             `json:"household_size"` // default 1
	Restrictions  json.RawMessage `json:"restrictions"`   // e.g. ["udang", "kacang"]
	UpdatedAt     time.Time       `json:"updated_at"`
}
