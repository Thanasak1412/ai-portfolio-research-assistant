package domain

import (
	"fmt"

	"github.com/google/uuid"
)

type UserID struct{ value uuid.UUID }

func NewUserID(value uuid.UUID) (UserID, error) {
	if value == uuid.Nil {
		return UserID{}, ErrInvalidUserID
	}
	return UserID{value: value}, nil
}

func ParseUserID(value string) (UserID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return UserID{}, fmt.Errorf("%w", ErrInvalidUserID)
	}
	return NewUserID(parsed)
}

func (id UserID) IsZero() bool    { return id.value == uuid.Nil }
func (id UserID) String() string  { return id.value.String() }
func (id UserID) Bytes() [16]byte { return id.value }

type SessionID struct{ value uuid.UUID }

func NewSessionID(value uuid.UUID) (SessionID, error) {
	if value == uuid.Nil {
		return SessionID{}, ErrInvalidSessionID
	}
	return SessionID{value: value}, nil
}

func ParseSessionID(value string) (SessionID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return SessionID{}, fmt.Errorf("%w", ErrInvalidSessionID)
	}
	return NewSessionID(parsed)
}

func (id SessionID) IsZero() bool    { return id.value == uuid.Nil }
func (id SessionID) String() string  { return id.value.String() }
func (id SessionID) Bytes() [16]byte { return id.value }

type TokenFamilyID struct{ value uuid.UUID }

func NewTokenFamilyID(value uuid.UUID) (TokenFamilyID, error) {
	if value == uuid.Nil {
		return TokenFamilyID{}, ErrInvalidTokenFamilyID
	}
	return TokenFamilyID{value: value}, nil
}

func ParseTokenFamilyID(value string) (TokenFamilyID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return TokenFamilyID{}, fmt.Errorf("%w", ErrInvalidTokenFamilyID)
	}
	return NewTokenFamilyID(parsed)
}

func (id TokenFamilyID) IsZero() bool    { return id.value == uuid.Nil }
func (id TokenFamilyID) String() string  { return id.value.String() }
func (id TokenFamilyID) Bytes() [16]byte { return id.value }
