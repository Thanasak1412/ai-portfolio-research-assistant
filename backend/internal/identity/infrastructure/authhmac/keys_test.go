package authhmac

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

func TestParsePairAcceptsIndependentCanonicalKeys(t *testing.T) {
	networkEncoded := encodedKey(1)
	rateEncoded := encodedKey(65)
	network, rate, err := ParsePair(networkEncoded, rateEncoded)
	if err != nil || len(network.Bytes()) != KeyLength || len(rate.Bytes()) != KeyLength {
		t.Fatalf("parse HMAC key pair: network=%v rate=%v err=%v", network, rate, err)
	}
	if fmt.Sprint(network) != "[REDACTED]" || fmt.Sprintf("%#v", rate) != "authhmac.RateLimitKey{[REDACTED]}" {
		t.Fatal("HMAC key formatting was not redacted")
	}
}

func TestLoadUsesOnlyApprovedEnvironmentNames(t *testing.T) {
	values := map[string]string{
		"AUTH_NETWORK_HMAC_KEY":    encodedKey(1),
		"AUTH_RATE_LIMIT_HMAC_KEY": encodedKey(65),
	}
	network, rate, err := Load(func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	})
	if err != nil || network.IsZero() || rate.IsZero() {
		t.Fatalf("load HMAC secrets: %v", err)
	}
	if _, _, err := Load(nil); !errors.Is(err, application.ErrInvalidSecurityConfig) {
		t.Fatalf("nil lookup accepted: %v", err)
	}
}

func TestParsePairRejectsUnsafeRepresentationsWithoutLeakage(t *testing.T) {
	canonical := encodedKey(1)
	tests := []struct {
		name    string
		network string
		rate    string
	}{
		{name: "empty", network: "", rate: encodedKey(65)},
		{name: "malformed", network: "not-base64!", rate: encodedKey(65)},
		{name: "wrong length", network: base64.StdEncoding.EncodeToString(make([]byte, 31)), rate: encodedKey(65)},
		{name: "noncanonical whitespace", network: canonical + "\n", rate: encodedKey(65)},
		{name: "reused key", network: canonical, rate: canonical},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := ParsePair(test.network, test.rate)
			if !errors.Is(err, application.ErrInvalidSecurityConfig) {
				t.Fatalf("unsafe key representation accepted: %v", err)
			}
			if (test.network != "" && strings.Contains(err.Error(), test.network)) ||
				(test.rate != "" && strings.Contains(err.Error(), test.rate)) {
				t.Fatalf("secret leaked through validation error: %v", err)
			}
		})
	}
}

func encodedKey(seed byte) string {
	value := make([]byte, KeyLength)
	for index := range value {
		value[index] = seed + byte(index)
	}
	return base64.StdEncoding.EncodeToString(value)
}
