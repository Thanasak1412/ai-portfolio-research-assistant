package http

import (
	"net"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestBrowserSecurityUsesTrustedHTTPSAttestation(t *testing.T) {
	attestor, err := NewTrustedHTTPSAttestor([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, peer        string
		protos            []string
		origin, requested string
		want              int
	}{
		{"trusted exact", "10.1.2.3", []string{"https"}, "https://app.localhost:3443", RequestedWith, 200},
		{"untrusted spoof", "198.51.100.1", []string{"https"}, "https://app.localhost:3443", RequestedWith, 403},
		{"trusted missing scheme", "10.1.2.3", nil, "https://app.localhost:3443", RequestedWith, 403},
		{"trusted http", "10.1.2.3", []string{"http"}, "https://app.localhost:3443", RequestedWith, 403},
		{"trusted duplicate", "10.1.2.3", []string{"https", "https"}, "https://app.localhost:3443", RequestedWith, 403},
		{"wrong origin", "10.1.2.3", []string{"https"}, "https://evil.example", RequestedWith, 403},
		{"wrong requested header", "10.1.2.3", []string{"https"}, "https://app.localhost:3443", "wrong", 403},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operations := validFakeOperations(t)
			handler, err := NewHandler(operations, "https://app.localhost:3443", attestor)
			if err != nil {
				t.Fatal(err)
			}
			app := fiber.New()
			app.Use(func(ctx *fiber.Ctx) error {
				ctx.Context().SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP(test.peer), Port: 443})
				for _, proto := range test.protos {
					ctx.Context().Request.Header.Add(forwardedProtoHeader, proto)
				}
				return ctx.Next()
			})
			handler.Mount(app.Group("/api/v1"))
			request := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/refresh", nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("X-Requested-With", test.requested)
			request.AddCookie(&nethttp.Cookie{Name: RefreshCookieName, Value: "refresh-secret"})
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.want {
				t.Fatalf("status = %d", response.StatusCode)
			}
		})
	}
}
