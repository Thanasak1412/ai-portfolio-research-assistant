//go:build integration

package http

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

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	identityaudit "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/audit"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/authhmac"
	identitydatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/database"
	identitynetwork "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/network"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/password"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/ratelimit"
	identityruntime "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/runtime"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/infrastructure/token"
	platformdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
)

func TestPostgresBackedAuthenticationFlow(t *testing.T) {
	pool := integrationPool(t)
	service := integrationService(t, pool)
	server := platformhttp.New(slog.New(slog.NewJSONHandler(io.Discard, nil)), ready{})
	handler, err := NewHandler(service, "https://app.localhost:3443", attestor(true))
	if err != nil {
		t.Fatal(err)
	}
	handler.Mount(server.App().Group("/api/v1"))

	email := "flow-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.com"
	register := integrationCredentialsRequest(t, server.App(), "/api/v1/auth/register", email, "correct horse battery staple")
	if register.StatusCode != 201 {
		t.Fatalf("register status=%d body=%s", register.StatusCode, readBody(register))
	}
	registrationCookie := cookieValue(t, register, RefreshCookieName)
	registeredAccess := decodeAccess(t, register)
	assertMe(t, server.App(), registeredAccess, email)

	login := integrationCredentialsRequest(t, server.App(), "/api/v1/auth/login", email, "correct horse battery staple")
	if login.StatusCode != 200 {
		t.Fatalf("login status=%d body=%s", login.StatusCode, readBody(login))
	}
	loginCookie := cookieValue(t, login, RefreshCookieName)

	refresh := integrationBrowserRequest(t, server.App(), "/api/v1/auth/refresh", loginCookie)
	if refresh.StatusCode != 200 {
		t.Fatalf("refresh status=%d body=%s", refresh.StatusCode, readBody(refresh))
	}
	rotatedCookie := cookieValue(t, refresh, RefreshCookieName)
	refreshedAccess := decodeAccess(t, refresh)
	assertMe(t, server.App(), refreshedAccess, email)

	logout := integrationBrowserRequest(t, server.App(), "/api/v1/auth/logout", rotatedCookie)
	if logout.StatusCode != 204 {
		t.Fatalf("logout status=%d body=%s", logout.StatusCode, readBody(logout))
	}
	rejected := integrationBrowserRequest(t, server.App(), "/api/v1/auth/refresh", rotatedCookie)
	if rejected.StatusCode != 401 {
		t.Fatalf("post-logout refresh status=%d body=%s", rejected.StatusCode, readBody(rejected))
	}

	// The independent registration family is not revoked by logout of the login family.
	stillValid := integrationBrowserRequest(t, server.App(), "/api/v1/auth/refresh", registrationCookie)
	if stillValid.StatusCode != 200 {
		t.Fatalf("unrelated family status=%d body=%s", stillValid.StatusCode, readBody(stillValid))
	}
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func integrationService(t *testing.T, pool *pgxpool.Pool) *application.Service {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, _ := x509.MarshalPKCS8PrivateKey(privateKey)
	publicDER, _ := x509.MarshalPKIXPublicKey(publicKey)
	keyID := "auth-ed25519-20260808-01"
	ring, err := token.ParseKeyRing(token.KeyRingInput{ActiveKeyID: keyID, ActivePrivateKeyB64: base64.StdEncoding.EncodeToString(privateDER), VerificationKeys: []token.VerificationKeyInput{{KeyID: keyID, PublicKeyB64: base64.StdEncoding.EncodeToString(publicDER)}}})
	if err != nil {
		t.Fatal(err)
	}
	accessTokens, err := token.NewAccessTokenAdapter(ring, "https://app.localhost:3443")
	if err != nil {
		t.Fatal(err)
	}
	networkKey, rateKey, err := authhmac.ParsePair(base64.StdEncoding.EncodeToString(integrationKey(1)), base64.StdEncoding.EncodeToString(integrationKey(2)))
	if err != nil {
		t.Fatal(err)
	}
	resolver, _ := identitynetwork.NewResolver(nil)
	networkHasher, _ := identitynetwork.NewIdentityHasher(networkKey)
	rateKeys, _ := ratelimit.NewKeyDeriver(rateKey)
	passwords := password.New()
	dummy, err := passwords.Hash(context.Background(), "integration dummy password")
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewService(application.ServiceDependencies{
		Users: identitydatabase.NewPostgresUserRepository(pool), Transactor: identitydatabase.NewPostgresTransactor(pool),
		Passwords: passwords, AccessTokens: accessTokens, RefreshTokens: token.NewRefreshTokenAdapter(), NetworkResolver: resolver, NetworkHasher: networkHasher,
		Audit: identityaudit.NewPostgresWriter(platformdatabase.NewAuthenticationAuditStore(pool)), RateLimiter: ratelimit.NewPostgresLimiter(platformdatabase.NewAuthenticationRateLimitStore(pool)),
		RateKeys: rateKeys, Clock: identityruntime.Clock{}, IDs: identityruntime.IDGenerator{}, DummyHash: dummy,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func integrationCredentialsRequest(t *testing.T, app *fiber.App, path, email, password string) *nethttp.Response {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"email": email, "password": password})
	request := httptest.NewRequest(nethttp.MethodPost, path, strings.NewReader(string(payload)))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request, 15000)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func integrationBrowserRequest(t *testing.T, app *fiber.App, path, cookie string) *nethttp.Response {
	t.Helper()
	request := httptest.NewRequest(nethttp.MethodPost, path, nil)
	request.Header.Set("Origin", "https://app.localhost:3443")
	request.Header.Set("X-Requested-With", RequestedWith)
	request.AddCookie(&nethttp.Cookie{Name: RefreshCookieName, Value: cookie})
	response, err := app.Test(request, 15000)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func cookieValue(t *testing.T, response *nethttp.Response, name string) string {
	t.Helper()
	for _, cookie := range response.Cookies() {
		if cookie.Name == name && cookie.Value != "" {
			return cookie.Value
		}
	}
	t.Fatal("refresh cookie missing")
	return ""
}

func decodeAccess(t *testing.T, response *nethttp.Response) string {
	t.Helper()
	var body struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.AccessToken == "" {
		t.Fatal("access token missing")
	}
	return body.AccessToken
}

func assertMe(t *testing.T, app *fiber.App, accessToken, email string) {
	t.Helper()
	request := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := app.Test(request, 15000)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("me status=%d body=%s", response.StatusCode, readBody(response))
	}
	var user authenticatedUserResponse
	if err := json.NewDecoder(response.Body).Decode(&user); err != nil {
		t.Fatal(err)
	}
	if user.Email != email || user.Status != "active" {
		t.Fatalf("me user=%+v", user)
	}
}

func readBody(response *nethttp.Response) string {
	body, _ := io.ReadAll(response.Body)
	return string(body)
}
func integrationKey(value byte) []byte {
	key := make([]byte, 32)
	for index := range key {
		key[index] = value
	}
	return key
}
