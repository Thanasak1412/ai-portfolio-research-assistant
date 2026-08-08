package ratelimit

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/netip"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/authhmac"
	"github.com/google/uuid"
)

func TestKeyDeriverMatchesPolicyAndSeparatesNamespaces(t *testing.T) {
	deriver, secret := testKeyDeriver(t)
	email, err := domain.NormalizeEmail(" User+alias@Example.COM ")
	if err != nil {
		t.Fatal(err)
	}
	emailKey, err := deriver.LoginEmailFailure(email)
	if err != nil {
		t.Fatal(err)
	}
	want := expectedRateLimitHMAC(secret, application.RateLimitLoginEmailFailure, "user+alias@example.com")
	if !bytes.Equal(emailKey.Bytes(), want) {
		t.Fatal("email key does not match AUTH_HMAC_KEYS-v1")
	}

	address := netip.MustParseAddr("::ffff:192.0.2.10")
	loginIP, err := deriver.LoginIPAttempt(address)
	if err != nil {
		t.Fatal(err)
	}
	registrationIP, err := deriver.RegistrationIPAttempt(address)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(loginIP.Bytes(), registrationIP.Bytes()) {
		t.Fatal("rate-limit policy namespaces were not separated")
	}
	canonicalLoginIP, err := deriver.LoginIPAttempt(netip.MustParseAddr("192.0.2.10"))
	if err != nil || !bytes.Equal(loginIP.Bytes(), canonicalLoginIP.Bytes()) {
		t.Fatal("equivalent IP forms produced different rate-limit keys")
	}

	familyID, _ := domain.NewTokenFamilyID(uuid.MustParse("11111111-2222-4333-8444-555555555555"))
	familyKey, err := deriver.RefreshFamilyAttempt(familyID)
	if err != nil {
		t.Fatal(err)
	}
	familyWant := expectedRateLimitHMAC(secret, application.RateLimitRefreshFamilyAttempt, familyID.String())
	if !bytes.Equal(familyKey.Bytes(), familyWant) {
		t.Fatal("family key does not use canonical lowercase UUID")
	}
}

func TestKeyDeriverRejectsInvalidCanonicalIdentity(t *testing.T) {
	if _, err := NewKeyDeriver(authhmac.RateLimitKey{}); !errors.Is(err, application.ErrInvalidSecurityConfig) {
		t.Fatalf("zero HMAC key accepted: %v", err)
	}
	deriver, _ := testKeyDeriver(t)
	if _, err := deriver.LoginIPAttempt(netip.Addr{}); !errors.Is(err, application.ErrInvalidRateLimitKey) {
		t.Fatalf("invalid IP accepted: %v", err)
	}
	if _, err := deriver.LoginEmailFailure(domain.NormalizedEmail{}); !errors.Is(err, application.ErrInvalidRateLimitKey) {
		t.Fatalf("zero email accepted: %v", err)
	}
	if _, err := deriver.RefreshFamilyAttempt(domain.TokenFamilyID{}); !errors.Is(err, application.ErrInvalidRateLimitKey) {
		t.Fatalf("zero family accepted: %v", err)
	}
}

func testKeyDeriver(t *testing.T) (*KeyDeriver, []byte) {
	t.Helper()
	networkSecret := make([]byte, authhmac.KeyLength)
	rateSecret := make([]byte, authhmac.KeyLength)
	for index := range networkSecret {
		networkSecret[index] = byte(index + 1)
		rateSecret[index] = byte(index + 65)
	}
	_, rateKey, err := authhmac.ParsePair(
		base64.StdEncoding.EncodeToString(networkSecret),
		base64.StdEncoding.EncodeToString(rateSecret),
	)
	if err != nil {
		t.Fatal(err)
	}
	deriver, err := NewKeyDeriver(rateKey)
	if err != nil {
		t.Fatal(err)
	}
	return deriver, rateSecret
}

func expectedRateLimitHMAC(secret []byte, policy application.RateLimitPolicy, identity string) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte("auth_rate_limit\x00v1\x00"))
	_, _ = mac.Write([]byte(policy))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(application.RateLimitPolicyVersion))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(identity))
	return mac.Sum(nil)
}
