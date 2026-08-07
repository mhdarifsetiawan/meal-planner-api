package model

import (
	"encoding/json"
	"time"
)

type SubscriptionPlan struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"` // "free" | "premium"
	Price         int             `json:"price"`
	BillingPeriod *string         `json:"billing_period,omitempty"`
	Features      json.RawMessage `json:"features"`
	IsActive      bool            `json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
}

type UserSubscription struct {
	ID        int        `json:"id"`
	UserID    string     `json:"user_id"`
	PlanID    int        `json:"plan_id"`
	CouponID  *int       `json:"coupon_id,omitempty"`
	Status    string     `json:"status"` // "active" | "expired" | "canceled"
	StartedAt time.Time  `json:"started_at"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
}

type UserSubscriptionResult struct {
	SubscriptionID int        `json:"subscription_id"`
	PlanName       string     `json:"plan_name"`
	Status         string     `json:"status"`
	Amount         int        `json:"amount"`
	PaymentGateway string     `json:"payment_gateway"`
	GatewayRef     string     `json:"gateway_ref"`
	StartedAt      time.Time  `json:"started_at"`
	EndsAt         *time.Time `json:"ends_at,omitempty"`
}
