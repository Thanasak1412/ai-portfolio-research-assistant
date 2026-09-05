// Package outbox defines domain-neutral durable-event persistence contracts.
// It contains no Transaction business logic, database driver types, or worker
// delivery loop.
package outbox

import (
	"context"
	"errors"
	"regexp"
	"time"
)

const MaximumClaimBatchSize int32 = 100

type Payload struct {
	SchemaVersion int `json:"schemaVersion"`
}

// Event contains only durable routing and stable-reference fields. The
// authoritative aggregate is re-read by future consumers; financial command
// bodies and provider data are deliberately not event payload fields.
type Event struct {
	ID            [16]byte
	Type          string
	Version       int32
	AggregateType string
	AggregateID   [16]byte
	PortfolioID   [16]byte
	TransactionID *[16]byte
	CorrectionID  *[16]byte
	OccurredAt    time.Time
	CorrelationID string
	Payload       Payload
	NextAttemptAt time.Time
}

type ClaimedEvent struct {
	Event
	AttemptCount   int32
	ClaimToken     [16]byte
	ClaimedAt      time.Time
	LeaseExpiresAt time.Time
}

type ClaimRequest struct {
	AsOf           time.Time
	ClaimToken     [16]byte
	LeaseExpiresAt time.Time
	BatchLimit     int32
}

type Appender interface {
	Append(context.Context, Event) error
}

type DeliveryStore interface {
	Appender
	ClaimDue(context.Context, ClaimRequest) ([]ClaimedEvent, error)
	MarkPublished(context.Context, [16]byte, [16]byte, time.Time) (bool, error)
	Reschedule(context.Context, [16]byte, [16]byte, time.Time, string) (bool, error)
	MarkDeadLetter(context.Context, [16]byte, [16]byte, string) (bool, error)
}

type ConsumerDeduplicator interface {
	RecordIfNew(context.Context, string, [16]byte, time.Time) (bool, error)
}

var ErrInvalidEvent = errors.New("invalid platform outbox event")
var ErrInvalidClaimRequest = errors.New("invalid platform outbox claim request")
var ErrInvalidFailureCode = errors.New("invalid platform outbox failure code")
var ErrInvalidConsumerName = errors.New("invalid platform consumer name")

var eventTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}$`)
var aggregateTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
var failureCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
var consumerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)

func (event Event) Validate() error {
	if event.ID == [16]byte{} || event.Version <= 0 || event.AggregateID == [16]byte{} ||
		event.PortfolioID == [16]byte{} || event.OccurredAt.IsZero() || event.NextAttemptAt.IsZero() ||
		event.Payload.SchemaVersion <= 0 || len(event.Type) > 128 ||
		!eventTypePattern.MatchString(event.Type) || !aggregateTypePattern.MatchString(event.AggregateType) ||
		!correlationIDPattern.MatchString(event.CorrelationID) {
		return ErrInvalidEvent
	}
	for _, reference := range []*[16]byte{event.TransactionID, event.CorrectionID} {
		if reference != nil && *reference == [16]byte{} {
			return ErrInvalidEvent
		}
	}
	return nil
}

func (request ClaimRequest) Validate() error {
	if request.AsOf.IsZero() || request.ClaimToken == [16]byte{} || request.LeaseExpiresAt.IsZero() ||
		!request.LeaseExpiresAt.After(request.AsOf) || request.BatchLimit < 1 ||
		request.BatchLimit > MaximumClaimBatchSize {
		return ErrInvalidClaimRequest
	}
	return nil
}

func ValidateFailureCode(value string) error {
	if !failureCodePattern.MatchString(value) {
		return ErrInvalidFailureCode
	}
	return nil
}

func ValidateConsumerName(value string) error {
	if !consumerNamePattern.MatchString(value) {
		return ErrInvalidConsumerName
	}
	return nil
}
