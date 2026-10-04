package http

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/gofiber/fiber/v2"
)

type createOperation struct {
	Operations
	call func(identity.Principal, portfolio.PortfolioID, application.CommandInput, application.CommandMetadata)
}

func (operation createOperation) Create(_ context.Context, principal identity.Principal, id portfolio.PortfolioID, input application.CommandInput, metadata application.CommandMetadata) (application.Result, error) {
	operation.call(principal, id, input, metadata)
	return application.Result{}, errors.New("private database detail")
}

func TestTrustedPrincipalMetadataAndInternalFailure(t *testing.T) {
	uid, _ := identity.ParseUserID(testID)
	principal, _ := identity.NewPrincipal(uid)
	calls, bearerCalls := 0, 0
	operation := createOperation{call: func(actual identity.Principal, id portfolio.PortfolioID, input application.CommandInput, metadata application.CommandMetadata) {
		calls++
		if actual != principal || id.String() != testID || input.Amount.String() != "123.123456789012" || metadata.IdempotencyKey != "request-key-12345" || metadata.CorrelationID != "trusted-correlation" {
			t.Fatal("untrusted or altered application input")
		}
	}}
	bearer := func(ctx *fiber.Ctx) error { bearerCalls++; return ctx.Next() }
	extract := func(*fiber.Ctx) (identity.Principal, bool) { return principal, true }
	handler, err := NewHandler(operation, bearer, extract)
	if err != nil {
		t.Fatal(err)
	}
	server := platform.New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, handler)
	request := httptest.NewRequest("POST", "/api/v1/portfolios/"+testID+"/transactions", strings.NewReader(cashCommand))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "request-key-12345")
	request.Header.Set("X-Correlation-ID", "trusted-correlation")
	request.Header.Set("X-User-ID", "attacker")
	response, err := server.App().Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 500 || calls != 1 || bearerCalls != 1 || strings.Contains(string(body), "private") || !strings.Contains(string(body), "INTERNAL_ERROR") || !strings.Contains(string(body), "trusted-correlation") {
		t.Fatal("unsafe error or dispatch")
	}
	for _, test := range []struct {
		operations Operations
		bearer     fiber.Handler
		principal  PrincipalExtractor
	}{{nil, bearer, extract}, {operation, nil, extract}, {operation, bearer, nil}} {
		if _, err := NewHandler(test.operations, test.bearer, test.principal); err == nil {
			t.Fatal("missing dependency accepted")
		}
	}
}

func TestMissingExtractedPrincipalFailsClosed(t *testing.T) {
	operation := createOperation{call: func(identity.Principal, portfolio.PortfolioID, application.CommandInput, application.CommandMetadata) {
		t.Fatal("operation called without principal")
	}}
	handler, err := NewHandler(operation, func(ctx *fiber.Ctx) error { return ctx.Next() }, func(*fiber.Ctx) (identity.Principal, bool) { return identity.Principal{}, false })
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	handler.Mount(app.Group("/api/v1"))
	for _, route := range []struct{ method, path string }{{"POST", ""}, {"GET", ""}, {"GET", "/" + testID}, {"POST", "/" + testID + "/corrections"}} {
		res, err := app.Test(httptest.NewRequest(route.method, "/api/v1/portfolios/"+testID+"/transactions"+route.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 401 {
			t.Fatal("principal missing but request accepted")
		}
	}
}
