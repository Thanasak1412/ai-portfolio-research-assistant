//go:build integration

package database_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/composition"
	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/composition"
	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/composition"
	transaction "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/composition"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func apiMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// An isolated synthetic schema, never a shared developer/production database.
func apiPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	config, err := pgxpool.ParseConfig(url)
	apiMust(t, err)
	if !strings.HasSuffix(config.ConnConfig.Database, "_test") {
		t.Fatal("requires explicitly named _test database")
	}
	ctx := context.Background()
	admin, err := pgxpool.NewWithConfig(ctx, config)
	apiMust(t, err)
	schema := "m3http_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = admin.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize())
	apiMust(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec(ctx, "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		if err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	copyConfig := config.Copy()
	copyConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, copyConfig)
	apiMust(t, err)
	t.Cleanup(pool.Close)
	files, err := filepath.Glob("../../../../migrations/[0-9]*.sql")
	apiMust(t, err)
	if len(files) != 5 {
		t.Fatal("unexpected migration inventory")
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		apiMust(t, err)
		tx, err := pool.Begin(ctx)
		apiMust(t, err)
		_, err = tx.Exec(ctx, strings.Split(string(data), "-- +goose Down")[0])
		if err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
		apiMust(t, tx.Commit(ctx))
	}
	return pool
}

func apiLookup(t *testing.T) identity.LookupFunc {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	apiMust(t, err)
	privateDER, err := x509.MarshalPKCS8PrivateKey(private)
	apiMust(t, err)
	publicDER, err := x509.MarshalPKIXPublicKey(public)
	apiMust(t, err)
	keys, err := json.Marshal([]map[string]string{{"kid": "auth-ed25519-20260808-01", "publicKeyB64": base64.StdEncoding.EncodeToString(publicDER)}})
	apiMust(t, err)
	network, rate := make([]byte, 32), make([]byte, 32)
	_, err = rand.Read(network)
	apiMust(t, err)
	_, err = rand.Read(rate)
	apiMust(t, err)
	values := map[string]string{
		"AUTH_PUBLIC_ORIGIN": "https://app.localhost:3443", "AUTH_JWT_ACTIVE_KID": "auth-ed25519-20260808-01",
		"AUTH_JWT_ACTIVE_PRIVATE_KEY_B64": base64.StdEncoding.EncodeToString(privateDER), "AUTH_JWT_VERIFICATION_KEYS_JSON": string(keys),
		"AUTH_NETWORK_HMAC_KEY": base64.StdEncoding.EncodeToString(network), "AUTH_RATE_LIMIT_HMAC_KEY": base64.StdEncoding.EncodeToString(rate),
	}
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}

func apiRuntime(t *testing.T, pool *pgxpool.Pool) *fiber.App {
	t.Helper()
	auth, err := identity.BuildHTTP(context.Background(), pool, "test", apiLookup(t))
	apiMust(t, err)
	ledger, err := transaction.BuildHTTP(pool, portfolio.BindOwnership, asset.BindLookup, auth.BearerMiddleware(), auth.PrincipalExtractor())
	apiMust(t, err)
	portfolios, err := portfolio.BuildHTTP(pool, auth.BearerMiddleware(), auth.PrincipalExtractor())
	apiMust(t, err)
	assets, err := asset.BuildHTTP(pool, auth.BearerMiddleware(), auth.PrincipalExtractor())
	apiMust(t, err)
	return platform.New(slog.New(slog.NewTextHandler(io.Discard, nil)), pool, auth, ledger, portfolios, assets).App()
}

type apiResult map[string]json.RawMessage

func (r apiResult) text(t *testing.T, key string) string {
	t.Helper()
	var value string
	apiMust(t, json.Unmarshal(r[key], &value))
	return value
}
func (r apiResult) child(t *testing.T, key string) apiResult {
	t.Helper()
	var value apiResult
	apiMust(t, json.Unmarshal(r[key], &value))
	return value
}

func apiRequest(t *testing.T, app *fiber.App, method, path, token, key, body string, status int, code string) apiResult {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", "m3-http-integration")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	response, err := app.Test(request, 10000)
	apiMust(t, err)
	defer response.Body.Close()
	var result apiResult
	apiMust(t, json.NewDecoder(response.Body).Decode(&result))
	if response.StatusCode != status {
		// Never print response bodies: Authentication results contain credentials.
		var publicCode string
		if result["error"] != nil {
			publicCode = result.child(t, "error").text(t, "code")
		}
		t.Fatalf("%s %s status=%d want=%d code=%s", method, path, response.StatusCode, status, publicCode)
	}
	if response.Header.Get("X-Correlation-ID") != "m3-http-integration" {
		t.Fatal("correlation header lost")
	}
	if code != "" {
		envelope := result.child(t, "error")
		if envelope.text(t, "code") != code || envelope.text(t, "correlationId") != "m3-http-integration" {
			t.Fatalf("wrong error envelope: code=%s want=%s correlation=%s", envelope.text(t, "code"), code, envelope.text(t, "correlationId"))
		}
	}
	return result
}

func apiAccount(t *testing.T, app *fiber.App) string {
	t.Helper()
	credentials, _ := json.Marshal(map[string]string{"email": "m3-http-" + uuid.NewString() + "@example.test", "password": uuid.NewString() + uuid.NewString()})
	return apiRequest(t, app, "POST", "/api/v1/auth/register", "", "", string(credentials), 201, "").text(t, "accessToken")
}
func apiPortfolio(t *testing.T, app *fiber.App, token string) string {
	t.Helper()
	return apiRequest(t, app, "POST", "/api/v1/portfolios", token, "", `{"name":"`+uuid.NewString()+`","baseCurrency":"USD"}`, 201, "").text(t, "id")
}

func apiAsset(t *testing.T, pool *pgxpool.Pool, kind, exchange string) string {
	t.Helper()
	id := uuid.NewString()
	_, err := pool.Exec(context.Background(), `INSERT INTO assets(asset_id,symbol,name,asset_type,exchange,currency,created_at,updated_at) VALUES($1,$2,'Synthetic HTTP fixture',$3,$4,'USD',now(),now())`, id, "HTTP"+strings.ReplaceAll(uuid.NewString(), "-", ""), kind, exchange)
	apiMust(t, err)
	return id
}

func apiCommand(kind, asset, amount, at string) string {
	fields := map[string]string{"kind": kind, "currency": "USD", "effectiveAt": at}
	if kind == "BUY" || kind == "SELL" {
		fields["quantity"], fields["unitPrice"] = amount, "10.123456789012"
	} else {
		fields["amount"] = amount
	}
	if asset != "" {
		fields["assetId"] = asset
	}
	body, _ := json.Marshal(fields)
	return string(body)
}
