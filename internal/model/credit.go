package model

import "time"

type UserCredit struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Balance   int       `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreditTransaction struct {
	ID          int       `json:"id"`
	UserID      string    `json:"user_id"`
	Amount      int       `json:"amount"` // positive (earn) or negative (spend)
	Type        string    `json:"type"`   // e.g. "earn_submission", "spend_generate"
	ReferenceID *int      `json:"reference_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserCreditSummary struct {
	Balance      int                 `json:"balance"`
	Transactions []CreditTransaction `json:"transactions"`
}
