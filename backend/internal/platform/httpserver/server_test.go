package httpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type testRegistrar struct{}

func (testRegistrar) Mount(router fiber.Router) {
	router.Post("/auth/register", func(ctx *fiber.Ctx) error { return ctx.SendStatus(fiber.StatusNoContent) })
}

func TestV1RouteRegistrarMountsWithoutChangingHealthRoutes(t *testing.T) {
	server := New(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{}, testRegistrar{})
	for _, path := range []string{"/api/v1/health/live", "/api/v1/health/ready"} {
		response, err := server.App().Test(httptest.NewRequest(http.MethodGet, path, nil))
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d", path, response.StatusCode)
		}
	}
	response, err := server.App().Test(httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("route status = %d", response.StatusCode)
	}
}

type readinessStub struct{ err error }

func (stub readinessStub) Ping(context.Context) error { return stub.err }

func TestHealthPropagatesCorrelationID(t *testing.T) {
	server := New(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{})
	request := httptest.NewRequest("GET", "/api/v1/health/live", nil)
	request.Header.Set(correlationHeader, "bootstrap-test-correlation")
	response, err := server.App().Test(request)
	if err != nil {
		t.Fatalf("request health endpoint: %v", err)
	}
	if response.StatusCode != 200 || response.Header.Get(correlationHeader) != "bootstrap-test-correlation" {
		t.Fatalf("unexpected health response: status=%d correlation=%q", response.StatusCode, response.Header.Get(correlationHeader))
	}
}

func TestReadinessReflectsDatabaseFailure(t *testing.T) {
	server := New(slog.New(slog.NewTextHandler(io.Discard, nil)), readinessStub{err: errors.New("database unavailable")})
	request := httptest.NewRequest("GET", "/api/v1/health/ready", nil)
	response, err := server.App().Test(request)
	if err != nil {
		t.Fatalf("request readiness endpoint: %v", err)
	}
	if response.StatusCode != 503 {
		t.Fatalf("expected status 503, got %d", response.StatusCode)
	}
}
