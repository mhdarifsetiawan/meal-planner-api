package payment

import "context"

type PaymentStatus string

const (
	StatusPending PaymentStatus = "pending"
	StatusSuccess PaymentStatus = "success"
	StatusFailed  PaymentStatus = "failed"
)

type PaymentRequest struct {
	UserID        string `json:"user_id"`
	PlanID        int    `json:"plan_id"`
	Amount        int    `json:"amount"`         // Rupiah integer
	PaymentMethod string `json:"payment_method"` // e.g. "qris", "bank_transfer", "dummy"
}

type PaymentResponse struct {
	TransactionID string        `json:"transaction_id"`
	GatewayRef    string        `json:"gateway_ref"`
	Amount        int           `json:"amount"`
	Status        PaymentStatus `json:"status"`
	PaymentURL    string        `json:"payment_url,omitempty"`
}

type PaymentProvider interface {
	ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
	VerifyPayment(ctx context.Context, gatewayRef string) (*PaymentResponse, error)
}
