package network

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/netip"
	"strings"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/authhmac"
)

func TestIdentityHasherMatchesPolicyAndCanonicalizesAddresses(t *testing.T) {
	networkSecret := make([]byte, authhmac.KeyLength)
	rateSecret := make([]byte, authhmac.KeyLength)
	for index := range networkSecret {
		networkSecret[index] = byte(index + 1)
		rateSecret[index] = byte(index + 65)
	}
	networkKey, _, err := authhmac.ParsePair(
		base64.StdEncoding.EncodeToString(networkSecret),
		base64.StdEncoding.EncodeToString(rateSecret),
	)
	if err != nil {
		t.Fatal(err)
	}
	hasher, err := NewIdentityHasher(networkKey)
	if err != nil {
		t.Fatal(err)
	}

	canonical := netip.MustParseAddr("192.0.2.10")
	mapped := netip.MustParseAddr("::ffff:192.0.2.10")
	got, err := hasher.Hash(mapped)
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, networkSecret)
	_, _ = mac.Write([]byte("ip_hmac\x00v1\x00"))
	_, _ = mac.Write([]byte(canonical.String()))
	want := "ip_hmac_v1:" + hex.EncodeToString(mac.Sum(nil))
	if got != want || len(got) != 75 || got != strings.ToLower(got) {
		t.Fatalf("network identity mismatch: got=%q want=%q", got, want)
	}
	canonicalResult, err := hasher.Hash(canonical)
	if err != nil || canonicalResult != got {
		t.Fatal("equivalent address forms produced different identities")
	}
	if strings.Contains(got, canonical.String()) {
		t.Fatal("raw IP leaked into persisted network identity")
	}
}

func TestIdentityHasherRejectsInvalidInput(t *testing.T) {
	if _, err := NewIdentityHasher(authhmac.NetworkKey{}); !errors.Is(err, application.ErrInvalidSecurityConfig) {
		t.Fatalf("zero HMAC key accepted: %v", err)
	}
	hasher := &IdentityHasher{}
	if _, err := hasher.Hash(netip.Addr{}); !errors.Is(err, application.ErrClientIdentityRejected) {
		t.Fatalf("invalid address accepted: %v", err)
	}
}
