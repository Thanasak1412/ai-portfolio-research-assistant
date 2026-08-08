package network

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/netip"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/authhmac"
)

const (
	networkIdentityPrefix = "ip_hmac_v1:"
	networkHMACDomain     = "ip_hmac\x00v1\x00"
)

type IdentityHasher struct{ key authhmac.NetworkKey }

func NewIdentityHasher(key authhmac.NetworkKey) (*IdentityHasher, error) {
	if key.IsZero() {
		return nil, application.ErrInvalidSecurityConfig
	}
	return &IdentityHasher{key: key}, nil
}

func (hasher *IdentityHasher) Hash(address netip.Addr) (string, error) {
	if hasher == nil || hasher.key.IsZero() || !address.IsValid() || address.Zone() != "" {
		return "", application.ErrClientIdentityRejected
	}
	canonical := address.Unmap().String()
	mac := hmac.New(sha256.New, hasher.key.Bytes())
	_, _ = mac.Write([]byte(networkHMACDomain))
	_, _ = mac.Write([]byte(canonical))
	return networkIdentityPrefix + hex.EncodeToString(mac.Sum(nil)), nil
}

var _ application.NetworkIdentityHasher = (*IdentityHasher)(nil)
