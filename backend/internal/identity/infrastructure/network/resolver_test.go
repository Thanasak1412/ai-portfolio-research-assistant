package network

import (
	"errors"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

func TestResolverDirectAndUntrustedForwarding(t *testing.T) {
	resolver, _ := NewResolver(nil)
	got, err := resolver.Resolve(application.ClientNetworkRequest{DirectPeerIP: "::ffff:192.0.2.10", ForwardedFor: "203.0.113.9"})
	if err != nil || got.String() != "192.0.2.10" {
		t.Fatalf("direct canonicalization: %v %v", got, err)
	}
	resolver, _ = NewResolver([]string{"10.0.0.0/8"})
	got, err = resolver.Resolve(application.ClientNetworkRequest{DirectPeerIP: "192.0.2.10", ForwardedFor: "203.0.113.9"})
	if err != nil || got.String() != "192.0.2.10" {
		t.Fatal("untrusted peer header was trusted")
	}
}

func TestResolverTrustedChainRightToLeft(t *testing.T) {
	resolver, _ := NewResolver([]string{"10.0.0.0/8"})
	got, err := resolver.Resolve(application.ClientNetworkRequest{DirectPeerIP: "10.0.0.5", ForwardedFor: "198.51.100.1, 203.0.113.7, 10.0.0.4"})
	if err != nil || got.String() != "203.0.113.7" {
		t.Fatalf("trusted chain: %v %v", got, err)
	}
}

func TestResolverRejectsUnsafeForwarding(t *testing.T) {
	resolver, _ := NewResolver([]string{"10.0.0.0/8"})
	for _, xff := range []string{"", "garbage", "fe80::1%eth0", "10.0.0.1,10.0.0.2"} {
		if _, err := resolver.Resolve(application.ClientNetworkRequest{DirectPeerIP: "10.0.0.5", ForwardedFor: xff}); !errors.Is(err, application.ErrClientIdentityRejected) {
			t.Fatalf("unsafe xff accepted %q: %v", xff, err)
		}
	}
	if _, err := NewResolver([]string{"invalid"}); !errors.Is(err, application.ErrInvalidSecurityConfig) {
		t.Fatalf("invalid CIDR accepted: %v", err)
	}
}

func TestParseTrustedProxyCIDRs(t *testing.T) {
	values, err := ParseTrustedProxyCIDRs("10.0.0.0/8, 2001:db8::/32")
	if err != nil || len(values) != 2 {
		t.Fatalf("parse trusted CIDRs: values=%v err=%v", values, err)
	}
	values, err = ParseTrustedProxyCIDRs(" ")
	if err != nil || values != nil {
		t.Fatalf("empty CIDRs must select direct-peer mode: values=%v err=%v", values, err)
	}
	if _, err := ParseTrustedProxyCIDRs("10.0.0.0/8,"); !errors.Is(err, application.ErrInvalidSecurityConfig) {
		t.Fatalf("empty CIDR segment accepted: %v", err)
	}
}
