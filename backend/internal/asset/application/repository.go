package application

import (
	"context"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

type LookupReader interface {
	GetAsset(context.Context, identity.Principal, domain.AssetID) (domain.Asset, error)
}

type CursorPosition struct {
	Symbol   string
	Exchange string
	AssetID  domain.AssetID
}

type SearchQuery struct {
	Search    string
	HasSearch bool
	AssetType *domain.AssetType
	After     *CursorPosition
	Limit     int
}

// Repository exposes only normal-user read operations for the global catalog.
type Repository interface {
	GetByID(context.Context, domain.AssetID) (domain.Asset, error)
	Search(context.Context, SearchQuery) ([]domain.Asset, error)
}
