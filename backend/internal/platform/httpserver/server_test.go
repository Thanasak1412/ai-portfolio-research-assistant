package httpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
)

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
