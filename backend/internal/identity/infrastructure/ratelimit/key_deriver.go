package ratelimit

import (
	"crypto/hmac"
	"crypto/sha256"
	"net/netip"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/authhmac"
)

const rateLimitHMACDomain = "auth_rate_limit\x00v1\x00"

type KeyDeriver struct{ key authhmac.RateLimitKey }

func NewKeyDeriver(key authhmac.RateLimitKey) (*KeyDeriver, error) {
	if key.IsZero() {
		return nil, application.ErrInvalidSecurityConfig
	}
	return &KeyDeriver{key: key}, nil
}

func (deriver *KeyDeriver) LoginEmailFailure(email domain.NormalizedEmail) (application.RateLimitKey, error) {
	if email.IsZero() {
		return application.RateLimitKey{}, application.ErrInvalidRateLimitKey
	}
	return deriver.derive(application.RateLimitLoginEmailFailure, email.String())
}

func (deriver *KeyDeriver) LoginIPAttempt(address netip.Addr) (application.RateLimitKey, error) {
	canonical, err := canonicalAddress(address)
	if err != nil {
		return application.RateLimitKey{}, err
	}
	return deriver.derive(application.RateLimitLoginIPAttempt, canonical)
}

func (deriver *KeyDeriver) RegistrationIPAttempt(address netip.Addr) (application.RateLimitKey, error) {
	canonical, err := canonicalAddress(address)
	if err != nil {
		return application.RateLimitKey{}, err
	}
	return deriver.derive(application.RateLimitRegistrationIPAttempt, canonical)
}

func (deriver *KeyDeriver) RefreshFamilyAttempt(familyID domain.TokenFamilyID) (application.RateLimitKey, error) {
	if familyID.IsZero() {
		return application.RateLimitKey{}, application.ErrInvalidRateLimitKey
	}
	return deriver.derive(application.RateLimitRefreshFamilyAttempt, familyID.String())
}

func (deriver *KeyDeriver) derive(policy application.RateLimitPolicy, canonicalIdentity string) (application.RateLimitKey, error) {
	if deriver == nil || deriver.key.IsZero() || canonicalIdentity == "" {
		return application.RateLimitKey{}, application.ErrInvalidRateLimitKey
	}
	mac := hmac.New(sha256.New, deriver.key.Bytes())
	_, _ = mac.Write([]byte(rateLimitHMACDomain))
	_, _ = mac.Write([]byte(policy))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(application.RateLimitPolicyVersion))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(canonicalIdentity))
	return application.NewRateLimitKey(mac.Sum(nil))
}

func canonicalAddress(address netip.Addr) (string, error) {
	if !address.IsValid() || address.Zone() != "" {
		return "", application.ErrInvalidRateLimitKey
	}
	return address.Unmap().String(), nil
}

var _ application.RateLimitKeyDeriver = (*KeyDeriver)(nil)
