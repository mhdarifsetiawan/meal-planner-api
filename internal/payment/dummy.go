package payment

import (
	"context"
	"fmt"
	"time"
)

type DummyPaymentProvider struct {
	simulatedDelay time.Duration
}

func NewDummyPaymentProvider(simulatedDelay time.Duration) *DummyPaymentProvider {
	return &DummyPaymentProvider{
		simulatedDelay: simulatedDelay,
	}
}

func (p *DummyPaymentProvider) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	if p.simulatedDelay > 0 {
		select {
		case <-time.After(p.simulatedDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	ref := fmt.Sprintf("dummy_ref_%d", time.Now().UnixNano())
	txID := fmt.Sprintf("tx_%d", time.Now().UnixNano())

	return &PaymentResponse{
		TransactionID: txID,
		GatewayRef:    ref,
		Amount:        req.Amount,
		Status:        StatusSuccess,
		PaymentURL:    "https://dummy-payment.example.com/pay/" + ref,
	}, nil
}

func (p *DummyPaymentProvider) VerifyPayment(ctx context.Context, gatewayRef string) (*PaymentResponse, error) {
	if gatewayRef == "" {
		return nil, fmt.Errorf("gateway reference cannot be empty")
	}

	return &PaymentResponse{
		GatewayRef: gatewayRef,
		Status:     StatusSuccess,
	}, nil
}
