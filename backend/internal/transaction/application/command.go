package application

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	assetapp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]{15,127}$`)
var correlationPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func normalize(input CommandInput, now time.Time) (CommandInput, error) {
	if !input.Kind.IsPublicCreatable() {
		return input, domain.ErrInvalidTransactionKind
	}
	if input.Currency != domain.CurrencyUSD {
		return input, domain.ErrInvalidCurrency
	}
	effective, err := domain.ValidateEffectiveAt(input.EffectiveAt, now)
	if err != nil {
		return input, err
	}
	input.EffectiveAt = effective
	if value, ok := input.Note.Value(); ok {
		if _, err := domain.NewNote(value); err != nil {
			return input, err
		}
	}
	if value, ok := input.ExternalReference.Value(); ok {
		if _, err := domain.NewExternalReference(value); err != nil {
			return input, err
		}
	}
	switch input.Kind {
	case domain.KindBuy, domain.KindSell:
		if !input.Fee.IsValid() {
			input.Fee, _ = domain.ParseNonNegativeDecimal("0")
		}
		if input.AssetID.IsZero() || !input.Quantity.IsPositive() || !input.UnitPrice.IsPositive() || !input.Fee.IsNonNegative() || input.Amount.IsValid() {
			return input, domain.ErrInvalidTransactionFields
		}
	case domain.KindDividend:
		if input.AssetID.IsZero() || !input.Amount.IsPositive() || input.Quantity.IsValid() || input.UnitPrice.IsValid() || input.Fee.IsValid() {
			return input, domain.ErrInvalidTransactionFields
		}
	default:
		if !input.AssetID.IsZero() || !input.Amount.IsPositive() || input.Quantity.IsValid() || input.UnitPrice.IsValid() || input.Fee.IsValid() {
			return input, domain.ErrInvalidTransactionFields
		}
	}
	return input, nil
}

// Fixed field order, explicit absent/empty text, canonical decimals and UTC
// microseconds form a versioned semantic fingerprint. Asset metadata, caller
// correlation, server IDs, and current time are deliberately excluded.
func fingerprint(scope string, target domain.TransactionID, input CommandInput) [32]byte {
	optional := func(value string, present bool) *string {
		if !present {
			return nil
		}
		return &value
	}
	encoded, _ := json.Marshal(struct {
		Version                                                                             int
		Scope, Target, Kind, Asset, Quantity, UnitPrice, Fee, Amount, Currency, EffectiveAt string
		Note, ExternalReference                                                             *string
	}{
		Version: 1, Scope: scope, Target: target.String(), Kind: string(input.Kind),
		Asset: input.AssetID.String(), Quantity: input.Quantity.String(),
		UnitPrice: input.UnitPrice.String(), Fee: input.Fee.String(), Amount: input.Amount.String(),
		Currency: string(input.Currency), EffectiveAt: input.EffectiveAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
		Note: optional(input.Note.Value()), ExternalReference: optional(input.ExternalReference.Value()),
	})
	return sha256.Sum256(encoded)
}

func (s *Service) resolveCommand(ctx context.Context, assets AssetReader, principal identity.Principal, input CommandInput, now time.Time) (domain.Command, error) {
	var snapshot domain.AssetSnapshot
	if !input.AssetID.IsZero() {
		item, err := assets.GetAsset(ctx, principal, input.AssetID)
		if errors.Is(err, assetapp.ErrAssetNotFound) {
			return domain.Command{}, ErrAssetNotFound
		}
		if err != nil {
			return domain.Command{}, err
		}
		if item.ID() != input.AssetID {
			return domain.Command{}, ErrAssetNotFound
		}
		snapshot, err = domain.NewAssetSnapshot(item.ID(), item.AssetType(), item.Exchange(), item.Currency())
		if err != nil {
			return domain.Command{}, err
		}
	}
	c := domain.CommandContext{Currency: input.Currency, EffectiveAt: input.EffectiveAt, Now: now, Note: input.Note, ExternalReference: input.ExternalReference}
	switch input.Kind {
	case domain.KindBuy:
		return domain.NewBuyCommand(snapshot, input.Quantity, input.UnitPrice, input.Fee, c)
	case domain.KindSell:
		return domain.NewSellCommand(snapshot, input.Quantity, input.UnitPrice, input.Fee, c)
	case domain.KindDividend:
		return domain.NewDividendCommand(snapshot, input.Amount, c)
	case domain.KindDeposit:
		return domain.NewDepositCommand(input.Amount, c)
	case domain.KindWithdrawal:
		return domain.NewWithdrawalCommand(input.Amount, c)
	case domain.KindFee:
		return domain.NewFeeCommand(input.Amount, c)
	default:
		return domain.Command{}, domain.ErrInvalidTransactionKind
	}
}
