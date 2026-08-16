//go:build integration

package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	portfoliodatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/infrastructure/database"
)

func TestPortfolioHTTPRepresentativePostgresFlow(t *testing.T) {
	pool := openTransportIntegrationPool(t)
	owner := insertTransportIntegrationUser(t, pool)
	other := insertTransportIntegrationUser(t, pool)
	service, err := application.NewService(application.ServiceDependencies{
		Repository: portfoliodatabase.NewPostgresPortfolioRepository(pool),
		Clock:      transportIntegrationClock{},
		IDs:        transportIntegrationIDs{},
	})
	if err != nil {
		t.Fatal(err)
	}
	app := transportIntegrationApp(t, service, owner, other)

	created := requestTransportPortfolio(t, app, "owner", "POST", "/api/v1/portfolios", `{"name":"Primary","baseCurrency":"USD"}`)
	if created.Code != fiber.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var portfolio portfolioResponse
	decode(t, created, &portfolio)
	if portfolio.Status != "ACTIVE" || portfolio.ArchivedAt != nil {
		t.Fatalf("created=%#v", portfolio)
	}

	got := requestTransportPortfolio(t, app, "owner", "GET", "/api/v1/portfolios/"+portfolio.ID, "")
	if got.Code != fiber.StatusOK {
		t.Fatalf("get status=%d", got.Code)
	}
	updated := requestTransportPortfolio(t, app, "owner", "PATCH", "/api/v1/portfolios/"+portfolio.ID, `{"name":"Renamed"}`)
	if updated.Code != fiber.StatusOK {
		t.Fatalf("update status=%d body=%s", updated.Code, updated.Body.String())
	}
	decode(t, updated, &portfolio)
	if portfolio.Name != "Renamed" {
		t.Fatalf("name=%q", portfolio.Name)
	}

	archived := requestTransportPortfolio(t, app, "owner", "POST", "/api/v1/portfolios/"+portfolio.ID+"/archive", "")
	if archived.Code != fiber.StatusOK {
		t.Fatalf("archive status=%d body=%s", archived.Code, archived.Body.String())
	}
	var firstArchive portfolioResponse
	decode(t, archived, &firstArchive)
	if firstArchive.Status != "ARCHIVED" || firstArchive.ArchivedAt == nil {
		t.Fatalf("archive=%#v", firstArchive)
	}
	retry := requestTransportPortfolio(t, app, "owner", "POST", "/api/v1/portfolios/"+portfolio.ID+"/archive", "")
	if retry.Code != fiber.StatusOK {
		t.Fatalf("archive retry status=%d", retry.Code)
	}
	var retried portfolioResponse
	decode(t, retry, &retried)
	if retried.ArchivedAt == nil || !retried.ArchivedAt.Equal(*firstArchive.ArchivedAt) {
		t.Fatalf("archive retry changed timestamp: first=%v retry=%v", firstArchive.ArchivedAt, retried.ArchivedAt)
	}

	crossOwner := requestTransportPortfolio(t, app, "other", "GET", "/api/v1/portfolios/"+portfolio.ID, "")
	assertError(t, crossOwner, fiber.StatusNotFound, "PORTFOLIO_NOT_FOUND")
}

func transportIntegrationApp(t *testing.T, service *application.Service, owner, other identitydomain.Principal) *fiber.App {
	t.Helper()
	bearer := func(ctx *fiber.Ctx) error {
		var principal identitydomain.Principal
		switch ctx.Get(fiber.HeaderAuthorization) {
		case "Bearer owner":
			principal = owner
		case "Bearer other":
			principal = other
		default:
			return ctx.Status(fiber.StatusUnauthorized).JSON(platformhttp.ErrorEnvelope{Error: platformhttp.ErrorDetail{Code: "ACCESS_TOKEN_INVALID", Message: "The access token is invalid", CorrelationID: platformhttp.CorrelationID(ctx)}})
		}
		ctx.Locals("integration-principal", principal)
		return ctx.Next()
	}
	extract := func(ctx *fiber.Ctx) (identitydomain.Principal, bool) {
		value, ok := ctx.Locals("integration-principal").(identitydomain.Principal)
		return value, ok
	}
	handler, err := NewHandler(service, bearer, extract)
	if err != nil {
		t.Fatal(err)
	}
	return platformhttp.New(slog.New(slog.NewTextHandler(io.Discard, nil)), ready{}, handler).App()
}

func requestTransportPortfolio(t *testing.T, app *fiber.App, actor, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(fiber.HeaderAuthorization, "Bearer "+actor)
	if body != "" {
		request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	recorder.Code, recorder.HeaderMap = response.StatusCode, response.Header
	_, _ = io.Copy(recorder.Body, response.Body)
	_ = response.Body.Close()
	return recorder
}

func openTransportIntegrationPool(t *testing.T) *pgxpool.Pool {
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

func insertTransportIntegrationUser(t *testing.T, pool *pgxpool.Pool) identitydomain.Principal {
	t.Helper()
	id, err := identitydomain.NewUserID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := pool.Exec(context.Background(), `INSERT INTO users (user_id, normalized_email, password_hash, account_status, created_at, updated_at, disabled_at) VALUES ($1, $2, $3, 'active', $4, $4, NULL)`, id.String(), fmt.Sprintf("portfolio-http-%s@example.test", uuid.NewString()), "$argon2id$v=19$m=1,t=1,p=1$fixture$fixture", now); err != nil {
		t.Fatal(err)
	}
	principal, err := identitydomain.NewPrincipal(id)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

type transportIntegrationClock struct{}

func (transportIntegrationClock) Now() time.Time { return time.Now().UTC() }

type transportIntegrationIDs struct{}

func (transportIntegrationIDs) PortfolioID() (domain.PortfolioID, error) {
	return domain.NewPortfolioID(uuid.New())
}
