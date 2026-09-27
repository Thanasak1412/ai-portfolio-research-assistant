package domain

import (
	"math/big"
	"strings"
)

// Decimal stores a canonical exact base-ten value. The unexported string avoids
// exposing mutable big.Int pointers through copies or accessors.
type Decimal struct{ canonical string }

func ParsePositiveDecimal(value string) (Decimal, error)    { return parseDecimal(value, false) }
func ParseNonNegativeDecimal(value string) (Decimal, error) { return parseDecimal(value, true) }

func parseDecimal(value string, allowZero bool) (Decimal, error) {
	if value == "" {
		return Decimal{}, ErrInvalidDecimal
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || (len(parts[0]) > 1 && parts[0][0] == '0') {
		return Decimal{}, ErrInvalidDecimal
	}
	for _, ch := range parts[0] {
		if ch < '0' || ch > '9' {
			return Decimal{}, ErrInvalidDecimal
		}
	}
	if len(parts) == 2 {
		if len(parts[1]) == 0 || len(parts[1]) > 12 {
			return Decimal{}, ErrInvalidDecimal
		}
		for _, ch := range parts[1] {
			if ch < '0' || ch > '9' {
				return Decimal{}, ErrInvalidDecimal
			}
		}
	}
	canonical := parts[0]
	if len(parts) == 2 {
		fraction := strings.TrimRight(parts[1], "0")
		if fraction != "" {
			canonical += "." + fraction
		}
	}
	if canonical == "0" && !allowZero {
		return Decimal{}, ErrInvalidDecimal
	}
	return Decimal{canonical}, nil
}

func (d Decimal) IsValid() bool       { return d.canonical != "" }
func (d Decimal) String() string      { return d.canonical }
func (d Decimal) IsPositive() bool    { return d.IsValid() && d.canonical != "0" && d.canonical[0] != '-' }
func (d Decimal) IsNonNegative() bool { return d.IsValid() && d.canonical[0] != '-' }

func (d Decimal) scaled() *big.Int {
	if !d.IsValid() {
		return nil
	}
	negative := strings.HasPrefix(d.canonical, "-")
	value := strings.TrimPrefix(d.canonical, "-")
	parts := strings.Split(value, ".")
	digits := parts[0]
	fracLen := 0
	if len(parts) == 2 {
		digits += parts[1]
		fracLen = len(parts[1])
	}
	n, _ := new(big.Int).SetString(digits, 10)
	n.Mul(n, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(12-fracLen)), nil))
	if negative {
		n.Neg(n)
	}
	return n
}

func decimalFromScaled(n *big.Int) Decimal {
	if n.Sign() == 0 {
		return Decimal{"0"}
	}
	negative := n.Sign() < 0
	abs := new(big.Int).Abs(n)
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(abs, big.NewInt(1_000_000_000_000), r)
	value := q.String()
	if r.Sign() != 0 {
		frac := r.String()
		frac = strings.Repeat("0", 12-len(frac)) + frac
		value += "." + strings.TrimRight(frac, "0")
	}
	if negative {
		value = "-" + value
	}
	return Decimal{value}
}

func (d Decimal) Add(other Decimal) (Decimal, error) {
	if !d.IsValid() || !other.IsValid() {
		return Decimal{}, ErrInvalidDecimal
	}
	return decimalFromScaled(new(big.Int).Add(d.scaled(), other.scaled())), nil
}
func (d Decimal) Sub(other Decimal) (Decimal, error) {
	if !d.IsValid() || !other.IsValid() {
		return Decimal{}, ErrInvalidDecimal
	}
	return decimalFromScaled(new(big.Int).Sub(d.scaled(), other.scaled())), nil
}
func (d Decimal) Compare(other Decimal) (int, error) {
	if !d.IsValid() || !other.IsValid() {
		return 0, ErrInvalidDecimal
	}
	return d.scaled().Cmp(other.scaled()), nil
}
