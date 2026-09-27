package domain

import (
	"github.com/google/uuid"
)

type TransactionID struct{ value uuid.UUID }
type CorrectionID struct{ value uuid.UUID }

func NewTransactionID(value uuid.UUID) (TransactionID, error) {
	if value == uuid.Nil {
		return TransactionID{}, ErrInvalidTransactionID
	}
	return TransactionID{value}, nil
}
func ParseTransactionID(value string) (TransactionID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil || parsed.String() != value {
		return TransactionID{}, ErrInvalidTransactionID
	}
	return NewTransactionID(parsed)
}
func (id TransactionID) IsZero() bool    { return id.value == uuid.Nil }
func (id TransactionID) String() string  { return id.value.String() }
func (id TransactionID) Bytes() [16]byte { return id.value }

func NewCorrectionID(value uuid.UUID) (CorrectionID, error) {
	if value == uuid.Nil {
		return CorrectionID{}, ErrInvalidCorrectionID
	}
	return CorrectionID{value}, nil
}
func ParseCorrectionID(value string) (CorrectionID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil || parsed.String() != value {
		return CorrectionID{}, ErrInvalidCorrectionID
	}
	return NewCorrectionID(parsed)
}
func (id CorrectionID) IsZero() bool    { return id.value == uuid.Nil }
func (id CorrectionID) String() string  { return id.value.String() }
func (id CorrectionID) Bytes() [16]byte { return id.value }
