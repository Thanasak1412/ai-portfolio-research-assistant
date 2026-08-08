package token

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

const (
	Audience            = "ai-portfolio-research-assistant-api"
	AccessTokenLifetime = 15 * time.Minute
	ClockSkew           = 60 * time.Second
)

type AccessTokenAdapter struct {
	ring   *KeyRing
	issuer string
}

func NewAccessTokenAdapter(ring *KeyRing, issuer string) (*AccessTokenAdapter, error) {
	if ring == nil || validateIssuer(issuer) != nil {
		return nil, fieldError("AUTH_PUBLIC_ORIGIN")
	}
	return &AccessTokenAdapter{ring: ring, issuer: issuer}, nil
}

func (adapter *AccessTokenAdapter) Issue(ctx context.Context, userID domain.UserID, now time.Time) (application.AccessToken, error) {
	if err := ctx.Err(); err != nil {
		return application.AccessToken{}, err
	}
	if userID.IsZero() || now.IsZero() {
		return application.AccessToken{}, application.ErrAccessTokenRejected
	}
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return application.AccessToken{}, application.ErrAccessTokenRejected
	}
	claims := jwt.RegisteredClaims{
		Issuer: adapter.issuer, Subject: userID.String(), Audience: jwt.ClaimStrings{Audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenLifetime)), IssuedAt: jwt.NewNumericDate(now),
		ID: base64.RawURLEncoding.EncodeToString(jtiBytes),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = adapter.ring.ActiveKeyID()
	signed, err := token.SignedString(adapter.ring.signingKey())
	if err != nil {
		return application.AccessToken{}, application.ErrAccessTokenRejected
	}
	return application.NewAccessToken(signed)
}

func (adapter *AccessTokenAdapter) Verify(ctx context.Context, accessToken application.AccessToken, now time.Time) (domain.Principal, error) {
	if err := ctx.Err(); err != nil {
		return domain.Principal{}, err
	}
	if now.IsZero() {
		return domain.Principal{}, application.ErrAccessTokenRejected
	}
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(accessToken.Value(), claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodEdDSA || token.Method.Alg() != "EdDSA" {
			return nil, application.ErrAccessTokenRejected
		}
		keyID, ok := token.Header["kid"].(string)
		if !ok || keyID == "" {
			return nil, application.ErrAccessTokenRejected
		}
		key, exists := adapter.ring.verificationKey(keyID)
		if !exists {
			return nil, application.ErrAccessTokenRejected
		}
		return key, nil
	}, jwt.WithValidMethods([]string{"EdDSA"}), jwt.WithIssuer(adapter.issuer), jwt.WithAudience(Audience),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(ClockSkew), jwt.WithTimeFunc(func() time.Time { return now }), jwt.WithStrictDecoding())
	if err != nil || !parsed.Valid || claims.Issuer == "" || len(claims.Audience) == 0 || claims.Subject == "" || claims.IssuedAt == nil || claims.ExpiresAt == nil || claims.ID == "" {
		return domain.Principal{}, application.ErrAccessTokenRejected
	}
	if claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) != AccessTokenLifetime {
		return domain.Principal{}, application.ErrAccessTokenRejected
	}
	jti, err := base64.RawURLEncoding.Strict().DecodeString(claims.ID)
	if err != nil || len(jti) != 16 {
		return domain.Principal{}, application.ErrAccessTokenRejected
	}
	userID, err := domain.ParseUserID(claims.Subject)
	if err != nil {
		return domain.Principal{}, application.ErrAccessTokenRejected
	}
	principal, err := domain.NewPrincipal(userID)
	if err != nil {
		return domain.Principal{}, application.ErrAccessTokenRejected
	}
	return principal, nil
}

func validateIssuer(value string) error {
	if strings.HasSuffix(value, "/") {
		return application.ErrInvalidSecurityConfig
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return application.ErrInvalidSecurityConfig
	}
	return nil
}

var _ application.AccessTokenService = (*AccessTokenAdapter)(nil)
