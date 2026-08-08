//go:build integration

package composition

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
)

type runtimeReady struct{}

func (runtimeReady) Ping(context.Context) error { return nil }

func TestConfiguredRuntimeMountsAuthenticationRoutes(t *testing.T) {
	pool := runtimePool(t)
	handler, err := BuildHTTP(context.Background(), pool, "test", testLookup(t))
	if err != nil {
		t.Fatal(err)
	}
	server := platformhttp.New(slog.New(slog.NewTextHandler(io.Discard, nil)), runtimeReady{}, handler)
	email := "runtime-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.com"
	register := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"`+email+`","password":"correct horse battery staple"}`))
	register.Header.Set("Content-Type", "application/json")
	response, err := server.App().Test(register)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != nethttp.StatusCreated {
		t.Fatalf("register status = %d", response.StatusCode)
	}
	var payload struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.AccessToken == "" {
		t.Fatal("registration did not issue an access token")
	}
	me := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/me", nil)
	me.Header.Set("Authorization", "Bearer "+payload.AccessToken)
	response, err = server.App().Test(me)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != nethttp.StatusOK {
		t.Fatalf("me status = %d", response.StatusCode)
	}
	for _, path := range []string{"/api/v1/auth/login", "/api/v1/auth/refresh", "/api/v1/auth/logout"} {
		request := httptest.NewRequest(nethttp.MethodPost, path, nil)
		response, err := server.App().Test(request)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode == nethttp.StatusNotFound {
			t.Fatalf("%s was not mounted", path)
		}
	}
}

func runtimePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func testLookup(t *testing.T) LookupFunc {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	publicB64 := base64.StdEncoding.EncodeToString(publicDER)
	keys, err := json.Marshal([]map[string]string{{"kid": "auth-ed25519-20260808-01", "publicKeyB64": publicB64}})
	if err != nil {
		t.Fatal(err)
	}
	network := make([]byte, 32)
	rate := make([]byte, 32)
	if _, err := rand.Read(network); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(rate); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{
		"AUTH_PUBLIC_ORIGIN": "https://app.localhost:3443", "AUTH_JWT_ACTIVE_KID": "auth-ed25519-20260808-01",
		"AUTH_JWT_ACTIVE_PRIVATE_KEY_B64": base64.StdEncoding.EncodeToString(privateDER), "AUTH_JWT_VERIFICATION_KEYS_JSON": string(keys),
		"AUTH_NETWORK_HMAC_KEY": base64.StdEncoding.EncodeToString(network), "AUTH_RATE_LIMIT_HMAC_KEY": base64.StdEncoding.EncodeToString(rate),
	}
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}
