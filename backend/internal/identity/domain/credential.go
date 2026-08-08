package domain

import (
	"fmt"
	"strings"
)

const TokenDigestLength = 32

type PasswordHash struct{ encoded string }

func NewPasswordHash(encoded string) (PasswordHash, error) {
	if strings.TrimSpace(encoded) == "" {
		return PasswordHash{}, ErrInvalidPasswordHash
	}
	return PasswordHash{encoded: encoded}, nil
}

func (hash PasswordHash) Encoded() string  { return hash.encoded }
func (hash PasswordHash) IsZero() bool     { return hash.encoded == "" }
func (hash PasswordHash) String() string   { return "[REDACTED]" }
func (hash PasswordHash) GoString() string { return "domain.PasswordHash{[REDACTED]}" }

type TokenDigest struct {
	value [TokenDigestLength]byte
	valid bool
}

func NewTokenDigest(value []byte) (TokenDigest, error) {
	if len(value) != TokenDigestLength {
		return TokenDigest{}, ErrInvalidTokenDigest
	}
	var digest TokenDigest
	copy(digest.value[:], value)
	digest.valid = true
	return digest, nil
}

func (digest TokenDigest) Bytes() []byte {
	result := make([]byte, TokenDigestLength)
	copy(result, digest.value[:])
	return result
}

func (digest TokenDigest) IsZero() bool {
	return !digest.valid
}

func (digest TokenDigest) String() string   { return "[REDACTED]" }
func (digest TokenDigest) GoString() string { return "domain.TokenDigest{[REDACTED]}" }

var _ fmt.Stringer = PasswordHash{}
var _ fmt.Stringer = TokenDigest{}
