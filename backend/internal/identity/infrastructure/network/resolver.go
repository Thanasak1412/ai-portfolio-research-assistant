package network

import (
	"net/netip"
	"strings"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

const MaximumForwardedForLength = 4096

type Resolver struct{ trusted []netip.Prefix }

func ParseTrustedProxyCIDRs(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, application.ErrInvalidSecurityConfig
		}
		if _, err := netip.ParsePrefix(trimmed); err != nil {
			return nil, application.ErrInvalidSecurityConfig
		}
		result = append(result, trimmed)
	}
	return result, nil
}

func NewResolver(cidrs []string) (*Resolver, error) {
	trusted := make([]netip.Prefix, 0, len(cidrs))
	for _, value := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil {
			return nil, application.ErrInvalidSecurityConfig
		}
		trusted = append(trusted, prefix.Masked())
	}
	return &Resolver{trusted: trusted}, nil
}

func (resolver *Resolver) Resolve(request application.ClientNetworkRequest) (netip.Addr, error) {
	direct, err := parseAddress(request.DirectPeerIP)
	if err != nil {
		return netip.Addr{}, application.ErrClientIdentityRejected
	}
	if len(resolver.trusted) == 0 || !resolver.isTrusted(direct) {
		return direct, nil
	}
	if request.ForwardedFor == "" || len(request.ForwardedFor) > MaximumForwardedForLength {
		return netip.Addr{}, application.ErrClientIdentityRejected
	}
	parts := strings.Split(request.ForwardedFor, ",")
	addresses := make([]netip.Addr, 0, len(parts))
	for _, part := range parts {
		address, parseErr := parseAddress(strings.TrimSpace(part))
		if parseErr != nil {
			return netip.Addr{}, application.ErrClientIdentityRejected
		}
		addresses = append(addresses, address)
	}
	for index := len(addresses) - 1; index >= 0; index-- {
		if resolver.isTrusted(addresses[index]) {
			continue
		}
		return addresses[index], nil
	}
	return netip.Addr{}, application.ErrClientIdentityRejected
}

func (resolver *Resolver) isTrusted(address netip.Addr) bool {
	for _, prefix := range resolver.trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
func parseAddress(value string) (netip.Addr, error) {
	address, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Addr{}, err
	}
	if address.Zone() != "" {
		return netip.Addr{}, application.ErrClientIdentityRejected
	}
	return address.Unmap(), nil
}

var _ application.ClientNetworkResolver = (*Resolver)(nil)
