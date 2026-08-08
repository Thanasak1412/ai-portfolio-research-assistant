package composition

import (
	"errors"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

func TestTrustedHTTPSProxyCIDRConfiguration(t *testing.T) {
	tests := []struct {
		name, environment, value string
		want                     []string
		invalid                  bool
	}{
		{"empty development", "development", "", nil, false},
		{"one", "development", "10.0.0.1/8", []string{"10.0.0.0/8"}, false},
		{"multiple whitespace", "development", " 10.0.0.0/8 , 2001:db8::/32 ", []string{"10.0.0.0/8", "2001:db8::/32"}, false},
		{"malformed", "development", "not-a-cidr", nil, true},
		{"empty item", "development", "10.0.0.0/8,", nil, true},
		{"staging missing", "staging", "", nil, true},
		{"production missing", "production", "", nil, true},
		{"staging universal ipv4", "staging", "0.0.0.0/0", nil, true},
		{"production universal ipv6", "production", "::/0", nil, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := map[string]string{"AUTH_PUBLIC_ORIGIN": "https://app.localhost:3443", "AUTH_TRUSTED_PROXY_CIDRS": "192.0.2.0/24", "AUTH_TRUSTED_HTTPS_PROXY_CIDRS": test.value}
			configuration, err := loadConfig(test.environment, lookup(values))
			if test.invalid {
				if err == nil || !errors.Is(err, application.ErrInvalidSecurityConfig) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(configuration.TrustedHTTPSProxyCIDRs) != len(test.want) {
				t.Fatalf("CIDRs = %#v", configuration.TrustedHTTPSProxyCIDRs)
			}
			for index := range test.want {
				if configuration.TrustedHTTPSProxyCIDRs[index] != test.want[index] {
					t.Fatalf("CIDRs = %#v", configuration.TrustedHTTPSProxyCIDRs)
				}
			}
		})
	}
}

func TestProxyTrustDecisionsRemainIndependent(t *testing.T) {
	configuration, err := loadConfig("staging", lookup(map[string]string{
		"AUTH_PUBLIC_ORIGIN": "https://app.localhost:3443", "AUTH_TRUSTED_PROXY_CIDRS": "192.0.2.0/24", "AUTH_TRUSTED_HTTPS_PROXY_CIDRS": "198.51.100.0/24",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if configuration.TrustedProxyCIDRs[0] == configuration.TrustedHTTPSProxyCIDRs[0] {
		t.Fatal("independent trust settings were conflated")
	}
}

func lookup(values map[string]string) LookupFunc {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}
