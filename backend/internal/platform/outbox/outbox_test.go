package outbox

import (
	"errors"
	"testing"
	"time"
)

func TestEventAndClaimValidationAreDomainNeutralAndBounded(t *testing.T) {
	now := time.Now().UTC()
	event := Event{
		ID: [16]byte{1}, Type: "transaction.recorded.v1", Version: 1,
		AggregateType: "portfolio", AggregateID: [16]byte{2}, PortfolioID: [16]byte{3},
		OccurredAt: now, CorrelationID: "corr-outbox", Payload: Payload{SchemaVersion: 1},
		NextAttemptAt: now,
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("validate event: %v", err)
	}
	event.Version = 0
	if !errors.Is(event.Validate(), ErrInvalidEvent) {
		t.Fatal("non-positive event version was accepted")
	}
	event.Version = 1
	event.Type = "transaction recorded"
	if !errors.Is(event.Validate(), ErrInvalidEvent) {
		t.Fatal("unbounded event type was accepted")
	}

	request := ClaimRequest{AsOf: now, ClaimToken: [16]byte{4}, LeaseExpiresAt: now.Add(time.Minute), BatchLimit: 1}
	if err := request.Validate(); err != nil {
		t.Fatalf("validate claim: %v", err)
	}
	request.BatchLimit = MaximumClaimBatchSize + 1
	if !errors.Is(request.Validate(), ErrInvalidClaimRequest) {
		t.Fatal("oversized claim batch was accepted")
	}
	if err := ValidateFailureCode("transient_delivery_failure"); err != nil {
		t.Fatalf("validate failure code: %v", err)
	}
	if !errors.Is(ValidateFailureCode("unsafe failure text"), ErrInvalidFailureCode) {
		t.Fatal("unsafe failure text was accepted")
	}
	if err := ValidateConsumerName("holding_projection_v1"); err != nil {
		t.Fatalf("validate consumer name: %v", err)
	}
	if !errors.Is(ValidateConsumerName("Holding Projection"), ErrInvalidConsumerName) {
		t.Fatal("unsafe consumer name was accepted")
	}
}
