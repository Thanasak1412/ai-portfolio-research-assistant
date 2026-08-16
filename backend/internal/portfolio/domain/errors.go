package domain

import "errors"

var (
	ErrInvalidPortfolioID     = errors.New("invalid portfolio identifier")
	ErrInvalidPortfolioName   = errors.New("invalid portfolio name")
	ErrInvalidBaseCurrency    = errors.New("invalid portfolio base currency")
	ErrInvalidPortfolioStatus = errors.New("invalid portfolio status")
	ErrInvalidPortfolio       = errors.New("invalid portfolio")
	ErrPortfolioArchived      = errors.New("portfolio is archived")
)
