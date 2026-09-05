// Package outbox defines domain-neutral durable-event persistence contracts.
// It contains no Transaction business logic, database driver types, or worker
// delivery loop.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

const MaximumClaimBatchSize int32 = 100
const MaximumPayloadReferences = 8

type Payload struct {
	SchemaVersion int         `json:"schemaVersion"`
	References    []Reference `json:"references"`
}

// Reference is a bounded, domain-neutral stable identifier. Platform records
// its role and UUID, but does not attach business semantics to either.
type Reference struct {
	Role string
	ID   [16]byte
}

func (reference Reference) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	}{Role: reference.Role, ID: uuid.UUID(reference.ID[:]).String()})
}

func (reference *Reference) UnmarshalJSON(data []byte) error {
	var encoded struct {
		Role string `json:"role"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal(data, &encoded); err != nil {
		return err
	}
	parsed, err := uuid.Parse(encoded.ID)
	if err != nil {
		return fmt.Errorf("parse Platform reference UUID: %w", err)
	}
	reference.Role = encoded.Role
	copy(reference.ID[:], parsed[:])
	return nil
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
	OccurredAt    time.Time
	CorrelationID string
	Payload       Payload
	NextAttemptAt time.Time
}

type ClaimedEvent struct {
	Event
	AggregatePosition int64
	AttemptCount      int32
	ClaimToken        [16]byte
	ClaimedAt         time.Time
	LeaseExpiresAt    time.Time
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
var referenceRolePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
var correlationIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
var failureCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
var consumerNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)

func (event Event) Validate() error {
	if event.ID == [16]byte{} || event.Version <= 0 || event.AggregateID == [16]byte{} ||
		event.PortfolioID == [16]byte{} || event.OccurredAt.IsZero() || event.NextAttemptAt.IsZero() ||
		len(event.Type) > 128 ||
		!eventTypePattern.MatchString(event.Type) || !aggregateTypePattern.MatchString(event.AggregateType) ||
		!correlationIDPattern.MatchString(event.CorrelationID) || event.Payload.Validate() != nil {
		return ErrInvalidEvent
	}
	return nil
}

func (payload Payload) Validate() error {
	if payload.SchemaVersion <= 0 || len(payload.References) > MaximumPayloadReferences {
		return ErrInvalidEvent
	}
	for _, reference := range payload.References {
		if !referenceRolePattern.MatchString(reference.Role) || reference.ID == [16]byte{} {
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
