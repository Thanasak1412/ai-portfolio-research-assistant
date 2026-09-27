package domain

import (
	"time"
	"unicode/utf8"

	assetdomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
)

type Currency string

const CurrencyUSD Currency = "USD"

func ParseCurrency(value string) (Currency, error) {
	if value != string(CurrencyUSD) {
		return "", ErrInvalidCurrency
	}
	return CurrencyUSD, nil
}

// OptionalText distinguishes absent from a supplied empty string.
type OptionalText struct {
	value   string
	present bool
}

func NewNote(value string) (OptionalText, error)              { return newOptionalText(value, 2000) }
func NewExternalReference(value string) (OptionalText, error) { return newOptionalText(value, 256) }
func newOptionalText(value string, max int) (OptionalText, error) {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return OptionalText{}, ErrInvalidTransactionFields
	}
	return OptionalText{value, true}, nil
}
func (text OptionalText) Value() (string, bool) { return text.value, text.present }

// ValidateEffectiveAt accepts only representable, already-effective instants.
func ValidateEffectiveAt(value, now time.Time) (time.Time, error) {
	if value.IsZero() || now.IsZero() || value.Nanosecond()%1000 != 0 || value.After(now) {
		return time.Time{}, ErrInvalidEffectiveAt
	}
	return value.UTC(), nil
}

type AssetSnapshot struct {
	id        assetdomain.AssetID
	assetType assetdomain.AssetType
	exchange  string
	currency  assetdomain.Currency
}

func NewAssetSnapshot(id assetdomain.AssetID, assetType assetdomain.AssetType, exchange string, currency assetdomain.Currency) (AssetSnapshot, error) {
	if id.IsZero() || (assetType != assetdomain.AssetTypeEquity && assetType != assetdomain.AssetTypeETF) || currency != assetdomain.CurrencyUSD {
		return AssetSnapshot{}, ErrAssetFinanciallyIneligible
	}
	switch exchange {
	case "NYSE", "NASDAQ", "NYSEARCA", "AMEX":
	default:
		return AssetSnapshot{}, ErrAssetFinanciallyIneligible
	}
	return AssetSnapshot{id, assetType, exchange, currency}, nil
}
func (asset AssetSnapshot) IsValid() bool {
	_, err := NewAssetSnapshot(asset.id, asset.assetType, asset.exchange, asset.currency)
	return err == nil
}
func (asset AssetSnapshot) ID() assetdomain.AssetID        { return asset.id }
func (asset AssetSnapshot) Type() assetdomain.AssetType    { return asset.assetType }
func (asset AssetSnapshot) Exchange() string               { return asset.exchange }
func (asset AssetSnapshot) Currency() assetdomain.Currency { return asset.currency }
