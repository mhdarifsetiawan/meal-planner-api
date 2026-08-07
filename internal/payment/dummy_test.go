package payment

import (
	"context"
	"testing"
	"time"
)

func TestDummyPaymentProvider_ProcessAndVerify(t *testing.T) {
	provider := NewDummyPaymentProvider(10 * time.Millisecond)
	ctx := context.Background()

	req := PaymentRequest{
		UserID:        "user-uuid-123",
		PlanID:        2,
		Amount:        29000,
		PaymentMethod: "qris",
	}

	resp, err := provider.ProcessPayment(ctx, req)
	if err != nil {
		t.Fatalf("ProcessPayment failed: %v", err)
	}

	if resp.Status != StatusSuccess {
		t.Errorf("Expected status 'success', got '%s'", resp.Status)
	}

	if resp.Amount != 29000 {
		t.Errorf("Expected amount 29000, got %d", resp.Amount)
	}

	if resp.GatewayRef == "" {
		t.Error("Expected non-empty gateway_ref")
	}

	// Verify Payment
	verifyResp, err := provider.VerifyPayment(ctx, resp.GatewayRef)
	if err != nil {
		t.Fatalf("VerifyPayment failed: %v", err)
	}

	if verifyResp.Status != StatusSuccess {
		t.Errorf("Expected verify status 'success', got '%s'", verifyResp.Status)
	}
}
