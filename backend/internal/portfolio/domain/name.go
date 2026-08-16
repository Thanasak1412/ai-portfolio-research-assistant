package domain

import "unicode/utf8"

const portfolioNameMaximumCharacters = 200

// PortfolioName preserves display characters and internal whitespace. Only the
// narrow ASCII whitespace set used by PostgreSQL btrim is removed at each edge.
type PortfolioName struct{ value string }

func NewPortfolioName(value string) (PortfolioName, error) {
	if !utf8.ValidString(value) {
		return PortfolioName{}, ErrInvalidPortfolioName
	}
	value = trimPortfolioEdgeWhitespace(value)
	if value == "" || utf8.RuneCountInString(value) > portfolioNameMaximumCharacters {
		return PortfolioName{}, ErrInvalidPortfolioName
	}
	return PortfolioName{value: value}, nil
}

func (name PortfolioName) IsZero() bool   { return name.value == "" }
func (name PortfolioName) String() string { return name.value }

func trimPortfolioEdgeWhitespace(value string) string {
	start, end := 0, len(value)
	for start < end && isPortfolioTrimByte(value[start]) {
		start++
	}
	for end > start && isPortfolioTrimByte(value[end-1]) {
		end--
	}
	return value[start:end]
}

func isPortfolioTrimByte(value byte) bool {
	switch value {
	case 0x20, 0x09, 0x0A, 0x0D, 0x0C, 0x0B:
		return true
	default:
		return false
	}
}
