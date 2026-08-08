package token

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

const (
	refreshTokenPrefix        = "rt_v1_"
	refreshTokenEntropyLength = 32
	refreshTokenPayloadLength = 43
	refreshTokenLength        = len(refreshTokenPrefix) + refreshTokenPayloadLength
)

type RefreshTokenAdapter struct{ random io.Reader }

func NewRefreshTokenAdapter() *RefreshTokenAdapter {
	return &RefreshTokenAdapter{random: rand.Reader}
}

func newRefreshTokenAdapter(random io.Reader) *RefreshTokenAdapter {
	return &RefreshTokenAdapter{random: random}
}

func (adapter *RefreshTokenAdapter) Generate(ctx context.Context) (application.RefreshToken, error) {
	if adapter == nil || adapter.random == nil {
		return application.RefreshToken{}, application.ErrRefreshTokenRejected
	}
	if err := ctx.Err(); err != nil {
		return application.RefreshToken{}, err
	}
	randomBytes := make([]byte, refreshTokenEntropyLength)
	if _, err := io.ReadFull(adapter.random, randomBytes); err != nil {
		return application.RefreshToken{}, application.ErrRefreshTokenRejected
	}
	external := refreshTokenPrefix + base64.RawURLEncoding.EncodeToString(randomBytes)
	return application.NewRefreshToken(external)
}

func (adapter *RefreshTokenAdapter) Parse(external string) (application.RefreshToken, error) {
	if !isCanonicalRefreshToken(external) {
		return application.RefreshToken{}, application.ErrRefreshTokenRejected
	}
	return application.NewRefreshToken(external)
}

func (adapter *RefreshTokenAdapter) Digest(token application.RefreshToken) (domain.TokenDigest, error) {
	if token.IsZero() || !isCanonicalRefreshToken(token.Value()) {
		return domain.TokenDigest{}, application.ErrRefreshTokenRejected
	}
	digest := sha256.Sum256([]byte(token.Value()))
	return domain.NewTokenDigest(digest[:])
}

func isCanonicalRefreshToken(external string) bool {
	if len(external) != refreshTokenLength || !strings.HasPrefix(external, refreshTokenPrefix) {
		return false
	}
	payload := external[len(refreshTokenPrefix):]
	if len(payload) != refreshTokenPayloadLength || strings.Contains(payload, "=") {
		return false
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(payload)
	return err == nil && len(decoded) == refreshTokenEntropyLength && base64.RawURLEncoding.EncodeToString(decoded) == payload
}

var _ application.RefreshTokenService = (*RefreshTokenAdapter)(nil)
