// Package application orchestrates authenticated, read-only Asset operations.
package application

import (
	"context"
	"unicode/utf8"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

const (
	DefaultSearchLimit = 25
	MaximumSearchLimit = 100
)

type Service struct{ repository Repository }

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, ErrAssetService
	}
	return &Service{repository: repository}, nil
}

type SearchInput struct {
	Search    string
	HasSearch bool
	AssetType *domain.AssetType
	Cursor    *CursorPosition
	Limit     int
}

type SearchResult struct {
	Assets []domain.Asset
	Next   *CursorPosition
}

func (service *Service) GetAsset(ctx context.Context, principal identitydomain.Principal, id domain.AssetID) (domain.Asset, error) {
	if !principal.IsAuthenticated() {
		return domain.Asset{}, ErrUnauthenticated
	}
	if id.IsZero() {
		return domain.Asset{}, ErrInvalidAssetInput
	}
	return service.repository.GetByID(ctx, id)
}

func (service *Service) SearchAssets(ctx context.Context, principal identitydomain.Principal, input SearchInput) (SearchResult, error) {
	if !principal.IsAuthenticated() {
		return SearchResult{}, ErrUnauthenticated
	}
	if input.Limit < 1 || input.Limit > MaximumSearchLimit || (input.HasSearch && (input.Search == "" || !utf8.ValidString(input.Search) || utf8.RuneCountInString(input.Search) > 100)) {
		return SearchResult{}, ErrInvalidAssetInput
	}
	if input.AssetType != nil {
		if _, err := domain.ParseAssetType(string(*input.AssetType)); err != nil {
			return SearchResult{}, ErrInvalidAssetInput
		}
	}
	if input.Cursor != nil && (input.Cursor.AssetID.IsZero() || input.Cursor.Symbol == "" || input.Cursor.Exchange == "") {
		return SearchResult{}, ErrInvalidAssetInput
	}
	assets, err := service.repository.Search(ctx, SearchQuery{Search: input.Search, HasSearch: input.HasSearch, AssetType: input.AssetType, After: input.Cursor, Limit: input.Limit + 1})
	if err != nil {
		return SearchResult{}, err
	}
	result := SearchResult{Assets: assets}
	if len(result.Assets) > input.Limit {
		result.Assets = result.Assets[:input.Limit]
		last := result.Assets[len(result.Assets)-1]
		result.Next = &CursorPosition{Symbol: last.Symbol(), Exchange: last.Exchange(), AssetID: last.ID()}
	}
	return result, nil
}
