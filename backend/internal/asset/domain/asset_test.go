package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAssetIDAndClosedMetadata(t *testing.T) {
	if _, err := NewAssetID(uuid.Nil); err == nil {
		t.Fatal("nil Asset ID accepted")
	}
	if _, err := ParseAssetType("BOND"); err == nil {
		t.Fatal("unsupported Asset type accepted")
	}
	for _, value := range []string{"EQUITY", "ETF", "CRYPTO"} {
		if _, err := ParseAssetType(value); err != nil {
			t.Fatalf("type %q: %v", value, err)
		}
	}
	if _, err := ParseCurrency("THB"); err == nil {
		t.Fatal("non-USD accepted")
	}
}

func TestAssetRehydrationValidatesPersistenceBoundaries(t *testing.T) {
	now := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	id, _ := NewAssetID(uuid.New())
	for _, fixture := range []struct {
		assetType                    AssetType
		exchange, normalizedExchange string
	}{
		{AssetTypeEquity, "NASDAQ", "NASDAQ"}, {AssetTypeETF, "NYSEARCA", "NYSEARCA"}, {AssetTypeCrypto, "CRYPTO", "CRYPTO"},
	} {
		if _, err := RehydrateAsset(id, "ABC", "Asset name", fixture.assetType, fixture.exchange, CurrencyUSD, "ABC", fixture.normalizedExchange, now, now); err != nil {
			t.Fatalf("valid %s Asset: %v", fixture.assetType, err)
		}
	}
	for _, invalid := range []struct {
		symbol, name                 string
		assetType                    AssetType
		exchange, normalizedExchange string
		currency                     Currency
	}{
		{" ABC", "Name", AssetTypeEquity, "NASDAQ", "NASDAQ", CurrencyUSD},
		{"ABC", " \t", AssetTypeEquity, "NASDAQ", "NASDAQ", CurrencyUSD},
		{"ABC", "Name", AssetTypeCrypto, "COINBASE", "COINBASE", CurrencyUSD},
		{"ABC", "Name", AssetTypeEquity, "NASDAQ", "NASDAQ", Currency("THB")},
		{strings.Repeat("A", 65), "Name", AssetTypeEquity, "NASDAQ", "NASDAQ", CurrencyUSD},
	} {
		if _, err := RehydrateAsset(id, invalid.symbol, invalid.name, invalid.assetType, invalid.exchange, invalid.currency, "ABC", invalid.normalizedExchange, now, now); err == nil {
			t.Fatalf("invalid Asset accepted: %#v", invalid)
		}
	}
}
