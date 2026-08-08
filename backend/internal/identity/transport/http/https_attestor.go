package http

import (
	"bytes"
	"net"
	"net/netip"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

const forwardedProtoHeader = "X-Forwarded-Proto"

// TrustedHTTPSAttestor accepts direct TLS, or one exact scheme assertion from
// a configured TLS-terminating direct peer. It deliberately does not select a
// value from a forwarding chain.
type TrustedHTTPSAttestor struct {
	trusted []netip.Prefix
}

func NewTrustedHTTPSAttestor(cidrs []string) (*TrustedHTTPSAttestor, error) {
	trusted := make([]netip.Prefix, 0, len(cidrs))
	for _, value := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
		if err != nil || prefix.Bits() == 0 {
			return nil, application.ErrInvalidSecurityConfig
		}
		trusted = append(trusted, prefix.Masked())
	}
	return &TrustedHTTPSAttestor{trusted: trusted}, nil
}

func (attestor *TrustedHTTPSAttestor) OriginalRequestWasHTTPS(ctx *fiber.Ctx) bool {
	if state := ctx.Context().TLSConnectionState(); state != nil && state.HandshakeComplete {
		return true
	}
	peer, ok := directPeerAddress(ctx.Context().RemoteAddr())
	if !ok || !attestor.isTrusted(peer) {
		return false
	}
	values := ctx.Context().Request.Header.PeekAll(forwardedProtoHeader)
	return len(values) == 1 && bytes.Equal(values[0], []byte("https"))
}

func (attestor *TrustedHTTPSAttestor) isTrusted(address netip.Addr) bool {
	for _, prefix := range attestor.trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func directPeerAddress(peer net.Addr) (netip.Addr, bool) {
	if peer == nil {
		return netip.Addr{}, false
	}
	if tcp, ok := peer.(*net.TCPAddr); ok {
		address, ok := netip.AddrFromSlice(tcp.IP)
		return normalizedAddress(address, ok)
	}
	host, _, err := net.SplitHostPort(peer.String())
	if err != nil {
		return netip.Addr{}, false
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return normalizedAddress(address, true)
}

func normalizedAddress(address netip.Addr, ok bool) (netip.Addr, bool) {
	if !ok || address.Zone() != "" {
		return netip.Addr{}, false
	}
	return address.Unmap(), true
}

var _ HTTPSAttestor = (*TrustedHTTPSAttestor)(nil)
