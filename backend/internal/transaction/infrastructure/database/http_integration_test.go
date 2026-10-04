//go:build integration

package database_test

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestTransactionHTTPRealAuthenticationLedgerAndSecurity(t *testing.T) {
	pool := apiPool(t)
	app := apiRuntime(t, pool)
	owner, other := apiAccount(t, app), apiAccount(t, app)
	pid, otherPID := apiPortfolio(t, app, owner), apiPortfolio(t, app, other)
	base := "/api/v1/portfolios/" + pid + "/transactions"
	equity, etf, crypto := apiAsset(t, pool, "EQUITY", "NASDAQ"), apiAsset(t, pool, "ETF", "NYSEARCA"), apiAsset(t, pool, "CRYPTO", "CRYPTO")
	const at = "2026-01-02T00:00:00.123456Z"
	cash := apiCommand("DEPOSIT", "", "100", at)
	for _, route := range []struct{ method, path string }{{"POST", base}, {"GET", base}, {"GET", base + "/" + uuid.NewString()}, {"POST", base + "/" + uuid.NewString() + "/corrections"}} {
		for _, token := range []string{"", "not-a-valid-access-token"} {
			apiRequest(t, app, route.method, route.path, token, "", cash, 401, "ACCESS_TOKEN_INVALID")
		}
	}
	apiRequest(t, app, "GET", "/api/v1/health/live", "", "", "", 200, "")
	apiRequest(t, app, "GET", "/api/v1/health/ready", "", "", "", 200, "")
	apiRequest(t, app, "POST", base, owner, "", cash, 400, "INVALID_IDEMPOTENCY_KEY")
	apiRequest(t, app, "POST", base, owner, "short", cash, 400, "INVALID_IDEMPOTENCY_KEY")
	for name, body := range map[string]string{"unknown": strings.TrimSuffix(cash, "}") + `,"ownerUserId":"attacker"}`, "number": strings.Replace(cash, `"100"`, `100`, 1), "json": cash + "{}", "limit": cash + strings.Repeat(" ", 8192)} {
		code := "INVALID_REQUEST"
		if name == "unknown" {
			code = "INVALID_TRANSACTION_FIELDS"
		}
		apiRequest(t, app, "POST", base, owner, uuid.NewString(), body, 400, code)
	}
	key := uuid.NewString()
	created := apiRequest(t, app, "POST", base, owner, key, cash, 201, "")
	if len(created) != 14 || created.text(t, "amount") != "100" || created.text(t, "portfolioSequence") != "1" || string(created["note"]) != "null" {
		t.Fatal("public response shape differs from contract")
	}
	for _, forbidden := range []string{"ownerUserId", "portfolioId", "assetType", "gross", "net", "fingerprint", "outbox", "balance"} {
		if _, ok := created[forbidden]; ok {
			t.Fatal("internal field exposed")
		}
	}
	id := created.text(t, "id")
	replayed := apiRequest(t, app, "POST", base, owner, key, cash, 201, "")
	if !reflect.DeepEqual(created, replayed) {
		t.Fatal("create replay changed result")
	}
	apiRequest(t, app, "POST", base, owner, key, apiCommand("DEPOSIT", "", "101", at), 409, "IDEMPOTENCY_CONFLICT")
	got := apiRequest(t, app, "GET", base+"/"+id, owner, "", "", 200, "")
	if !reflect.DeepEqual(created, got) {
		t.Fatal("get changed committed fact")
	}
	for _, kind := range []string{"BUY", "SELL", "DIVIDEND", "WITHDRAWAL", "FEE"} {
		assetID := ""
		amount := "1"
		if kind == "BUY" || kind == "SELL" || kind == "DIVIDEND" {
			assetID = equity
		}
		if kind == "BUY" {
			amount = "4"
		}
		result := apiRequest(t, app, "POST", base, owner, uuid.NewString(), apiCommand(kind, assetID, amount, at), 201, "")
		if result.text(t, "kind") != kind {
			t.Fatal("wrong kind")
		}
		if kind == "BUY" || kind == "SELL" {
			if result.text(t, "fee") != "0" || result.text(t, "unitPrice") != "10.123456789012" {
				t.Fatal("trade precision/default fee lost")
			}
		}
	}
	apiRequest(t, app, "POST", base, owner, uuid.NewString(), apiCommand("BUY", etf, "1", at), 201, "")
	for _, test := range []struct{ body, code string }{
		{apiCommand("BUY", crypto, "1", at), "ASSET_FINANCIALLY_INELIGIBLE"},
		{apiCommand("BUY", uuid.NewString(), "1", at), "ASSET_NOT_FOUND"},
		{apiCommand("BUY", "opaque-not-uuid", "1", at), "ASSET_NOT_FOUND"},
		{apiCommand("DEPOSIT", "", "1", "2099-01-01T00:00:00Z"), "INVALID_EFFECTIVE_AT"},
		{apiCommand("SELL", equity, "100", at), "INSUFFICIENT_ORDERED_ASSET_QUANTITY"},
		{apiCommand("SELL", equity, "1", "2026-01-01T00:00:00Z"), "INSUFFICIENT_ORDERED_ASSET_QUANTITY"},
	} {
		apiRequest(t, app, "POST", base, owner, uuid.NewString(), test.body, 422, test.code)
	}
	backdatedBase := "/api/v1/portfolios/" + apiPortfolio(t, app, owner) + "/transactions"
	apiRequest(t, app, "POST", backdatedBase, owner, uuid.NewString(), apiCommand("BUY", equity, "4", "2026-01-01T00:00:00Z"), 201, "")
	apiRequest(t, app, "POST", backdatedBase, owner, uuid.NewString(), apiCommand("SELL", equity, "3", "2026-01-03T00:00:00Z"), 201, "")
	apiRequest(t, app, "POST", backdatedBase, owner, uuid.NewString(), apiCommand("SELL", equity, "2", "2026-01-02T00:00:00Z"), 422, "INVALID_BACKDATED_LEDGER")

	// Identical 404 envelopes for missing, other-owner, other-portfolio and
	// unrepresentable opaque identities (correlation deliberately fixed).
	var absence apiResult
	for _, path := range []string{base + "/" + uuid.NewString(), base + "/opaque", "/api/v1/portfolios/opaque/transactions/" + id, "/api/v1/portfolios/" + otherPID + "/transactions/" + id, "/api/v1/portfolios/" + uuid.NewString() + "/transactions/" + id} {
		result := apiRequest(t, app, "GET", path, owner, "", "", 404, "TRANSACTION_NOT_FOUND")
		if absence != nil && !reflect.DeepEqual(absence, result) {
			t.Fatal("enumeration difference")
		}
		absence = result
	}
	apiRequest(t, app, "GET", base+"/"+id, other, "", "", 404, "TRANSACTION_NOT_FOUND")
	for _, token := range []string{owner, other} {
		path := base
		if token == owner {
			path = "/api/v1/portfolios/" + uuid.NewString() + "/transactions"
		}
		apiRequest(t, app, "GET", path, token, "", "", 404, "PORTFOLIO_NOT_FOUND")
		apiRequest(t, app, "POST", path, token, uuid.NewString(), cash, 404, "PORTFOLIO_NOT_FOUND")
	}
	correctionPath := base + "/" + id + "/corrections"
	replacement := `{"replacement":` + apiCommand("DEPOSIT", "", "125", at) + `}`
	apiRequest(t, app, "POST", correctionPath, owner, "", replacement, 400, "INVALID_IDEMPOTENCY_KEY")
	apiRequest(t, app, "POST", correctionPath, owner, "invalid", replacement, 400, "INVALID_IDEMPOTENCY_KEY")
	apiRequest(t, app, "POST", correctionPath, other, uuid.NewString(), replacement, 404, "TRANSACTION_NOT_FOUND")
	apiRequest(t, app, "POST", base+"/opaque/corrections", owner, uuid.NewString(), replacement, 404, "TRANSACTION_NOT_FOUND")
	apiRequest(t, app, "POST", correctionPath, owner, uuid.NewString(), `{"replacement":null}`, 400, "INVALID_REQUEST")
	correctionKey := uuid.NewString()
	corrected := apiRequest(t, app, "POST", correctionPath, owner, correctionKey, replacement, 201, "")
	original, reversal, newFact := corrected.child(t, "original"), corrected.child(t, "reversal"), corrected.child(t, "replacement")
	if len(corrected) != 3 || original.text(t, "id") != id || reversal.text(t, "kind") != "REVERSAL" || newFact.text(t, "amount") != "125" {
		t.Fatal("invalid correction result")
	}
	if original.child(t, "correctionLinks").text(t, "reversalTransactionId") != reversal.text(t, "id") || original.child(t, "correctionLinks").text(t, "replacementTransactionId") != newFact.text(t, "id") || reversal.child(t, "correctionLinks").text(t, "reversesTransactionId") != id || newFact.child(t, "correctionLinks").text(t, "replacesTransactionId") != id {
		t.Fatal("correction links lost")
	}
	apiRequest(t, app, "POST", correctionPath, owner, uuid.NewString(), replacement, 409, "TRANSACTION_ALREADY_CORRECTED")
	apiRequest(t, app, "POST", correctionPath, owner, correctionKey, `{"replacement":`+cash+`}`, 409, "IDEMPOTENCY_CONFLICT")
	apiRequest(t, app, "POST", base+"/"+reversal.text(t, "id")+"/corrections", owner, uuid.NewString(), replacement, 422, "TRANSACTION_NOT_CORRECTABLE")
	second := apiRequest(t, app, "POST", base+"/"+newFact.text(t, "id")+"/corrections", owner, uuid.NewString(), `{"replacement":`+cash+`}`, 201, "")
	chain := second.child(t, "original").child(t, "correctionLinks")
	if chain.text(t, "replacesTransactionId") != id || string(chain["replacementTransactionId"]) == "null" {
		t.Fatal("chain cannot represent both roles")
	}
	if replay := apiRequest(t, app, "POST", correctionPath, owner, correctionKey, replacement, 201, ""); !reflect.DeepEqual(corrected, replay) {
		t.Fatal("correction replay changed after later correction")
	}
	if replay := apiRequest(t, app, "POST", base, owner, key, cash, 201, ""); !reflect.DeepEqual(created, replay) {
		t.Fatal("create replay changed after correction")
	}

	// Stable tied-time keyset pages: each committed immutable fact exactly once.
	seen := map[string]bool{}
	path := base + "?limit=2"
	for page := 0; page < 20; page++ {
		result := apiRequest(t, app, "GET", path, owner, "", "", 200, "")
		var items []apiResult
		apiMust(t, json.Unmarshal(result["items"], &items))
		for _, item := range items {
			itemID := item.text(t, "id")
			if seen[itemID] {
				t.Fatal("pagination duplicate")
			}
			seen[itemID] = true
		}
		if string(result["nextCursor"]) == "null" {
			break
		}
		path = base + "?limit=2&cursor=" + result.text(t, "nextCursor")
	}
	if len(seen) != 11 {
		t.Fatalf("history count=%d want=11", len(seen))
	}
	for query, want := range map[string]int{"kind=BUY": 2, "includeReversals=false": 9, "effectiveAtFrom=2099-01-01T00:00:00Z": 0, "effectiveAtTo=2026-01-01T00:00:00Z": 0} {
		result := apiRequest(t, app, "GET", base+"?"+query, owner, "", "", 200, "")
		var items []apiResult
		apiMust(t, json.Unmarshal(result["items"], &items))
		if len(items) != want {
			t.Fatalf("filter %s count=%d", query, len(items))
		}
	}
	apiRequest(t, app, "POST", "/api/v1/portfolios/"+pid+"/archive", owner, "", "", 200, "")
	apiRequest(t, app, "POST", base, owner, uuid.NewString(), cash, 422, "PORTFOLIO_ARCHIVED")
	apiRequest(t, app, "POST", base+"/"+second.child(t, "replacement").text(t, "id")+"/corrections", owner, uuid.NewString(), replacement, 422, "PORTFOLIO_ARCHIVED")
	apiRequest(t, app, "GET", base, owner, "", "", 200, "")
	for _, method := range []string{"PATCH", "DELETE"} {
		apiRequest(t, app, method, base+"/"+id, owner, "", cash, 405, "HTTP_ERROR")
	}

	var count int
	apiMust(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM platform_outbox_events WHERE correlation_id = 'm3-http-integration'`).Scan(&count))
	if count == 0 {
		t.Fatal("HTTP correlation was not propagated to durable outbox evidence")
	}
	apiMust(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM platform_outbox_events WHERE publication_state <> 'PENDING' OR attempt_count <> 0`).Scan(&count))
	if count != 0 {
		t.Fatal("HTTP activation changed publication state")
	}
}
