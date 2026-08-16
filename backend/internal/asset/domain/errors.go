package domain

import "errors"

var (
	ErrInvalidAssetID       = errors.New("invalid asset identifier")
	ErrInvalidAssetType     = errors.New("invalid asset type")
	ErrInvalidAssetSymbol   = errors.New("invalid asset symbol")
	ErrInvalidAssetName     = errors.New("invalid asset name")
	ErrInvalidAssetExchange = errors.New("invalid asset exchange")
	ErrInvalidAssetCurrency = errors.New("invalid asset currency")
	ErrInvalidAsset         = errors.New("invalid asset")
)
