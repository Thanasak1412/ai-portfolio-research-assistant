//go:build integration

package http

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	assetdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/infrastructure/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/infrastructure/database/sqlcgen"
	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
)

func TestAssetHTTPPostgresReadSearchAndGlobalAccess(t *testing.T) {
	pool := openAssetHTTPPool(t)
	queries := sqlcgen.New(pool)
	suffix := strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	now := time.Now().UTC().Truncate(time.Microsecond)
	equity := bootstrapHTTPAsset(t, queries, "AA"+suffix, "Atlas "+suffix, "EQUITY", "NYSE", now)
	bootstrapHTTPAsset(t, queries, "BB"+suffix, "Beacon "+suffix, "ETF", "NASDAQ", now)
	crypto := bootstrapHTTPAsset(t, queries, "CC"+suffix, "Cipher "+suffix, "CRYPTO", "CRYPTO", now)
	service, err := application.NewService(assetdatabase.NewPostgresAssetRepository(pool))
	if err != nil {
		t.Fatal(err)
	}
	app := assetHTTPIntegrationApp(t, service)

	response := assetHTTPRequest(t, app, "owner", "/api/v1/assets?type=EQUITY&search="+strings.ToLower("AA"+suffix)+"&limit=1")
	if response.Code != fiber.StatusOK {
		t.Fatalf("equity search status=%d body=%s", response.Code, response.Body.String())
	}
	var list assetListResponse
	decodeAsset(t, response, &list)
	if len(list.Items) != 1 || list.Items[0].ID != uuid.UUID(equity.AssetID.Bytes).String() {
		t.Fatalf("equity response=%#v", list)
	}
	response = assetHTTPRequest(t, app, "owner", "/api/v1/assets?type=CRYPTO&search="+suffix)
	if response.Code != fiber.StatusOK {
		t.Fatalf("crypto status=%d", response.Code)
	}
	decodeAsset(t, response, &list)
	if len(list.Items) != 1 || list.Items[0].Exchange != "CRYPTO" || list.Items[0].ID != uuid.UUID(crypto.AssetID.Bytes).String() {
		t.Fatalf("crypto=%#v", list)
	}
	response = assetHTTPRequest(t, app, "other", "/api/v1/assets/"+uuid.UUID(equity.AssetID.Bytes).String())
	if response.Code != fiber.StatusOK {
		t.Fatalf("global get status=%d", response.Code)
	}
}

func assetHTTPIntegrationApp(t *testing.T, service *application.Service) *fiber.App {
	t.Helper()
	owner, other := integrationPrincipal(t), integrationPrincipal(t)
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
		ctx.Locals("asset-integration-principal", principal)
		return ctx.Next()
	}
	extract := func(ctx *fiber.Ctx) (identitydomain.Principal, bool) {
		principal, ok := ctx.Locals("asset-integration-principal").(identitydomain.Principal)
		return principal, ok
	}
	handler, err := NewHandler(service, bearer, extract)
	if err != nil {
		t.Fatal(err)
	}
	return platformhttp.New(slog.New(slog.NewTextHandler(io.Discard, nil)), transportReady{}, handler).App()
}

func assetHTTPRequest(t *testing.T, app *fiber.App, actor, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest("GET", path, nil)
	request.Header.Set(fiber.HeaderAuthorization, "Bearer "+actor)
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
func openAssetHTTPPool(t *testing.T) *pgxpool.Pool {
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
func bootstrapHTTPAsset(t *testing.T, queries *sqlcgen.Queries, symbol, name, assetType, exchange string, now time.Time) sqlcgen.Asset {
	t.Helper()
	id := uuid.New()
	row, err := queries.BootstrapCanonicalAsset(context.Background(), sqlcgen.BootstrapCanonicalAssetParams{AssetID: pgtype.UUID{Bytes: id, Valid: true}, Symbol: symbol, Name: name, AssetType: assetType, Exchange: exchange, CreatedAt: pgtype.Timestamptz{Time: now, Valid: true}, UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true}})
	if err != nil {
		t.Fatalf("bootstrap %s: %v", symbol, err)
	}
	return row
}
func integrationPrincipal(t *testing.T) identitydomain.Principal {
	t.Helper()
	id, _ := identitydomain.NewUserID(uuid.New())
	principal, err := identitydomain.NewPrincipal(id)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}
