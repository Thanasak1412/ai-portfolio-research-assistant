package database

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
)

func TestLedgerQuerySurfaceIsImmutableAndPrivate(t *testing.T) {
	files, err := filepath.Glob("../../../../queries/transaction/*.sql")
	if err != nil || len(files) != 4 {
		t.Fatalf("query inventory: %v %v", files, err)
	}
	forbidden := regexp.MustCompile(`(?i)\b(UPDATE|DELETE\s+FROM)\s+(transactions|transaction_corrections)\b`)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if forbidden.Match(data) {
			t.Fatalf("mutable ledger query in %s", file)
		}
	}
	for _, row := range []any{sqlcgen.Transaction{}, sqlcgen.TransactionCorrection{}, sqlcgen.TransactionIdempotency{}} {
		typ := reflect.TypeOf(row)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.Tag.Get("json") != "" {
				t.Fatalf("public serialization tag on %s", field.Name)
			}
			if field.Type.Kind() == reflect.Float32 || field.Type.Kind() == reflect.Float64 {
				t.Fatal("binary financial float")
			}
		}
	}
	data, err := os.ReadFile("../../../../migrations/00005_m3_transaction_ledger.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"CREATE TRIGGER", "owner_user_id", "request_body", "response_body", "gross", "net_amount", "holding", "valuation"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("forbidden persistence surface %s", forbidden)
		}
	}
}
