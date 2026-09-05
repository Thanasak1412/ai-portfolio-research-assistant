package audit

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestM3AuditActionsAndSafeRecordBoundary(t *testing.T) {
	record := Record{
		EventID:       [16]byte{1},
		OccurredAt:    time.Now().UTC(),
		Result:        ResultSuccess,
		Severity:      SeverityInfo,
		CorrelationID: "corr-platform-audit",
	}
	for _, action := range []Action{
		ActionTransactionCreateSuccess,
		ActionTransactionCreateFailure,
		ActionTransactionIdempotentReplay,
		ActionTransactionIdempotencyConflict,
		ActionTransactionCorrectionInitiated,
		ActionTransactionCorrectionCompleted,
		ActionTransactionCorrectionRejected,
		ActionTransactionReversalCreated,
		ActionTransactionReplacementCreated,
		ActionTransactionOwnershipRejection,
	} {
		record.Action = action
		record.Result = expectedResult(action)
		if err := record.Validate(); err != nil {
			t.Fatalf("validate %q: %v", action, err)
		}
	}
	record.Action = ActionTransactionCreateSuccess
	record.Result = ResultFailure
	if !errors.Is(record.Validate(), ErrInvalidRecord) {
		t.Fatal("mismatched M3 action/result pair was accepted")
	}

	record.Action = "free_form_action"
	if !errors.Is(record.Validate(), ErrInvalidRecord) {
		t.Fatal("free-form action was accepted")
	}

	recordType := reflect.TypeOf(Record{})
	for _, forbidden := range []string{
		"Amount", "Quantity", "UnitPrice", "Fee", "Note", "ExternalReference",
		"RequestBody", "Metadata", "AuthorizationHeader", "AccessToken", "RefreshToken", "Credential",
	} {
		if _, exists := recordType.FieldByName(forbidden); exists {
			t.Fatalf("platform audit record exposes forbidden field %s", forbidden)
		}
	}
}

func expectedResult(action Action) Result {
	switch action {
	case ActionTransactionCreateFailure,
		ActionTransactionIdempotencyConflict,
		ActionTransactionCorrectionRejected,
		ActionTransactionOwnershipRejection:
		return ResultFailure
	default:
		return ResultSuccess
	}
}
