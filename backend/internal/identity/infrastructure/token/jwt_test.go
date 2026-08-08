package token

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

func TestAccessTokenIssueAndVerify(t *testing.T) {
	ring := testKeyRing(t, "auth-ed25519-20260808-01")
	adapter, err := NewAccessTokenAdapter(ring, "https://portfolio.example.test")
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := domain.NewUserID(uuid.New())
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	first, err := adapter.Issue(context.Background(), userID, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Issue(context.Background(), userID, now)
	if err != nil {
		t.Fatal(err)
	}
	if first.Value() == second.Value() || strings.Contains(first.String(), first.Value()) {
		t.Fatal("token randomness or redaction failed")
	}
	principal, err := adapter.Verify(context.Background(), first, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := principal.UserID()
	if !ok || got != userID {
		t.Fatal("verified principal mismatch")
	}
	if _, err := adapter.Verify(context.Background(), first, now.Add(AccessTokenLifetime+ClockSkew+time.Second)); !errors.Is(err, application.ErrAccessTokenRejected) {
		t.Fatalf("expired token accepted: %v", err)
	}
}

func TestAccessTokenRejectsAlgorithmIssuerAudienceAndUnknownKey(t *testing.T) {
	ring := testKeyRing(t, "auth-ed25519-20260808-01")
	now := time.Now().UTC()
	userID, _ := domain.NewUserID(uuid.New())
	adapter, _ := NewAccessTokenAdapter(ring, "https://portfolio.example.test")
	valid, _ := adapter.Issue(context.Background(), userID, now)
	wrongIssuer, _ := NewAccessTokenAdapter(ring, "https://other.example.test")
	if _, err := wrongIssuer.Verify(context.Background(), valid, now); !errors.Is(err, application.ErrAccessTokenRejected) {
		t.Fatal("wrong issuer accepted")
	}

	claims := jwt.RegisteredClaims{Issuer: "https://portfolio.example.test", Subject: userID.String(), Audience: jwt.ClaimStrings{"wrong"}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)), ID: "id"}
	badAudience := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	badAudience.Header["kid"] = ring.ActiveKeyID()
	raw, _ := badAudience.SignedString(ring.signingKey())
	wrapped, _ := application.NewAccessToken(raw)
	if _, err := adapter.Verify(context.Background(), wrapped, now); !errors.Is(err, application.ErrAccessTokenRejected) {
		t.Fatal("wrong audience accepted")
	}

	unknownClaims := claims
	unknownClaims.Audience = jwt.ClaimStrings{Audience}
	unknown := jwt.NewWithClaims(jwt.SigningMethodEdDSA, unknownClaims)
	unknown.Header["kid"] = "auth-ed25519-20260808-99"
	raw, _ = unknown.SignedString(ring.signingKey())
	wrapped, _ = application.NewAccessToken(raw)
	if _, err := adapter.Verify(context.Background(), wrapped, now); !errors.Is(err, application.ErrAccessTokenRejected) {
		t.Fatal("unknown key accepted")
	}

	none := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	none.Header["kid"] = ring.ActiveKeyID()
	raw, _ = none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	wrapped, _ = application.NewAccessToken(raw)
	if _, err := adapter.Verify(context.Background(), wrapped, now); !errors.Is(err, application.ErrAccessTokenRejected) {
		t.Fatal("alg none accepted")
	}
}

func TestAccessTokenRejectsMissingFutureAndInvalidClaims(t *testing.T) {
	ring := testKeyRing(t, "auth-ed25519-20260808-01")
	adapter, _ := NewAccessTokenAdapter(ring, "https://portfolio.example.test")
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	userID, _ := domain.NewUserID(uuid.New())
	valid := jwt.RegisteredClaims{
		Issuer: "https://portfolio.example.test", Subject: userID.String(), Audience: jwt.ClaimStrings{Audience},
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenLifetime)), ID: base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
	}
	tests := map[string]jwt.RegisteredClaims{
		"missing subject":   func() jwt.RegisteredClaims { value := valid; value.Subject = ""; return value }(),
		"missing issued at": func() jwt.RegisteredClaims { value := valid; value.IssuedAt = nil; return value }(),
		"missing expiry":    func() jwt.RegisteredClaims { value := valid; value.ExpiresAt = nil; return value }(),
		"missing jti":       func() jwt.RegisteredClaims { value := valid; value.ID = ""; return value }(),
		"future issued at": func() jwt.RegisteredClaims {
			value := valid
			value.IssuedAt = jwt.NewNumericDate(now.Add(ClockSkew + time.Second))
			value.ExpiresAt = jwt.NewNumericDate(value.IssuedAt.Time.Add(AccessTokenLifetime))
			return value
		}(),
		"future not before": func() jwt.RegisteredClaims {
			value := valid
			value.NotBefore = jwt.NewNumericDate(now.Add(ClockSkew + time.Second))
			return value
		}(),
		"wrong lifetime": func() jwt.RegisteredClaims {
			value := valid
			value.ExpiresAt = jwt.NewNumericDate(now.Add(AccessTokenLifetime + time.Second))
			return value
		}(),
	}
	for name, claims := range tests {
		t.Run(name, func(t *testing.T) {
			raw := signClaims(t, ring, claims, ring.ActiveKeyID())
			wrapped, _ := application.NewAccessToken(raw)
			if _, err := adapter.Verify(context.Background(), wrapped, now); !errors.Is(err, application.ErrAccessTokenRejected) {
				t.Fatalf("invalid claims accepted: %v", err)
			}
		})
	}

	otherRing := testKeyRing(t, "auth-ed25519-20260808-02")
	raw := signClaims(t, otherRing, valid, ring.ActiveKeyID())
	wrapped, _ := application.NewAccessToken(raw)
	if _, err := adapter.Verify(context.Background(), wrapped, now); !errors.Is(err, application.ErrAccessTokenRejected) {
		t.Fatal("invalid signature accepted")
	}
}

func TestKeyRingRejectsMalformedDuplicateAndUnsafeInputs(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	valid := keyRingInput(t, "auth-ed25519-20260808-01", publicKey, privateKey, []VerificationKeyInput{
		publicInput(t, "auth-ed25519-20260808-01", publicKey),
	})
	for name, mutate := range map[string]func(*KeyRingInput){
		"invalid base64":        func(input *KeyRingInput) { input.ActivePrivateKeyB64 = "%%%" },
		"active public missing": func(input *KeyRingInput) { input.VerificationKeys = nil },
		"duplicate kid": func(input *KeyRingInput) {
			input.VerificationKeys = append(input.VerificationKeys, input.VerificationKeys[0])
		},
		"invalid kid":      func(input *KeyRingInput) { input.ActiveKeyID = "unsafe" },
		"invalid kid date": func(input *KeyRingInput) { input.ActiveKeyID = "auth-ed25519-20261340-01" },
	} {
		t.Run(name, func(t *testing.T) {
			input := valid
			input.VerificationKeys = append([]VerificationKeyInput(nil), valid.VerificationKeys...)
			mutate(&input)
			_, err := ParseKeyRing(input)
			if !errors.Is(err, application.ErrInvalidSecurityConfig) || strings.Contains(err.Error(), input.ActivePrivateKeyB64) {
				t.Fatalf("unsafe key-ring result: %v", err)
			}
		})
	}
	if _, err := ParseVerificationKeysJSON(`[{"kid":"auth-ed25519-20260808-01","publicKeyB64":"bad"}] trailing`); !errors.Is(err, application.ErrInvalidSecurityConfig) {
		t.Fatalf("trailing JSON accepted: %v", err)
	}
}

func TestKeyRingEnvironmentAndOwnerOnlyLocalFileLoading(t *testing.T) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	input := keyRingInput(t, "auth-ed25519-20260808-01", publicKey, privateKey, []VerificationKeyInput{
		publicInput(t, "auth-ed25519-20260808-01", publicKey),
	})
	verificationJSON, _ := json.Marshal(input.VerificationKeys)
	if _, err := LoadEnvironmentKeyRing(input.ActiveKeyID, input.ActivePrivateKeyB64, string(verificationJSON)); err != nil {
		t.Fatalf("load environment key ring: %v", err)
	}

	content, _ := json.Marshal(input)
	path := filepath.Join(t.TempDir(), "key-ring.json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLocalKeyRing(path); err != nil {
		t.Fatalf("load owner-only local key ring: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLocalKeyRing(path); !errors.Is(err, application.ErrInvalidSecurityConfig) {
		t.Fatalf("insecure local permissions accepted: %v", err)
	}
}

func TestAccessTokenIssuerConfiguration(t *testing.T) {
	ring := testKeyRing(t, "auth-ed25519-20260808-01")
	for _, issuer := range []string{"", "http://portfolio.example.test", "https://portfolio.example.test/", "https://portfolio.example.test/path", "https://portfolio.example.test?query=value"} {
		if _, err := NewAccessTokenAdapter(ring, issuer); !errors.Is(err, application.ErrInvalidSecurityConfig) || strings.Contains(err.Error(), issuer+"private") {
			t.Fatalf("invalid issuer accepted or leaked %q: %v", issuer, err)
		}
	}
}

func TestKeyRingValidationAndOverlap(t *testing.T) {
	activePublic, activePrivate, _ := ed25519.GenerateKey(rand.Reader)
	oldPublic, oldPrivate, _ := ed25519.GenerateKey(rand.Reader)
	input := keyRingInput(t, "auth-ed25519-20260808-01", activePublic, activePrivate, []VerificationKeyInput{publicInput(t, "auth-ed25519-20260808-01", activePublic), publicInput(t, "auth-ed25519-20260708-01", oldPublic)})
	ring, err := ParseKeyRing(input)
	if err != nil {
		t.Fatal(err)
	}
	oldRingInput := keyRingInput(t, "auth-ed25519-20260708-01", oldPublic, oldPrivate, []VerificationKeyInput{publicInput(t, "auth-ed25519-20260708-01", oldPublic)})
	oldRing, _ := ParseKeyRing(oldRingInput)
	issuer := "https://portfolio.example.test"
	now := time.Now().UTC()
	userID, _ := domain.NewUserID(uuid.New())
	oldAdapter, _ := NewAccessTokenAdapter(oldRing, issuer)
	oldToken, _ := oldAdapter.Issue(context.Background(), userID, now)
	currentAdapter, _ := NewAccessTokenAdapter(ring, issuer)
	if _, err := currentAdapter.Verify(context.Background(), oldToken, now); err != nil {
		t.Fatalf("overlap key did not verify: %v", err)
	}
	input.VerificationKeys = input.VerificationKeys[:1]
	removed, _ := ParseKeyRing(input)
	removedAdapter, _ := NewAccessTokenAdapter(removed, issuer)
	if _, err := removedAdapter.Verify(context.Background(), oldToken, now); !errors.Is(err, application.ErrAccessTokenRejected) {
		t.Fatal("removed key accepted")
	}

	mismatch := keyRingInput(t, "auth-ed25519-20260808-01", activePublic, oldPrivate, []VerificationKeyInput{publicInput(t, "auth-ed25519-20260808-01", activePublic)})
	if _, err := ParseKeyRing(mismatch); !errors.Is(err, application.ErrInvalidSecurityConfig) || strings.Contains(err.Error(), mismatch.ActivePrivateKeyB64) {
		t.Fatalf("unsafe mismatch error: %v", err)
	}
}

func testKeyRing(t *testing.T, kid string) *KeyRing {
	t.Helper()
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	ring, err := ParseKeyRing(keyRingInput(t, kid, public, private, []VerificationKeyInput{publicInput(t, kid, public)}))
	if err != nil {
		t.Fatal(err)
	}
	return ring
}
func keyRingInput(t *testing.T, kid string, public ed25519.PublicKey, private ed25519.PrivateKey, keys []VerificationKeyInput) KeyRingInput {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	return KeyRingInput{ActiveKeyID: kid, ActivePrivateKeyB64: base64.StdEncoding.EncodeToString(der), VerificationKeys: keys}
}
func publicInput(t *testing.T, kid string, public ed25519.PublicKey) VerificationKeyInput {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	return VerificationKeyInput{KeyID: kid, PublicKeyB64: base64.StdEncoding.EncodeToString(der)}
}

func signClaims(t *testing.T, ring *KeyRing, claims jwt.RegisteredClaims, keyID string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = keyID
	raw, err := token.SignedString(ring.signingKey())
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
