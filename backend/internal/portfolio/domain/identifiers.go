package domain

import (
	"fmt"

	"github.com/google/uuid"
)

// PortfolioID is the opaque, application-generated Portfolio identifier.
type PortfolioID struct{ value uuid.UUID }

func NewPortfolioID(value uuid.UUID) (PortfolioID, error) {
	if value == uuid.Nil {
		return PortfolioID{}, ErrInvalidPortfolioID
	}
	return PortfolioID{value: value}, nil
}

func ParsePortfolioID(value string) (PortfolioID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return PortfolioID{}, fmt.Errorf("%w", ErrInvalidPortfolioID)
	}
	return NewPortfolioID(parsed)
}

func (id PortfolioID) IsZero() bool    { return id.value == uuid.Nil }
func (id PortfolioID) String() string  { return id.value.String() }
func (id PortfolioID) Bytes() [16]byte { return id.value }
