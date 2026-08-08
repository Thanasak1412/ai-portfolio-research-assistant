package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
)

type ready struct{}

func (ready) Ping(context.Context) error { return nil }

type attestor bool

func (value attestor) OriginalRequestWasHTTPS(*fiber.Ctx) bool { return bool(value) }

type fakeOperations struct {
	registerResult application.SessionResult
	loginResult    application.SessionResult
	refreshResult  application.SessionResult
	registerErr    error
	loginErr       error
	refreshErr     error
	logoutErr      error
	resolveErr     error
	currentErr     error
	principal      domain.Principal
	current        application.SafeUser
	refreshInput   application.RefreshInput
	logoutInput    application.RefreshInput
}

func (operations *fakeOperations) Register(context.Context, application.CredentialsInput) (application.SessionResult, error) {
	return operations.registerResult, operations.registerErr
}
func (operations *fakeOperations) Login(context.Context, application.CredentialsInput) (application.SessionResult, error) {
	return operations.loginResult, operations.loginErr
}
func (operations *fakeOperations) Refresh(_ context.Context, input application.RefreshInput) (application.SessionResult, error) {
	operations.refreshInput = input
	return operations.refreshResult, operations.refreshErr
}
func (operations *fakeOperations) Logout(_ context.Context, input application.RefreshInput) error {
	operations.logoutInput = input
	return operations.logoutErr
}
func (operations *fakeOperations) CurrentUser(context.Context, domain.Principal) (application.SafeUser, error) {
	return operations.current, operations.currentErr
}
func (operations *fakeOperations) ResolvePrincipal(context.Context, application.AccessToken) (domain.Principal, error) {
	return operations.principal, operations.resolveErr
}

func TestRegisterUsesSafeJSONAndSecureHostOnlyCookie(t *testing.T) {
	operations := validFakeOperations(t)
	app := testApp(t, operations, true)
	request := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"person@example.com","password":"correct horse battery staple"}`))
	request.Header.Set(nethttp.CanonicalHeaderKey("Content-Type"), "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 201 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if strings.Contains(string(body), "refresh-secret") || !strings.Contains(string(body), `"accessToken":"access-token"`) {
		t.Fatalf("unsafe or incomplete response: %s", body)
	}
	assertRefreshCookie(t, response.Header.Get("Set-Cookie"), true)
}

func TestStrictCredentialsJSONAndGenericErrorEnvelope(t *testing.T) {
	app := testApp(t, validFakeOperations(t), true)
	request := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"person@example.com","password":"correct horse battery staple","unknown":true}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 400 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var envelope errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Code != "INVALID_REQUEST" || envelope.Error.CorrelationID == "" {
		t.Fatalf("error = %+v", envelope.Error)
	}
}

func TestRefreshRequiresAllBrowserSecurityControls(t *testing.T) {
	tests := []struct {
		name                  string
		secure                bool
		origin, requestedWith string
	}{
		{"not attested", false, "https://app.localhost:3443", RequestedWith},
		{"missing origin", true, "", RequestedWith},
		{"wrong origin", true, "https://evil.example", RequestedWith},
		{"missing header", true, "https://app.localhost:3443", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := testApp(t, validFakeOperations(t), test.secure)
			request := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/refresh", nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("X-Requested-With", test.requestedWith)
			request.AddCookie(&nethttp.Cookie{Name: RefreshCookieName, Value: "refresh-secret"})
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != 403 {
				t.Fatalf("status = %d", response.StatusCode)
			}
			if response.Header.Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("CORS header must remain absent")
			}
		})
	}
}

func TestRefreshRotatesCookieAndRejectClearsIt(t *testing.T) {
	operations := validFakeOperations(t)
	app := testApp(t, operations, true)
	response := doBrowserRequest(t, app, "/api/v1/auth/refresh")
	if response.StatusCode != 200 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	assertRefreshCookie(t, response.Header.Get("Set-Cookie"), true)
	if operations.refreshInput.RawToken != "refresh-secret" {
		t.Fatal("cookie was not forwarded only to application")
	}

	operations.refreshErr = application.ErrSessionRefreshRejected
	response = doBrowserRequest(t, app, "/api/v1/auth/refresh")
	if response.StatusCode != 401 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	assertRefreshCookie(t, response.Header.Get("Set-Cookie"), false)
}

func TestLogoutClearsCookieAndHasNoBody(t *testing.T) {
	app := testApp(t, validFakeOperations(t), true)
	response := doBrowserRequest(t, app, "/api/v1/auth/logout")
	if response.StatusCode != 204 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if len(body) != 0 {
		t.Fatalf("body = %q", body)
	}
	assertRefreshCookie(t, response.Header.Get("Set-Cookie"), false)
}

func TestBearerMiddlewareDefaultsToDenyAndMeIsSafe(t *testing.T) {
	operations := validFakeOperations(t)
	app := testApp(t, operations, true)
	for _, header := range []string{"", "Basic abc", "Bearer", "Bearer one two"} {
		request := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/me", nil)
		request.Header.Set("Authorization", header)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 401 {
			t.Fatalf("header %q status = %d", header, response.StatusCode)
		}
	}
	request := httptest.NewRequest(nethttp.MethodGet, "/api/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("status = %d", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	for _, forbidden := range []string{"password", "refresh", "family", "audit"} {
		if strings.Contains(strings.ToLower(string(body)), forbidden) {
			t.Fatalf("response exposed %s: %s", forbidden, body)
		}
	}
}

func TestRateLimitResponseRoundsUpRetryAfter(t *testing.T) {
	operations := validFakeOperations(t)
	operations.loginErr = application.NewRateLimitError(1100 * time.Millisecond)
	app := testApp(t, operations, true)
	request := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"person@example.com","password":"correct horse battery staple"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 429 || response.Header.Get("Retry-After") != "2" {
		t.Fatalf("status=%d retry=%q", response.StatusCode, response.Header.Get("Retry-After"))
	}
}

func TestRequestLoggingDoesNotContainAuthenticationSecrets(t *testing.T) {
	operations := validFakeOperations(t)
	operations.loginErr = application.ErrAuthenticationFailed
	var logs bytes.Buffer
	server := platformhttp.New(slog.New(slog.NewJSONHandler(&logs, nil)), ready{})
	handler, err := NewHandler(operations, "https://app.localhost:3443", attestor(true))
	if err != nil {
		t.Fatal(err)
	}
	handler.Mount(server.App().Group("/api/v1"))
	submittedSecret := strings.Join([]string{"log", "safety", strings.Repeat("x", 16)}, "-")
	request := httptest.NewRequest(nethttp.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"person@example.com","password":"`+submittedSecret+`"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer do-not-log-authorization")
	request.Header.Set("Cookie", "pra_rt_v1=do-not-log-cookie")
	if _, err := server.App().Test(request); err != nil {
		t.Fatal(err)
	}
	output := logs.String()
	for _, forbidden := range []string{submittedSecret, "do-not-log-authorization", "do-not-log-cookie", "person@example.com"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("request logging leaked forbidden value %q: %s", forbidden, output)
		}
	}
}

func validFakeOperations(t *testing.T) *fakeOperations {
	t.Helper()
	access, _ := application.NewAccessToken("access-token")
	refresh, _ := application.NewRefreshToken("refresh-secret")
	userID, _ := domain.NewUserID(uuid.MustParse("11111111-1111-4111-8111-111111111111"))
	principal, _ := domain.NewPrincipal(userID)
	now := time.Date(2026, 8, 8, 10, 0, 0, 0, time.UTC)
	user := application.SafeUser{ID: userID.String(), Email: "person@example.com", Status: "active", CreatedAt: now, UpdatedAt: now}
	result := application.SessionResult{AccessToken: access, User: user, RefreshToken: refresh, CookieExpiresAt: now.Add(30 * 24 * time.Hour), CookieMaxAge: 30 * 24 * time.Hour}
	return &fakeOperations{registerResult: result, loginResult: result, refreshResult: result, principal: principal, current: user}
}

func testApp(t *testing.T, operations *fakeOperations, secure bool) *fiber.App {
	t.Helper()
	server := platformhttp.New(slog.New(slog.NewTextHandler(io.Discard, nil)), ready{})
	handler, err := NewHandler(operations, "https://app.localhost:3443", attestor(secure))
	if err != nil {
		t.Fatal(err)
	}
	handler.Mount(server.App().Group("/api/v1"))
	return server.App()
}

func doBrowserRequest(t *testing.T, app *fiber.App, path string) *nethttp.Response {
	t.Helper()
	request := httptest.NewRequest(nethttp.MethodPost, path, nil)
	request.Header.Set("Origin", "https://app.localhost:3443")
	request.Header.Set("X-Requested-With", RequestedWith)
	request.AddCookie(&nethttp.Cookie{Name: RefreshCookieName, Value: "refresh-secret"})
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func assertRefreshCookie(t *testing.T, raw string, set bool) {
	t.Helper()
	lower := strings.ToLower(raw)
	for _, expected := range []string{strings.ToLower(RefreshCookieName + "="), "path=/api/v1/auth", "httponly", "secure", "samesite=lax"} {
		if !strings.Contains(lower, expected) {
			t.Fatalf("cookie %q missing %q", raw, expected)
		}
	}
	if strings.Contains(lower, "domain=") {
		t.Fatalf("cookie must be host-only: %q", raw)
	}
	if set {
		if strings.Contains(lower, "max-age=0") {
			t.Fatalf("set cookie clears unexpectedly: %q", raw)
		}
	} else if !strings.Contains(lower, "max-age=0") {
		t.Fatalf("clear cookie lacks Max-Age=0: %q", raw)
	}
}

var _ application.Operations = (*fakeOperations)(nil)
var _ = errors.Is
