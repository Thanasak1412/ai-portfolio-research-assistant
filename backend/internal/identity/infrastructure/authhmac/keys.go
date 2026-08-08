package authhmac

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

const KeyLength = 32

type NetworkKey struct {
	value [KeyLength]byte
	valid bool
}

type RateLimitKey struct {
	value [KeyLength]byte
	valid bool
}

type LookupFunc func(string) (string, bool)

// Load reads the two approved environment variables without trimming or
// normalizing their security-sensitive representations.
func Load(lookup LookupFunc) (NetworkKey, RateLimitKey, error) {
	if lookup == nil {
		return NetworkKey{}, RateLimitKey{}, invalidField("Authentication HMAC key lookup")
	}
	networkEncoded, _ := lookup("AUTH_NETWORK_HMAC_KEY")
	rateLimitEncoded, _ := lookup("AUTH_RATE_LIMIT_HMAC_KEY")
	return ParsePair(networkEncoded, rateLimitEncoded)
}

// ParsePair validates the two independent AUTH_HMAC_KEYS-v1 environment
// representations without retaining or returning their encoded forms.
func ParsePair(networkEncoded, rateLimitEncoded string) (NetworkKey, RateLimitKey, error) {
	networkBytes, err := parseCanonicalKey("AUTH_NETWORK_HMAC_KEY", networkEncoded)
	if err != nil {
		return NetworkKey{}, RateLimitKey{}, err
	}
	rateLimitBytes, err := parseCanonicalKey("AUTH_RATE_LIMIT_HMAC_KEY", rateLimitEncoded)
	if err != nil {
		return NetworkKey{}, RateLimitKey{}, err
	}
	if subtle.ConstantTimeCompare(networkBytes, rateLimitBytes) == 1 {
		return NetworkKey{}, RateLimitKey{}, invalidField("AUTH_NETWORK_HMAC_KEY and AUTH_RATE_LIMIT_HMAC_KEY")
	}

	var network NetworkKey
	copy(network.value[:], networkBytes)
	network.valid = true
	var rateLimit RateLimitKey
	copy(rateLimit.value[:], rateLimitBytes)
	rateLimit.valid = true
	return network, rateLimit, nil
}

func (key NetworkKey) Bytes() []byte {
	if !key.valid {
		return nil
	}
	result := make([]byte, KeyLength)
	copy(result, key.value[:])
	return result
}

func (key NetworkKey) IsZero() bool     { return !key.valid }
func (key NetworkKey) String() string   { return "[REDACTED]" }
func (key NetworkKey) GoString() string { return "authhmac.NetworkKey{[REDACTED]}" }

func (key RateLimitKey) Bytes() []byte {
	if !key.valid {
		return nil
	}
	result := make([]byte, KeyLength)
	copy(result, key.value[:])
	return result
}

func (key RateLimitKey) IsZero() bool     { return !key.valid }
func (key RateLimitKey) String() string   { return "[REDACTED]" }
func (key RateLimitKey) GoString() string { return "authhmac.RateLimitKey{[REDACTED]}" }

func parseCanonicalKey(field, encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, invalidField(field)
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(decoded) != KeyLength || base64.StdEncoding.EncodeToString(decoded) != encoded {
		return nil, invalidField(field)
	}
	return decoded, nil
}

func invalidField(field string) error {
	return errors.Join(application.ErrInvalidSecurityConfig, errors.New(field+" is invalid"))
}
