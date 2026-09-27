package domain

import (
	"time"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	portfoliodomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

// TransactionState is a value-only rehydration input. The resulting record has
// private state and no mutating operations.
type TransactionState struct {
	ID                    TransactionID
	PortfolioID           portfoliodomain.PortfolioID
	CreatedBy             identitydomain.UserID
	Kind                  Kind
	Asset                 AssetSnapshot
	HasAsset              bool
	Quantity              Decimal
	UnitPrice             Decimal
	Fee                   Decimal
	Amount                Decimal
	Currency              Currency
	EffectiveAt           time.Time
	PortfolioSequence     int64
	Note                  OptionalText
	ExternalReference     OptionalText
	ReversalOf            TransactionID
	CorrectionOf          TransactionID
	OriginatingCorrection CorrectionID
	CreatedAt             time.Time
}

type Transaction struct{ state TransactionState }

func RehydrateTransaction(state TransactionState) (Transaction, error) {
	if state.ID.IsZero() || state.PortfolioID.IsZero() || state.CreatedBy.IsZero() || state.PortfolioSequence <= 0 || state.CreatedAt.IsZero() || state.CreatedAt.Nanosecond()%1000 != 0 || state.EffectiveAt.IsZero() || state.EffectiveAt.Nanosecond()%1000 != 0 || state.EffectiveAt.After(state.CreatedAt) || state.Currency != CurrencyUSD || validateOptional(state.Note, state.ExternalReference) != nil {
		return Transaction{}, ErrInvalidTransaction
	}
	if _, err := ParseKind(string(state.Kind)); err != nil {
		return Transaction{}, ErrInvalidTransaction
	}
	if state.HasAsset && !state.Asset.IsValid() {
		return Transaction{}, ErrInvalidTransaction
	}
	if !state.HasAsset && state.Asset.IsValid() {
		return Transaction{}, ErrInvalidTransaction
	}
	if !state.ReversalOf.IsZero() && state.ReversalOf == state.ID || !state.CorrectionOf.IsZero() && state.CorrectionOf == state.ID {
		return Transaction{}, ErrInvalidTransaction
	}
	if state.Kind == KindReversal {
		if state.ReversalOf.IsZero() || !state.CorrectionOf.IsZero() || state.OriginatingCorrection.IsZero() || state.Note.present || state.ExternalReference.present {
			return Transaction{}, ErrInvalidTransaction
		}
	} else {
		if !state.ReversalOf.IsZero() || (state.CorrectionOf.IsZero() != state.OriginatingCorrection.IsZero()) {
			return Transaction{}, ErrInvalidTransaction
		}
	}
	trade := state.HasAsset && state.Quantity.IsPositive() && state.UnitPrice.IsPositive() && state.Fee.IsNonNegative() && !state.Amount.IsValid()
	assetCash := state.HasAsset && state.Amount.IsPositive() && !state.Quantity.IsValid() && !state.UnitPrice.IsValid() && !state.Fee.IsValid()
	portfolioCash := !state.HasAsset && state.Amount.IsPositive() && !state.Quantity.IsValid() && !state.UnitPrice.IsValid() && !state.Fee.IsValid()
	switch state.Kind {
	case KindBuy, KindSell:
		if !trade {
			return Transaction{}, ErrInvalidTransaction
		}
	case KindDividend:
		if !assetCash {
			return Transaction{}, ErrInvalidTransaction
		}
	case KindDeposit, KindWithdrawal, KindFee:
		if !portfolioCash {
			return Transaction{}, ErrInvalidTransaction
		}
	case KindReversal:
		if !trade && !assetCash && !portfolioCash {
			return Transaction{}, ErrInvalidTransaction
		}
	}
	state.EffectiveAt = state.EffectiveAt.UTC()
	state.CreatedAt = state.CreatedAt.UTC()
	return Transaction{state}, nil
}

func NewTransactionFromCommand(id TransactionID, portfolioID portfoliodomain.PortfolioID, actor identitydomain.UserID, sequence int64, createdAt time.Time, command Command) (Transaction, error) {
	if !command.IsValid() {
		return Transaction{}, ErrInvalidTransactionFields
	}
	return RehydrateTransaction(TransactionState{ID: id, PortfolioID: portfolioID, CreatedBy: actor, Kind: command.kind, Asset: command.asset, HasAsset: command.hasAsset, Quantity: command.quantity, UnitPrice: command.unitPrice, Fee: command.fee, Amount: command.amount, Currency: command.currency, EffectiveAt: command.effectiveAt, PortfolioSequence: sequence, Note: command.note, ExternalReference: command.externalReference, CreatedAt: createdAt})
}

func (transaction Transaction) ID() TransactionID { return transaction.state.ID }
func (transaction Transaction) PortfolioID() portfoliodomain.PortfolioID {
	return transaction.state.PortfolioID
}
func (transaction Transaction) CreatedBy() identitydomain.UserID { return transaction.state.CreatedBy }
func (transaction Transaction) Kind() Kind                       { return transaction.state.Kind }
func (transaction Transaction) Asset() (AssetSnapshot, bool) {
	return transaction.state.Asset, transaction.state.HasAsset
}
func (transaction Transaction) Quantity() (Decimal, bool) {
	return transaction.state.Quantity, transaction.state.Quantity.IsValid()
}
func (transaction Transaction) UnitPrice() (Decimal, bool) {
	return transaction.state.UnitPrice, transaction.state.UnitPrice.IsValid()
}
func (transaction Transaction) Fee() (Decimal, bool) {
	return transaction.state.Fee, transaction.state.Fee.IsValid()
}
func (transaction Transaction) Amount() (Decimal, bool) {
	return transaction.state.Amount, transaction.state.Amount.IsValid()
}
func (transaction Transaction) Currency() Currency       { return transaction.state.Currency }
func (transaction Transaction) EffectiveAt() time.Time   { return transaction.state.EffectiveAt }
func (transaction Transaction) PortfolioSequence() int64 { return transaction.state.PortfolioSequence }
func (transaction Transaction) Note() (string, bool)     { return transaction.state.Note.Value() }
func (transaction Transaction) ExternalReference() (string, bool) {
	return transaction.state.ExternalReference.Value()
}
func (transaction Transaction) ReversalOf() TransactionID   { return transaction.state.ReversalOf }
func (transaction Transaction) CorrectionOf() TransactionID { return transaction.state.CorrectionOf }
func (transaction Transaction) OriginatingCorrection() CorrectionID {
	return transaction.state.OriginatingCorrection
}
func (transaction Transaction) CreatedAt() time.Time { return transaction.state.CreatedAt }
