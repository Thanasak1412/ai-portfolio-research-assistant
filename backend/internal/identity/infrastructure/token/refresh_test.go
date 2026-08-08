package token

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

func TestRefreshTokenGenerationParsingAndDigest(t *testing.T) {
	randomBytes := make([]byte, refreshTokenEntropyLength)
	for index := range randomBytes {
		randomBytes[index] = byte(index + 1)
	}
	adapter := newRefreshTokenAdapter(bytes.NewReader(randomBytes))
	token, err := adapter.Generate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expected := refreshTokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes)
	if token.Value() != expected || len(token.Value()) != refreshTokenLength {
		t.Fatalf("unexpected canonical representation")
	}
	parsed, err := adapter.Parse(expected)
	if err != nil || parsed.Value() != expected {
		t.Fatalf("parse canonical token: %v", err)
	}
	digest, err := adapter.Digest(parsed)
	if err != nil {
		t.Fatal(err)
	}
	want := sha256.Sum256([]byte(expected))
	if !bytes.Equal(digest.Bytes(), want[:]) || bytes.Contains(digest.Bytes(), []byte(expected)) {
		t.Fatal("refresh-token digest does not match REFRESH_TOKEN-v1")
	}
	if fmt.Sprint(token) != "[REDACTED]" || fmt.Sprintf("%#v", token) != "application.RefreshToken{[REDACTED]}" {
		t.Fatal("refresh token formatting was not redacted")
	}
	serialized, err := json.Marshal(token)
	if !errors.Is(err, application.ErrRefreshTokenRejected) || bytes.Contains(serialized, []byte(expected)) || strings.Contains(err.Error(), expected) {
		t.Fatalf("refresh token JSON serialization did not fail safely: bytes=%s err=%v", serialized, err)
	}
}

func TestRefreshTokenGenerationIsDistinct(t *testing.T) {
	adapter := NewRefreshTokenAdapter()
	first, err := adapter.Generate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Generate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Value() == second.Value() {
		t.Fatal("independent refresh tokens matched")
	}
}

func TestRefreshTokenRejectsMalformedValuesWithoutLeakage(t *testing.T) {
	validPayload := base64.RawURLEncoding.EncodeToString(make([]byte, refreshTokenEntropyLength))
	tests := []string{
		"", " " + refreshTokenPrefix + validPayload, refreshTokenPrefix + validPayload + " ",
		"rt_v2_" + validPayload, refreshTokenPrefix + validPayload + "=", refreshTokenPrefix + "bad!",
		refreshTokenPrefix + base64.RawURLEncoding.EncodeToString(make([]byte, refreshTokenEntropyLength-1)),
		refreshTokenPrefix + validPayload[:len(validPayload)-1] + "B",
	}
	adapter := NewRefreshTokenAdapter()
	for _, value := range tests {
		_, err := adapter.Parse(value)
		if !errors.Is(err, application.ErrRefreshTokenRejected) {
			t.Fatalf("malformed token accepted: %q", value)
		}
		if value != "" && strings.Contains(err.Error(), value) {
			t.Fatalf("token leaked through error: %v", err)
		}
	}
	unsafeToken, err := application.NewRefreshToken("not-a-canonical-token")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Digest(unsafeToken); !errors.Is(err, application.ErrRefreshTokenRejected) {
		t.Fatalf("noncanonical token was digested: %v", err)
	}
}

func TestRefreshTokenRandomFailureIsSafe(t *testing.T) {
	secret := "raw-refresh-token-material"
	adapter := newRefreshTokenAdapter(errorReader{err: errors.New(secret)})
	_, err := adapter.Generate(context.Background())
	if !errors.Is(err, application.ErrRefreshTokenRejected) || strings.Contains(err.Error(), secret) {
		t.Fatalf("unsafe random failure: %v", err)
	}
}

type errorReader struct{ err error }

func (reader errorReader) Read([]byte) (int, error) { return 0, reader.err }
