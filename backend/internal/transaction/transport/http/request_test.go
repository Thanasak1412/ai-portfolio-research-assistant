package http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/gofiber/fiber/v2"
)

const cashCommand = `{"kind":"DEPOSIT","currency":"USD","amount":"123.123456789012","effectiveAt":"2026-01-01T00:00:00.123456Z"}`
const testID = "1c66044f-9f1c-4690-b2fc-2e795910de08"

func TestCommandMatrixAndExactText(t *testing.T) {
	for _, kind := range []string{"BUY", "SELL", "DIVIDEND", "DEPOSIT", "WITHDRAWAL", "FEE"} {
		t.Run(kind, func(t *testing.T) {
			fields := map[string]string{"kind": kind, "currency": "USD", "effectiveAt": "2026-01-01T00:00:00.123456Z", "note": "  e\u0301 🪙  ", "externalReference": ""}
			if kind == "BUY" || kind == "SELL" {
				fields["quantity"], fields["unitPrice"], fields["assetId"] = "1.123456789012", "10", testID
			} else {
				fields["amount"] = "10"
			}
			if kind == "DIVIDEND" {
				fields["assetId"] = testID
			}
			body, _ := json.Marshal(fields)
			for _, correction := range []bool{false, true} {
				command := body
				if correction {
					command = append(append([]byte(`{"replacement":`), body...), '}')
				}
				input, err := parseCommand(command, correction)
				if err != nil {
					t.Fatal(err)
				}
				if note, ok := input.Note.Value(); !ok || note != fields["note"] {
					t.Fatal("note changed")
				}
				if external, ok := input.ExternalReference.Value(); !ok || external != "" {
					t.Fatal("empty became absent")
				}
			}
			for _, field := range []string{"quantity", "unitPrice", "fee", "amount", "assetId"} {
				_, present := fields[field]
				if field == "fee" && (kind == "BUY" || kind == "SELL") {
					continue
				}
				copyFields := make(map[string]string)
				for k, v := range fields {
					copyFields[k] = v
				}
				if present {
					delete(copyFields, field)
				} else {
					copyFields[field] = "1"
				}
				invalid, _ := json.Marshal(copyFields)
				if _, err := parseCommand(invalid, false); !errors.Is(err, domain.ErrInvalidTransactionFields) {
					t.Fatalf("%s: %v", field, err)
				}
			}
		})
	}
}

func TestStrictCommands(t *testing.T) {
	add := func(field string) string { return strings.TrimSuffix(cashCommand, "}") + "," + field + "}" }
	for name, body := range map[string]string{
		"null": "null", "array": "[]", "empty": "{}", "trailing": cashCommand + "{}", "duplicate": add(`"amount":"2"`),
		"case": strings.Replace(cashCommand, "kind", "Kind", 1), "null optional": add(`"note":null`), "number": strings.Replace(cashCommand, `"123.123456789012"`, `123`, 1),
		"owner": add(`"ownerUserId":"x"`), "portfolio": add(`"portfolioId":"x"`), "gross": add(`"gross":"1"`), "net": add(`"net":"1"`),
		"unknown": add(`"unknown":true`), "note bound": add(`"note":"` + strings.Repeat("x", 2001) + `"`), "reference bound": add(`"externalReference":"` + strings.Repeat("x", 257) + `"`),
		"reversal": strings.Replace(cashCommand, "DEPOSIT", "REVERSAL", 1), "adjustment": strings.Replace(cashCommand, "DEPOSIT", "ADJUSTMENT", 1),
		"currency": strings.Replace(cashCommand, "USD", "EUR", 1), "offset": strings.Replace(cashCommand, "123456Z", "123456+00:00", 1), "precision": strings.Replace(cashCommand, "123456Z", "1234567Z", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseCommand([]byte(body), false); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	for _, value := range []string{"0", "-1", "+1", "1e2", "01", "1.", ".1", " 1", "0.1234567890123"} {
		body := strings.Replace(cashCommand, "123.123456789012", value, 1)
		if _, err := parseCommand([]byte(body), false); !errors.Is(err, domain.ErrInvalidDecimal) {
			t.Fatalf("decimal %s: %v", value, err)
		}
	}
	for _, body := range []string{`{}`, `{"replacement":null}`, `{"replacement":` + cashCommand + `,"original":"x"}`, `{"replacement":` + cashCommand + `,"replacement":` + cashCommand + `}`} {
		if _, err := parseCommand([]byte(body), true); err == nil {
			t.Fatal("invalid correction accepted")
		}
	}
}

func TestBodyBoundaryAndKeys(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(ctx *fiber.Ctx) error {
		_, err := decodeCommand(ctx, false)
		if err != nil {
			return writeError(ctx, err, false)
		}
		return ctx.SendStatus(204)
	})
	for _, size := range []int{8192, 8193} {
		body := cashCommand + strings.Repeat(" ", size-len(cashCommand))
		request := httptest.NewRequest("POST", "/", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		want := 204
		if size > 8192 {
			want = 400
		}
		if response.StatusCode != want {
			t.Fatalf("size %d status %d", size, response.StatusCode)
		}
	}
	keys := fiber.New()
	keys.Post("/", func(ctx *fiber.Ctx) error {
		_, err := metadata(ctx, "correlation")
		if err != nil {
			return writeError(ctx, err, false)
		}
		return ctx.SendStatus(204)
	})
	for _, key := range []string{"", strings.Repeat("a", 15), strings.Repeat("a", 16), strings.Repeat("a", 128), strings.Repeat("a", 129), "a-valid-key-00001", "bad key has spaces", "_invalid-first-01"} {
		req := httptest.NewRequest("POST", "/", nil)
		req.Header.Set("Idempotency-Key", key)
		res, err := keys.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		want := 400
		if keyGrammar.MatchString(key) {
			want = 204
		}
		if res.StatusCode != want {
			t.Fatalf("key status %d", res.StatusCode)
		}
	}
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Add("Idempotency-Key", "a-valid-key-00001")
	req.Header.Add("Idempotency-Key", "b-valid-key-00002")
	res, err := keys.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatal("duplicate key accepted")
	}
}

func TestCursorAndHistoryBoundaries(t *testing.T) {
	id, _ := domain.ParseTransactionID(testID)
	position := application.Position{EffectiveAt: time.Date(2026, 1, 1, 0, 0, 0, 123456000, time.UTC), Sequence: 42, ID: id}
	cursor := encodeCursor(position)
	decoded, err := decodeCursor(cursor)
	if err != nil || *decoded != position {
		t.Fatalf("roundtrip: %v", err)
	}
	for _, value := range []string{"", "v2." + strings.TrimPrefix(cursor, "v1."), cursor + "=", strings.Repeat("a", 513), "v1.%%%"} {
		if _, err := decodeCursor(value); err == nil {
			t.Fatal("invalid cursor accepted")
		}
	}
	for _, tuple := range []string{`["2026-01-01T00:00:00Z","0","` + testID + `"]`, `["2026-01-01T00:00:00Z","01","` + testID + `"]`, `["2026-01-01T00:00:00Z","1","invalid"]`, `["2026-01-01T00:00:00Z","1","` + testID + `","extra"]`, `[ "2026-01-01T00:00:00Z","1","` + testID + `"]`} {
		if _, err := decodeCursor("v1." + base64.RawURLEncoding.EncodeToString([]byte(tuple))); err == nil {
			t.Fatal("noncanonical cursor accepted")
		}
	}
	app := fiber.New()
	app.Get("/", func(ctx *fiber.Ctx) error {
		_, err := historyInput(ctx)
		if err != nil {
			return writeError(ctx, err, false)
		}
		return ctx.SendStatus(204)
	})
	for query, want := range map[string]int{"": 204, "limit=1": 204, "limit=100": 204, "limit=0": 400, "limit=101": 400, "limit=1&limit=2": 400, "kind=REVERSAL": 204, "kind=ADJUSTMENT": 400, "includeReversals=false": 204, "includeReversals=1": 400, "effectiveAtTo=2099-01-01T00:00:00Z": 204, "effectiveAtFrom=2027-01-01T00:00:00Z&effectiveAtTo=2026-01-01T00:00:00Z": 400, "cursor=" + cursor: 204, "cursor=invalid": 400, "provider=x": 400} {
		res, err := app.Test(httptest.NewRequest("GET", "/?"+query, nil))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != want {
			t.Fatalf("%s: %d", query, res.StatusCode)
		}
	}
}

func TestFrozenErrorMappings(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
		code   string
	}{
		{application.ErrInvalidInput, 400, "INVALID_REQUEST"}, {application.ErrInvalidIdempotencyKey, 400, "INVALID_IDEMPOTENCY_KEY"},
		{domain.ErrInvalidTransactionKind, 400, "UNSUPPORTED_TRANSACTION_KIND"}, {domain.ErrInvalidCurrency, 400, "UNSUPPORTED_TRANSACTION_CURRENCY"},
		{domain.ErrInvalidTransactionFields, 400, "INVALID_TRANSACTION_FIELDS"}, {domain.ErrInvalidDecimal, 400, "INVALID_DECIMAL"},
		{application.ErrUnauthenticated, 401, "ACCESS_TOKEN_INVALID"}, {application.ErrPortfolioNotFound, 404, "PORTFOLIO_NOT_FOUND"}, {application.ErrTransactionNotFound, 404, "TRANSACTION_NOT_FOUND"},
		{application.ErrIdempotencyConflict, 409, "IDEMPOTENCY_CONFLICT"}, {domain.ErrTransactionAlreadyCorrected, 409, "TRANSACTION_ALREADY_CORRECTED"},
		{application.ErrAssetNotFound, 422, "ASSET_NOT_FOUND"}, {domain.ErrAssetFinanciallyIneligible, 422, "ASSET_FINANCIALLY_INELIGIBLE"}, {domain.ErrInvalidEffectiveAt, 422, "INVALID_EFFECTIVE_AT"},
		{domain.ErrInsufficientOrderedQuantity, 422, "INSUFFICIENT_ORDERED_ASSET_QUANTITY"}, {application.ErrInvalidBackdatedLedger, 422, "INVALID_BACKDATED_LEDGER"}, {application.ErrPortfolioArchived, 422, "PORTFOLIO_ARCHIVED"}, {domain.ErrTransactionNotCorrectable, 422, "TRANSACTION_NOT_CORRECTABLE"},
		{fmt.Errorf("secret sql error"), 500, "INTERNAL_ERROR"},
	} {
		t.Run(test.code, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(ctx *fiber.Ctx) error { return writeError(ctx, fmt.Errorf("wrapped: %w", test.err), false) })
			res, err := app.Test(httptest.NewRequest("GET", "/", nil))
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			var body struct {
				Error struct{ Code, Message string }
			}
			if json.NewDecoder(res.Body).Decode(&body) != nil || body.Error.Code != test.code || res.StatusCode != test.status || strings.Contains(body.Error.Message, "secret") {
				t.Fatal("wrong public error")
			}
		})
	}
}
