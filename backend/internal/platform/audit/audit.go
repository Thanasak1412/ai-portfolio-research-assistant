// Package audit defines the narrow, Platform-owned boundary for append-only
// audit evidence. It deliberately exposes only approved safe identifiers.
package audit

import (
	"context"
	"errors"
	"regexp"
	"time"
)

type Action string

const (
	ActionTransactionCreateSuccess       Action = "transaction_create_success"
	ActionTransactionCreateFailure       Action = "transaction_create_failure"
	ActionTransactionIdempotentReplay    Action = "transaction_idempotent_replay"
	ActionTransactionIdempotencyConflict Action = "transaction_idempotency_conflict"
	ActionTransactionCorrectionInitiated Action = "transaction_correction_initiated"
	ActionTransactionCorrectionCompleted Action = "transaction_correction_completed"
	ActionTransactionCorrectionRejected  Action = "transaction_correction_rejected"
	ActionTransactionReversalCreated     Action = "transaction_reversal_created"
	ActionTransactionReplacementCreated  Action = "transaction_replacement_created"
	ActionTransactionOwnershipRejection  Action = "transaction_ownership_rejection"
)

type Result string

const (
	ResultSuccess Result = "success"
	ResultFailure Result = "failure"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityHigh    Severity = "high"
)

// Record intentionally has no financial values, request body, metadata, or
// credential-related fields. References may be supplied without foreign keys
// because the Transaction authority is introduced only in M3-DB-001.
type Record struct {
	EventID       [16]byte
	OccurredAt    time.Time
	Action        Action
	Result        Result
	Severity      Severity
	ActorUserID   *[16]byte
	CorrelationID string
	PortfolioID   *[16]byte
	TransactionID *[16]byte
	CorrectionID  *[16]byte
}

type Store interface {
	Append(context.Context, Record) error
}

var ErrInvalidRecord = errors.New("invalid platform audit record")

var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func (record Record) Validate() error {
	if record.EventID == [16]byte{} || record.OccurredAt.IsZero() ||
		!allowedAction(record.Action) || !allowedResult(record.Result) ||
		!allowedSeverity(record.Severity) || !correlationIDPattern.MatchString(record.CorrelationID) {
		return ErrInvalidRecord
	}
	for _, reference := range []*[16]byte{record.ActorUserID, record.PortfolioID, record.TransactionID, record.CorrectionID} {
		if reference != nil && *reference == [16]byte{} {
			return ErrInvalidRecord
		}
	}
	return nil
}

func allowedAction(value Action) bool {
	switch value {
	case ActionTransactionCreateSuccess,
		ActionTransactionCreateFailure,
		ActionTransactionIdempotentReplay,
		ActionTransactionIdempotencyConflict,
		ActionTransactionCorrectionInitiated,
		ActionTransactionCorrectionCompleted,
		ActionTransactionCorrectionRejected,
		ActionTransactionReversalCreated,
		ActionTransactionReplacementCreated,
		ActionTransactionOwnershipRejection:
		return true
	default:
		return false
	}
}

func allowedResult(value Result) bool {
	return value == ResultSuccess || value == ResultFailure
}

func allowedSeverity(value Severity) bool {
	return value == SeverityInfo || value == SeverityWarning || value == SeverityHigh
}
