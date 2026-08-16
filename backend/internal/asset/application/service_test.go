package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

func TestAssetServiceDefaultsDenyAndUsesLimitPlusOne(t *testing.T) {
	repository := &fakeRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SearchAssets(context.Background(), identitydomain.Principal{}, SearchInput{Limit: 1}); !errors.Is(err, ErrUnauthenticated) || repository.searches != 0 {
		t.Fatalf("unauthenticated search=%v calls=%d", err, repository.searches)
	}
	principal := assetTestPrincipal(t)
	assets := []domain.Asset{assetFixture(t, "AAA", "NYSE"), assetFixture(t, "BBB", "NYSE")}
	repository.assets = assets
	result, err := service.SearchAssets(context.Background(), principal, SearchInput{Limit: 1})
	if err != nil || len(result.Assets) != 1 || result.Next == nil || repository.query.Limit != 2 {
		t.Fatalf("result=%+v query=%+v err=%v", result, repository.query, err)
	}
	if _, err := service.GetAsset(context.Background(), identitydomain.Principal{}, assets[0].ID()); !errors.Is(err, ErrUnauthenticated) || repository.gets != 0 {
		t.Fatalf("unauthenticated get=%v calls=%d", err, repository.gets)
	}
	if _, err := service.GetAsset(context.Background(), principal, assets[0].ID()); err != nil {
		t.Fatal(err)
	}
}

type fakeRepository struct {
	assets         []domain.Asset
	query          SearchQuery
	searches, gets int
	getErr         error
}

func (repository *fakeRepository) GetByID(_ context.Context, id domain.AssetID) (domain.Asset, error) {
	repository.gets++
	if repository.getErr != nil {
		return domain.Asset{}, repository.getErr
	}
	for _, asset := range repository.assets {
		if asset.ID() == id {
			return asset, nil
		}
	}
	return domain.Asset{}, ErrAssetNotFound
}
func (repository *fakeRepository) Search(_ context.Context, query SearchQuery) ([]domain.Asset, error) {
	repository.searches++
	repository.query = query
	return repository.assets, nil
}

func assetTestPrincipal(t *testing.T) identitydomain.Principal {
	t.Helper()
	id, _ := identitydomain.NewUserID(uuid.New())
	principal, err := identitydomain.NewPrincipal(id)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}
func assetFixture(t *testing.T, symbol, exchange string) domain.Asset {
	t.Helper()
	id, _ := domain.NewAssetID(uuid.New())
	asset, err := domain.RehydrateAsset(id, symbol, "Asset "+symbol, domain.AssetTypeEquity, exchange, domain.CurrencyUSD, symbol, exchange, time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return asset
}
