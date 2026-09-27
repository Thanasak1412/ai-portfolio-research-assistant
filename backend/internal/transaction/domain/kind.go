package domain

type Kind string

const (
	KindBuy        Kind = "BUY"
	KindSell       Kind = "SELL"
	KindDividend   Kind = "DIVIDEND"
	KindDeposit    Kind = "DEPOSIT"
	KindWithdrawal Kind = "WITHDRAWAL"
	KindFee        Kind = "FEE"
	KindReversal   Kind = "REVERSAL"
)

func (kind Kind) IsPublicCreatable() bool {
	switch kind {
	case KindBuy, KindSell, KindDividend, KindDeposit, KindWithdrawal, KindFee:
		return true
	default:
		return false
	}
}
func ParseKind(value string) (Kind, error) {
	kind := Kind(value)
	if kind.IsPublicCreatable() || kind == KindReversal {
		return kind, nil
	}
	return "", ErrInvalidTransactionKind
}
