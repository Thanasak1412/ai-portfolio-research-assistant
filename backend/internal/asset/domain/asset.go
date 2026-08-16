// Package domain contains pure canonical Asset reference metadata.
package domain

import (
	"time"
	"unicode/utf8"
)

type AssetType string

const (
	AssetTypeEquity AssetType = "EQUITY"
	AssetTypeETF    AssetType = "ETF"
	AssetTypeCrypto AssetType = "CRYPTO"
)

func ParseAssetType(value string) (AssetType, error) {
	typeValue := AssetType(value)
	if typeValue != AssetTypeEquity && typeValue != AssetTypeETF && typeValue != AssetTypeCrypto {
		return "", ErrInvalidAssetType
	}
	return typeValue, nil
}

type Currency string

const CurrencyUSD Currency = "USD"

func ParseCurrency(value string) (Currency, error) {
	if Currency(value) != CurrencyUSD {
		return "", ErrInvalidAssetCurrency
	}
	return CurrencyUSD, nil
}

type Asset struct {
	id                 AssetID
	symbol             string
	name               string
	assetType          AssetType
	exchange           string
	currency           Currency
	normalizedSymbol   string
	normalizedExchange string
	createdAt          time.Time
	updatedAt          time.Time
}

// RehydrateAsset validates a database row before it crosses the infrastructure
// boundary. Normalized values are retained solely as persisted identity state.
func RehydrateAsset(id AssetID, symbol, name string, assetType AssetType, exchange string, currency Currency, normalizedSymbol, normalizedExchange string, createdAt, updatedAt time.Time) (Asset, error) {
	if id.IsZero() || !validTrimmedDisplay(symbol, 64) || !validDisplayName(name) || !validTrimmedDisplay(exchange, 64) || normalizedSymbol == "" || normalizedExchange == "" || createdAt.IsZero() || updatedAt.IsZero() || updatedAt.Before(createdAt) {
		return Asset{}, ErrInvalidAsset
	}
	if _, err := ParseAssetType(string(assetType)); err != nil {
		return Asset{}, err
	}
	if _, err := ParseCurrency(string(currency)); err != nil {
		return Asset{}, err
	}
	if assetType == AssetTypeCrypto && (exchange != "CRYPTO" || normalizedExchange != "CRYPTO") {
		return Asset{}, ErrInvalidAsset
	}
	return Asset{id: id, symbol: symbol, name: name, assetType: assetType, exchange: exchange, currency: currency, normalizedSymbol: normalizedSymbol, normalizedExchange: normalizedExchange, createdAt: createdAt, updatedAt: updatedAt}, nil
}

func validTrimmedDisplay(value string, maximum int) bool {
	return utf8.ValidString(value) && value != "" && value == trimASCIIEdges(value) && utf8.RuneCountInString(value) <= maximum
}

func validDisplayName(value string) bool {
	return utf8.ValidString(value) && trimASCIIEdges(value) != "" && utf8.RuneCountInString(value) <= 256
}

func trimASCIIEdges(value string) string {
	start, end := 0, len(value)
	for start < end && isASCIIWhitespace(value[start]) {
		start++
	}
	for end > start && isASCIIWhitespace(value[end-1]) {
		end--
	}
	return value[start:end]
}

func isASCIIWhitespace(value byte) bool {
	switch value {
	case 0x20, 0x09, 0x0A, 0x0D, 0x0C, 0x0B:
		return true
	default:
		return false
	}
}

func (asset Asset) ID() AssetID          { return asset.id }
func (asset Asset) Symbol() string       { return asset.symbol }
func (asset Asset) Name() string         { return asset.name }
func (asset Asset) AssetType() AssetType { return asset.assetType }
func (asset Asset) Exchange() string     { return asset.exchange }
func (asset Asset) Currency() Currency   { return asset.currency }
func (asset Asset) CreatedAt() time.Time { return asset.createdAt }
func (asset Asset) UpdatedAt() time.Time { return asset.updatedAt }
