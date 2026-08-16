package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platformhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
)

type operationsFake struct {
	get    func(context.Context, identitydomain.Principal, domain.AssetID) (domain.Asset, error)
	search func(context.Context, identitydomain.Principal, application.SearchInput) (application.SearchResult, error)
}

func (fake operationsFake) GetAsset(ctx context.Context, principal identitydomain.Principal, id domain.AssetID) (domain.Asset, error) {
	return fake.get(ctx, principal, id)
}
func (fake operationsFake) SearchAssets(ctx context.Context, principal identitydomain.Principal, input application.SearchInput) (application.SearchResult, error) {
	return fake.search(ctx, principal, input)
}

func TestListValidatesFrozenQueryAndCursorContract(t *testing.T) {
	asset := transportAsset(t, "AAA", "NYSE", domain.AssetTypeEquity)
	seen := application.SearchInput{}
	app := assetTestApp(t, operationsFake{search: func(_ context.Context, _ identitydomain.Principal, input application.SearchInput) (application.SearchResult, error) {
		seen = input
		return application.SearchResult{Assets: []domain.Asset{asset}, Next: &application.CursorPosition{Symbol: asset.Symbol(), Exchange: asset.Exchange(), AssetID: asset.ID()}}, nil
	}})
	response := assetRequest(t, app, "GET", "/api/v1/assets", "")
	if response.Code != fiber.StatusOK || seen.Limit != application.DefaultSearchLimit {
		t.Fatalf("status=%d input=%+v", response.Code, seen)
	}
	var body assetListResponse
	decodeAsset(t, response, &body)
	if len(body.Items) != 1 || body.NextCursor == nil || body.Items[0].ID != asset.ID().String() {
		t.Fatalf("response=%#v", body)
	}
	for _, path := range []string{"/api/v1/assets?search=", "/api/v1/assets?search=" + strings.Repeat("a", 101), "/api/v1/assets?type=equity", "/api/v1/assets?limit=0", "/api/v1/assets?limit=101", "/api/v1/assets?limit=nope", "/api/v1/assets?cursor=not-a-cursor"} {
		assertAssetError(t, assetRequest(t, app, "GET", path, ""), fiber.StatusBadRequest, "INVALID_REQUEST")
	}
	response = assetRequest(t, app, "GET", "/api/v1/assets?type=EQUITY&limit=1&cursor="+*body.NextCursor, "")
	if response.Code != fiber.StatusOK || seen.AssetType == nil || *seen.AssetType != domain.AssetTypeEquity || seen.Limit != 1 || seen.Cursor == nil {
		t.Fatalf("input=%+v status=%d", seen, response.Code)
	}
}

func TestGetUsesOpaqueNotFoundAndApprovedDTO(t *testing.T) {
	asset := transportAsset(t, "BTC", "CRYPTO", domain.AssetTypeCrypto)
	app := assetTestApp(t, operationsFake{get: func(_ context.Context, _ identitydomain.Principal, id domain.AssetID) (domain.Asset, error) {
		if id == asset.ID() {
			return asset, nil
		}
		return domain.Asset{}, application.ErrAssetNotFound
	}})
	response := assetRequest(t, app, "GET", "/api/v1/assets/"+asset.ID().String(), "")
	if response.Code != fiber.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	var body map[string]any
	decodeAsset(t, response, &body)
	if len(body) != 6 || body["exchange"] != "CRYPTO" || body["currency"] != "USD" {
		t.Fatalf("dto=%#v", body)
	}
	assertAssetError(t, assetRequest(t, app, "GET", "/api/v1/assets/not-a-uuid", ""), fiber.StatusNotFound, "ASSET_NOT_FOUND")
}

func TestAssetRoutesRequireInjectedBearer(t *testing.T) {
	app := fiber.New()
	handler, err := NewHandler(operationsFake{}, func(ctx *fiber.Ctx) error {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid"})
	}, func(*fiber.Ctx) (identitydomain.Principal, bool) { return identitydomain.Principal{}, false })
	if err != nil {
		t.Fatal(err)
	}
	handler.Mount(app.Group("/api/v1"))
	response := assetRequest(t, app, "GET", "/api/v1/assets", "")
	if response.Code != fiber.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
	mutationApp := assetTestApp(t, operationsFake{})
	for _, method := range []string{"POST", "PATCH", "DELETE"} {
		request := httptest.NewRequest(method, "/api/v1/assets", nil)
		responseRaw, testErr := mutationApp.Test(request)
		if testErr != nil {
			t.Fatal(testErr)
		}
		if responseRaw.StatusCode != fiber.StatusMethodNotAllowed {
			t.Fatalf("%s mutation status=%d", method, responseRaw.StatusCode)
		}
		_ = responseRaw.Body.Close()
	}
}

func TestCursorStrictRoundTrip(t *testing.T) {
	asset := transportAsset(t, "AAA", "NYSE", domain.AssetTypeEquity)
	value, err := encodeCursor(application.CursorPosition{Symbol: asset.Symbol(), Exchange: asset.Exchange(), AssetID: asset.ID()})
	if err != nil || len(value) > 512 {
		t.Fatalf("cursor=%q err=%v", value, err)
	}
	decoded, err := decodeCursor(value)
	if err != nil || decoded.AssetID != asset.ID() || decoded.Symbol != asset.Symbol() {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
	payload := cursorPayload{Version: cursorVersion, Symbol: asset.Symbol(), Exchange: asset.Exchange(), AssetID: strings.ToUpper(asset.ID().String())}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCursor(base64.RawURLEncoding.EncodeToString(encoded)); err == nil {
		t.Fatal("expected non-canonical cursor asset ID rejection")
	}
}

func assetTestApp(t *testing.T, operations operationsFake) *fiber.App {
	t.Helper()
	principal := transportPrincipal(t)
	bearer := func(ctx *fiber.Ctx) error { ctx.Locals("asset-principal", principal); return ctx.Next() }
	extract := func(ctx *fiber.Ctx) (identitydomain.Principal, bool) {
		value, ok := ctx.Locals("asset-principal").(identitydomain.Principal)
		return value, ok
	}
	handler, err := NewHandler(operations, bearer, extract)
	if err != nil {
		t.Fatal(err)
	}
	return platformhttp.New(slog.New(slog.NewTextHandler(io.Discard, nil)), transportReady{}, handler).App()
}

type transportReady struct{}

func (transportReady) Ping(context.Context) error { return nil }
func assetRequest(t *testing.T, app *fiber.App, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set(platformhttp.CorrelationHeader, "asset-transport-test")
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
func decodeAsset(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatal(err)
	}
}
func assertAssetError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body platformhttp.ErrorEnvelope
	decodeAsset(t, response, &body)
	if body.Error.Code != code || body.Error.CorrelationID != response.Header().Get(platformhttp.CorrelationHeader) {
		t.Fatalf("error=%#v header=%q", body.Error, response.Header().Get(platformhttp.CorrelationHeader))
	}
}
func transportPrincipal(t *testing.T) identitydomain.Principal {
	t.Helper()
	id, _ := identitydomain.NewUserID(uuid.New())
	principal, err := identitydomain.NewPrincipal(id)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}
func transportAsset(t *testing.T, symbol, exchange string, assetType domain.AssetType) domain.Asset {
	t.Helper()
	id, _ := domain.NewAssetID(uuid.New())
	now := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	asset, err := domain.RehydrateAsset(id, symbol, "Asset "+symbol, assetType, exchange, domain.CurrencyUSD, symbol, exchange, now, now)
	if err != nil {
		t.Fatal(err)
	}
	return asset
}
