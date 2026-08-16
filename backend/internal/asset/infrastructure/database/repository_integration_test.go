//go:build integration

package database

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/infrastructure/database/sqlcgen"
)

func TestPostgresAssetRepositoryReadsCanonicalSearchPages(t *testing.T) {
	pool := openAssetTestPool(t)
	queries := sqlcgen.New(pool)
	repository := NewPostgresAssetRepository(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	suffix := strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	first := bootstrapAsset(t, queries, "AA"+suffix, "Atlas "+suffix, "EQUITY", "NYSE", now)
	second := bootstrapAsset(t, queries, "BB"+suffix, "Beacon "+suffix, "ETF", "NASDAQ", now)
	third := bootstrapAsset(t, queries, "CC"+suffix, "Cipher "+suffix, "CRYPTO", "CRYPTO", now)

	firstID, _ := domain.NewAssetID(uuid.UUID(first.AssetID.Bytes))
	got, err := repository.GetByID(context.Background(), firstID)
	if err != nil || got.Symbol() != first.Symbol || got.Currency() != domain.CurrencyUSD {
		t.Fatalf("get=%+v err=%v", got, err)
	}
	search := strings.ToLower("atlas " + suffix)
	assets, err := repository.Search(context.Background(), application.SearchQuery{Search: search, HasSearch: true, Limit: 10})
	if err != nil || len(assets) != 1 || assets[0].ID() != firstID {
		t.Fatalf("name search=%+v err=%v", assets, err)
	}
	filter := domain.AssetTypeETF
	assets, err = repository.Search(context.Background(), application.SearchQuery{Search: suffix, HasSearch: true, AssetType: &filter, Limit: 10})
	if err != nil || len(assets) != 1 || assets[0].Symbol() != second.Symbol {
		t.Fatalf("type search=%+v err=%v", assets, err)
	}

	assets, err = repository.Search(context.Background(), application.SearchQuery{Search: suffix, HasSearch: true, Limit: 2})
	if err != nil || len(assets) != 2 || assets[0].Symbol() != first.Symbol || assets[1].Symbol() != second.Symbol {
		t.Fatalf("first page=%+v err=%v", assets, err)
	}
	continuation := application.CursorPosition{Symbol: assets[1].Symbol(), Exchange: assets[1].Exchange(), AssetID: assets[1].ID()}
	assets, err = repository.Search(context.Background(), application.SearchQuery{Search: suffix, HasSearch: true, After: &continuation, Limit: 2})
	if err != nil || len(assets) != 1 || assets[0].Symbol() != third.Symbol || assets[0].AssetType() != domain.AssetTypeCrypto || assets[0].Exchange() != "CRYPTO" {
		t.Fatalf("continuation=%+v err=%v", assets, err)
	}
}
