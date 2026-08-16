package domain

import (
	"fmt"

	"github.com/google/uuid"
)

// AssetID is the opaque canonical Asset identifier.
type AssetID struct{ value uuid.UUID }

func NewAssetID(value uuid.UUID) (AssetID, error) {
	if value == uuid.Nil {
		return AssetID{}, ErrInvalidAssetID
	}
	return AssetID{value: value}, nil
}

func ParseAssetID(value string) (AssetID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return AssetID{}, fmt.Errorf("%w", ErrInvalidAssetID)
	}
	return NewAssetID(parsed)
}

func (id AssetID) IsZero() bool    { return id.value == uuid.Nil }
func (id AssetID) String() string  { return id.value.String() }
func (id AssetID) Bytes() [16]byte { return id.value }
