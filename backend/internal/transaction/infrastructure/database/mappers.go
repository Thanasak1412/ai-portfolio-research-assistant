package database

import (
	"time"

	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database/sqlcgen"
	"github.com/jackc/pgx/v5/pgtype"
)

func dbID(id [16]byte) pgtype.UUID { return pgtype.UUID{Bytes: id, Valid: id != [16]byte{}} }
func dbTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}
func dbText(value string, present bool) pgtype.Text {
	return pgtype.Text{String: value, Valid: present}
}
func dbDecimal(value domain.Decimal, present bool) (pgtype.Numeric, error) {
	var result pgtype.Numeric
	if present {
		if err := result.Scan(value.String()); err != nil {
			return result, application.ErrPersistence
		}
	}
	return result, nil
}
func recordParams(fact domain.Transaction) (sqlcgen.InsertTransactionParams, error) {
	p := sqlcgen.InsertTransactionParams{TransactionID: dbID(fact.ID().Bytes()), PortfolioID: dbID(fact.PortfolioID().Bytes()), CreatedByUserID: dbID(fact.CreatedBy().Bytes()), Kind: string(fact.Kind()), Currency: string(fact.Currency()), EffectiveAt: dbTime(fact.EffectiveAt()), CreatedAt: dbTime(fact.CreatedAt()), PortfolioSequence: fact.PortfolioSequence(), Note: dbText(fact.Note()), ExternalReference: dbText(fact.ExternalReference()), ReversalOfTransactionID: dbID(fact.ReversalOf().Bytes()), CorrectionOfTransactionID: dbID(fact.CorrectionOf().Bytes()), OriginatingCorrectionID: dbID(fact.OriginatingCorrection().Bytes())}
	if snapshot, ok := fact.Asset(); ok {
		p.AssetID = dbID(snapshot.ID().Bytes())
		p.AssetTypeSnapshot = dbText(string(snapshot.Type()), true)
		p.AssetExchangeSnapshot = dbText(snapshot.Exchange(), true)
		p.AssetCurrencySnapshot = dbText(string(snapshot.Currency()), true)
	}
	var err error
	p.Quantity, err = dbDecimal(fact.Quantity())
	if err != nil {
		return p, err
	}
	p.UnitPrice, err = dbDecimal(fact.UnitPrice())
	if err != nil {
		return p, err
	}
	p.Fee, err = dbDecimal(fact.Fee())
	if err != nil {
		return p, err
	}
	p.Amount, err = dbDecimal(fact.Amount())
	return p, err
}
func fromDecimal(value pgtype.Numeric) (domain.Decimal, error) {
	if !value.Valid {
		return domain.Decimal{}, nil
	}
	encoded, err := value.Value()
	if err != nil {
		return domain.Decimal{}, application.ErrPersistence
	}
	text, ok := encoded.(string)
	if !ok {
		return domain.Decimal{}, application.ErrPersistence
	}
	decimal, err := domain.ParseNonNegativeDecimal(text)
	if err != nil {
		return domain.Decimal{}, application.ErrPersistence
	}
	return decimal, nil
}
func mapRecord(row sqlcgen.Transaction) (domain.Transaction, error) {
	state := domain.TransactionState{Kind: domain.Kind(row.Kind), Currency: domain.Currency(row.Currency), EffectiveAt: row.EffectiveAt.Time, CreatedAt: row.CreatedAt.Time, PortfolioSequence: row.PortfolioSequence}
	state.ID, _ = domain.NewTransactionID(row.TransactionID.Bytes)
	state.PortfolioID, _ = portfolio.NewPortfolioID(row.PortfolioID.Bytes)
	state.CreatedBy, _ = identity.NewUserID(row.CreatedByUserID.Bytes)
	state.ReversalOf, _ = domain.NewTransactionID(row.ReversalOfTransactionID.Bytes)
	state.CorrectionOf, _ = domain.NewTransactionID(row.CorrectionOfTransactionID.Bytes)
	state.OriginatingCorrection, _ = domain.NewCorrectionID(row.OriginatingCorrectionID.Bytes)
	var err error
	if row.Note.Valid {
		state.Note, err = domain.NewNote(row.Note.String)
		if err != nil {
			return domain.Transaction{}, application.ErrPersistence
		}
	}
	if row.ExternalReference.Valid {
		state.ExternalReference, err = domain.NewExternalReference(row.ExternalReference.String)
		if err != nil {
			return domain.Transaction{}, application.ErrPersistence
		}
	}
	if row.AssetID.Valid {
		id, err := asset.NewAssetID(row.AssetID.Bytes)
		if err != nil {
			return domain.Transaction{}, application.ErrPersistence
		}
		state.Asset, err = domain.NewAssetSnapshot(id, asset.AssetType(row.AssetTypeSnapshot.String), row.AssetExchangeSnapshot.String, asset.Currency(row.AssetCurrencySnapshot.String))
		if err != nil {
			return domain.Transaction{}, application.ErrPersistence
		}
		state.HasAsset = true
	}
	state.Quantity, err = fromDecimal(row.Quantity)
	if err != nil {
		return domain.Transaction{}, err
	}
	state.UnitPrice, err = fromDecimal(row.UnitPrice)
	if err != nil {
		return domain.Transaction{}, err
	}
	state.Fee, err = fromDecimal(row.Fee)
	if err != nil {
		return domain.Transaction{}, err
	}
	state.Amount, err = fromDecimal(row.Amount)
	if err != nil {
		return domain.Transaction{}, err
	}
	result, err := domain.RehydrateTransaction(state)
	if err != nil {
		return domain.Transaction{}, application.ErrPersistence
	}
	return result, nil
}
func mapRecords(rows []sqlcgen.Transaction) ([]domain.Transaction, error) {
	result := make([]domain.Transaction, 0, len(rows))
	for _, row := range rows {
		fact, err := mapRecord(row)
		if err != nil {
			return nil, err
		}
		result = append(result, fact)
	}
	return result, nil
}
func mapCorrection(row sqlcgen.TransactionCorrection) (domain.Correction, error) {
	id, _ := domain.NewCorrectionID(row.CorrectionID.Bytes)
	pid, _ := portfolio.NewPortfolioID(row.PortfolioID.Bytes)
	original, _ := domain.NewTransactionID(row.OriginalTransactionID.Bytes)
	reversal, _ := domain.NewTransactionID(row.ReversalTransactionID.Bytes)
	replacement, _ := domain.NewTransactionID(row.ReplacementTransactionID.Bytes)
	actor, _ := identity.NewUserID(row.CreatedByUserID.Bytes)
	result, err := domain.RehydrateCorrection(id, pid, original, reversal, replacement, actor, row.CreatedAt.Time)
	if err != nil {
		return domain.Correction{}, application.ErrPersistence
	}
	return result, nil
}
