package database

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/infrastructure/database/sqlcgen"
)

func mapAsset(row sqlcgen.Asset) (domain.Asset, error) {
	if !row.AssetID.Valid || !row.NormalizedSymbol.Valid || !row.NormalizedExchange.Valid || !row.CreatedAt.Valid || !row.UpdatedAt.Valid {
		return domain.Asset{}, domain.ErrInvalidAsset
	}
	id, err := domain.NewAssetID(uuid.UUID(row.AssetID.Bytes))
	if err != nil {
		return domain.Asset{}, err
	}
	typeValue, err := domain.ParseAssetType(row.AssetType)
	if err != nil {
		return domain.Asset{}, err
	}
	currency, err := domain.ParseCurrency(row.Currency)
	if err != nil {
		return domain.Asset{}, err
	}
	return domain.RehydrateAsset(id, row.Symbol, row.Name, typeValue, row.Exchange, currency, row.NormalizedSymbol.String, row.NormalizedExchange.String, row.CreatedAt.Time, row.UpdatedAt.Time)
}

func pgAssetID(id domain.AssetID) pgtype.UUID {
	return pgtype.UUID{Bytes: id.Bytes(), Valid: !id.IsZero()}
}
func pgSearchText(value string, set bool) pgtype.Text { return pgtype.Text{String: value, Valid: set} }
