package domain

import "errors"

var (
	ErrInvalidTransactionID        = errors.New("invalid transaction ID")
	ErrInvalidCorrectionID         = errors.New("invalid correction ID")
	ErrInvalidTransactionKind      = errors.New("invalid transaction kind")
	ErrInvalidTransactionFields    = errors.New("invalid transaction fields")
	ErrInvalidDecimal              = errors.New("invalid decimal")
	ErrInvalidCurrency             = errors.New("invalid currency")
	ErrInvalidEffectiveAt          = errors.New("invalid effective time")
	ErrAssetFinanciallyIneligible  = errors.New("asset financially ineligible")
	ErrInvalidTransaction          = errors.New("invalid transaction")
	ErrTransactionNotCorrectable   = errors.New("transaction not correctable")
	ErrTransactionAlreadyCorrected = errors.New("transaction already corrected")
	ErrInvalidCorrection           = errors.New("invalid correction")
	ErrInvalidLedger               = errors.New("invalid ledger")
	ErrInsufficientOrderedQuantity = errors.New("insufficient ordered asset quantity")
)
