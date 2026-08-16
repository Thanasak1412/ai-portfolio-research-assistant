package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	identityapplication "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	identityhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/transport/http"
	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

type operationFake struct {
	create  func(context.Context, identitydomain.Principal, application.CreatePortfolioInput) (domain.Portfolio, error)
	list    func(context.Context, identitydomain.Principal, domain.PortfolioStatus) ([]domain.Portfolio, error)
	get     func(context.Context, identitydomain.Principal, domain.PortfolioID) (domain.Portfolio, error)
	update  func(context.Context, identitydomain.Principal, domain.PortfolioID, application.UpdatePortfolioInput) (domain.Portfolio, error)
	archive func(context.Context, identitydomain.Principal, domain.PortfolioID) (domain.Portfolio, error)
}

func (fake operationFake) CreatePortfolio(ctx context.Context, principal identitydomain.Principal, input application.CreatePortfolioInput) (domain.Portfolio, error) {
	return fake.create(ctx, principal, input)
}
func (fake operationFake) ListPortfolios(ctx context.Context, principal identitydomain.Principal, status domain.PortfolioStatus) ([]domain.Portfolio, error) {
	return fake.list(ctx, principal, status)
}
func (fake operationFake) GetPortfolio(ctx context.Context, principal identitydomain.Principal, id domain.PortfolioID) (domain.Portfolio, error) {
	return fake.get(ctx, principal, id)
}
func (fake operationFake) UpdatePortfolio(ctx context.Context, principal identitydomain.Principal, id domain.PortfolioID, input application.UpdatePortfolioInput) (domain.Portfolio, error) {
	return fake.update(ctx, principal, id, input)
}
func (fake operationFake) ArchivePortfolio(ctx context.Context, principal identitydomain.Principal, id domain.PortfolioID) (domain.Portfolio, error) {
	return fake.archive(ctx, principal, id)
}

func TestHandlerCreateUsesTrustedPrincipalAndStrictJSON(t *testing.T) {
	principal := testPrincipal(t)
	portfolio := testPortfolio(t, principal)
	called := false
	app := testApp(t, operationFake{create: func(_ context.Context, got identitydomain.Principal, input application.CreatePortfolioInput) (domain.Portfolio, error) {
		called = true
		if got != principal || input.Name != "Primary" || input.BaseCurrency != "USD" {
			t.Fatalf("unexpected operation input: %#v", input)
		}
		return portfolio, nil
	}})

	response := perform(t, app, "POST", "/api/v1/portfolios", `{"name":"Primary","baseCurrency":"USD"}`, "application/json")
	if response.Code != fiber.StatusCreated || !called {
		t.Fatalf("status=%d called=%t", response.Code, called)
	}
	var body portfolioResponse
	decode(t, response, &body)
	if body.ID != portfolio.ID().String() || body.ArchivedAt != nil {
		t.Fatalf("unexpected portfolio response: %#v", body)
	}

	for _, payload := range []string{`{"name":"Primary","baseCurrency":"USD","ownerUserId":"ignored"}`, `{"name":"Primary","baseCurrency":"USD"} {}`, `{}`} {
		response := perform(t, app, "POST", "/api/v1/portfolios", payload, "application/json")
		assertError(t, response, fiber.StatusBadRequest, "INVALID_REQUEST")
	}
	response = perform(t, app, "POST", "/api/v1/portfolios", `{"name":"Primary","baseCurrency":"EUR"}`, "application/json")
	assertError(t, response, fiber.StatusBadRequest, "INVALID_REQUEST")
}

func TestHandlerListDefaultsAndValidatesStatus(t *testing.T) {
	principal := testPrincipal(t)
	portfolio := testPortfolio(t, principal)
	var statuses []domain.PortfolioStatus
	app := testApp(t, operationFake{list: func(_ context.Context, _ identitydomain.Principal, status domain.PortfolioStatus) ([]domain.Portfolio, error) {
		statuses = append(statuses, status)
		return []domain.Portfolio{portfolio}, nil
	}})
	for _, path := range []string{"/api/v1/portfolios", "/api/v1/portfolios?status=ARCHIVED"} {
		response := perform(t, app, "GET", path, "", "")
		if response.Code != fiber.StatusOK {
			t.Fatalf("%s: status=%d", path, response.Code)
		}
	}
	if len(statuses) != 2 || statuses[0] != domain.PortfolioStatusActive || statuses[1] != domain.PortfolioStatusArchived {
		t.Fatalf("statuses=%v", statuses)
	}
	assertError(t, perform(t, app, "GET", "/api/v1/portfolios?status=active", "", ""), fiber.StatusBadRequest, "INVALID_REQUEST")
}

func TestHandlerMapsOperationErrorsAndCorrelations(t *testing.T) {
	principal := testPrincipal(t)
	portfolio := testPortfolio(t, principal)
	app := testApp(t, operationFake{get: func(context.Context, identitydomain.Principal, domain.PortfolioID) (domain.Portfolio, error) {
		return domain.Portfolio{}, application.ErrPortfolioNotFound
	}, update: func(context.Context, identitydomain.Principal, domain.PortfolioID, application.UpdatePortfolioInput) (domain.Portfolio, error) {
		return domain.Portfolio{}, application.ErrPortfolioArchived
	}, archive: func(context.Context, identitydomain.Principal, domain.PortfolioID) (domain.Portfolio, error) {
		return portfolio, nil
	}})
	id := portfolio.ID().String()
	response := perform(t, app, "GET", "/api/v1/portfolios/"+id, "", "")
	assertError(t, response, fiber.StatusNotFound, "PORTFOLIO_NOT_FOUND")
	response = perform(t, app, "PATCH", "/api/v1/portfolios/"+id, `{"name":"Changed"}`, "application/json")
	assertError(t, response, fiber.StatusUnprocessableEntity, "PORTFOLIO_ARCHIVED")
	response = perform(t, app, "GET", "/api/v1/portfolios/not-a-uuid", "", "")
	assertError(t, response, fiber.StatusNotFound, "PORTFOLIO_NOT_FOUND")
	response = perform(t, app, "PATCH", "/api/v1/portfolios/not-a-uuid", `{"name":"Changed"}`, "application/json")
	assertError(t, response, fiber.StatusNotFound, "PORTFOLIO_NOT_FOUND")
	response = perform(t, app, "POST", "/api/v1/portfolios/not-a-uuid/archive", "", "")
	assertError(t, response, fiber.StatusNotFound, "PORTFOLIO_NOT_FOUND")
}

func TestHandlerRequiresInjectedBearerMiddleware(t *testing.T) {
	app := fiber.New()
	handler, err := NewHandler(operationFake{}, func(ctx *fiber.Ctx) error {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid"})
	}, func(*fiber.Ctx) (identitydomain.Principal, bool) { return identitydomain.Principal{}, false })
	if err != nil {
		t.Fatal(err)
	}
	api := app.Group("/api/v1")
	handler.Mount(api)
	response := perform(t, app, "GET", "/api/v1/portfolios", "", "")
	if response.Code != fiber.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestMountedRoutesReuseIdentityBearerAuthentication(t *testing.T) {
	principal := testPrincipal(t)
	identity := identityOperationFake{principal: principal}
	authHandler, err := identityhttp.NewHandler(&identity, "https://app.localhost:3443", alwaysHTTPS{})
	if err != nil {
		t.Fatal(err)
	}
	portfolioHandler, err := NewHandler(operationFake{list: func(context.Context, identitydomain.Principal, domain.PortfolioStatus) ([]domain.Portfolio, error) {
		return nil, nil
	}}, authHandler.BearerMiddleware(), authHandler.PrincipalExtractor())
	if err != nil {
		t.Fatal(err)
	}
	server := platformhttp.New(slog.New(slog.NewTextHandler(io.Discard, nil)), ready{}, authHandler, portfolioHandler)
	for _, authorization := range []string{"", "Basic nope", "Bearer malformed"} {
		request := httptest.NewRequest("GET", "/api/v1/portfolios", nil)
		request.Header.Set(fiber.HeaderAuthorization, authorization)
		response, testErr := server.App().Test(request)
		if testErr != nil {
			t.Fatal(testErr)
		}
		if response.StatusCode != fiber.StatusUnauthorized {
			t.Fatalf("authorization=%q status=%d", authorization, response.StatusCode)
		}
		_ = response.Body.Close()
	}
	request := httptest.NewRequest("GET", "/api/v1/portfolios", nil)
	request.Header.Set(fiber.HeaderAuthorization, "Bearer valid-token")
	response, err := server.App().Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("valid bearer status=%d", response.StatusCode)
	}
	_ = response.Body.Close()

	request = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	response, err = server.App().Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("auth routes are not mounted: status=%d", response.StatusCode)
	}
	_ = response.Body.Close()
}

func testApp(t *testing.T, operations operationFake) *fiber.App {
	t.Helper()
	principal := testPrincipal(t)
	bearer := func(ctx *fiber.Ctx) error { ctx.Locals("principal", principal); return ctx.Next() }
	extract := func(ctx *fiber.Ctx) (identitydomain.Principal, bool) {
		value, ok := ctx.Locals("principal").(identitydomain.Principal)
		return value, ok
	}
	handler, err := NewHandler(operations, bearer, extract)
	if err != nil {
		t.Fatal(err)
	}
	server := platformhttp.New(slog.New(slog.NewTextHandler(io.Discard, nil)), ready{}, handler)
	return server.App()
}

type ready struct{}

func (ready) Ping(context.Context) error { return nil }

type alwaysHTTPS struct{}

func (alwaysHTTPS) OriginalRequestWasHTTPS(*fiber.Ctx) bool { return true }

type identityOperationFake struct{ principal identitydomain.Principal }

func (fake *identityOperationFake) Register(context.Context, identityapplication.CredentialsInput) (identityapplication.SessionResult, error) {
	return identityapplication.SessionResult{}, identityapplication.ErrAuthenticationService
}
func (fake *identityOperationFake) Login(context.Context, identityapplication.CredentialsInput) (identityapplication.SessionResult, error) {
	return identityapplication.SessionResult{}, identityapplication.ErrAuthenticationService
}
func (fake *identityOperationFake) Refresh(context.Context, identityapplication.RefreshInput) (identityapplication.SessionResult, error) {
	return identityapplication.SessionResult{}, identityapplication.ErrAuthenticationService
}
func (fake *identityOperationFake) Logout(context.Context, identityapplication.RefreshInput) error {
	return identityapplication.ErrAuthenticationService
}
func (fake *identityOperationFake) CurrentUser(context.Context, identitydomain.Principal) (identityapplication.SafeUser, error) {
	return identityapplication.SafeUser{}, identityapplication.ErrAccessTokenInvalid
}
func (fake *identityOperationFake) ResolvePrincipal(_ context.Context, token identityapplication.AccessToken) (identitydomain.Principal, error) {
	if token.Value() != "valid-token" {
		return identitydomain.Principal{}, identityapplication.ErrAccessTokenInvalid
	}
	return fake.principal, nil
}

func perform(t *testing.T, app *fiber.App, method, path, payload, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(payload))
	if contentType != "" {
		request.Header.Set(fiber.HeaderContentType, contentType)
	}
	request.Header.Set(platformhttp.CorrelationHeader, "portfolio-transport-test")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	recorder.Code = response.StatusCode
	recorder.HeaderMap = response.Header
	_, _ = io.Copy(recorder.Body, response.Body)
	_ = response.Body.Close()
	return recorder
}

func decode(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatal(err)
	}
}

func assertError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body platformhttp.ErrorEnvelope
	decode(t, response, &body)
	if body.Error.Code != code || body.Error.CorrelationID != response.Header().Get(platformhttp.CorrelationHeader) {
		t.Fatalf("error=%#v header=%q", body.Error, response.Header().Get(platformhttp.CorrelationHeader))
	}
}

func testPrincipal(t *testing.T) identitydomain.Principal {
	t.Helper()
	userID, err := identitydomain.NewUserID(uuid.MustParse("10000000-0000-0000-0000-000000000001"))
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identitydomain.NewPrincipal(userID)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func testPortfolio(t *testing.T, principal identitydomain.Principal) domain.Portfolio {
	t.Helper()
	owner, _ := principal.UserID()
	id, err := domain.NewPortfolioID(uuid.MustParse("20000000-0000-0000-0000-000000000001"))
	if err != nil {
		t.Fatal(err)
	}
	name, err := domain.NewPortfolioName("Primary")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	portfolio, err := domain.NewPortfolio(id, owner, name, domain.BaseCurrencyUSD, at)
	if err != nil {
		t.Fatal(err)
	}
	return portfolio
}
