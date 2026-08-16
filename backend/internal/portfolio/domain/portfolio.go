package domain

import (
	"time"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

type BaseCurrency string

const BaseCurrencyUSD BaseCurrency = "USD"

func ParseBaseCurrency(value string) (BaseCurrency, error) {
	if BaseCurrency(value) != BaseCurrencyUSD {
		return "", ErrInvalidBaseCurrency
	}
	return BaseCurrencyUSD, nil
}

type PortfolioStatus string

const (
	PortfolioStatusActive   PortfolioStatus = "ACTIVE"
	PortfolioStatusArchived PortfolioStatus = "ARCHIVED"
)

func ParsePortfolioStatus(value string) (PortfolioStatus, error) {
	status := PortfolioStatus(value)
	if status != PortfolioStatusActive && status != PortfolioStatusArchived {
		return "", ErrInvalidPortfolioStatus
	}
	return status, nil
}

type Portfolio struct {
	id            PortfolioID
	ownerID       identitydomain.UserID
	name          PortfolioName
	baseCurrency  BaseCurrency
	status        PortfolioStatus
	archivedAt    time.Time
	hasArchivedAt bool
	createdAt     time.Time
	updatedAt     time.Time
}

func NewPortfolio(id PortfolioID, ownerID identitydomain.UserID, name PortfolioName, baseCurrency BaseCurrency, createdAt time.Time) (Portfolio, error) {
	return RehydratePortfolio(id, ownerID, name, baseCurrency, PortfolioStatusActive, nil, createdAt, createdAt)
}

func RehydratePortfolio(id PortfolioID, ownerID identitydomain.UserID, name PortfolioName, baseCurrency BaseCurrency, status PortfolioStatus, archivedAt *time.Time, createdAt, updatedAt time.Time) (Portfolio, error) {
	if id.IsZero() || ownerID.IsZero() || name.IsZero() || createdAt.IsZero() || updatedAt.IsZero() || updatedAt.Before(createdAt) {
		return Portfolio{}, ErrInvalidPortfolio
	}
	if _, err := ParseBaseCurrency(string(baseCurrency)); err != nil {
		return Portfolio{}, err
	}
	if _, err := ParsePortfolioStatus(string(status)); err != nil {
		return Portfolio{}, err
	}
	if status == PortfolioStatusActive && archivedAt != nil {
		return Portfolio{}, ErrInvalidPortfolio
	}
	if status == PortfolioStatusArchived && (archivedAt == nil || archivedAt.Before(createdAt) || archivedAt.After(updatedAt)) {
		return Portfolio{}, ErrInvalidPortfolio
	}
	portfolio := Portfolio{id: id, ownerID: ownerID, name: name, baseCurrency: baseCurrency, status: status, createdAt: createdAt, updatedAt: updatedAt}
	if archivedAt != nil {
		portfolio.archivedAt = *archivedAt
		portfolio.hasArchivedAt = true
	}
	return portfolio, nil
}

func (portfolio Portfolio) Rename(name PortfolioName, updatedAt time.Time) (Portfolio, error) {
	if portfolio.status == PortfolioStatusArchived {
		return Portfolio{}, ErrPortfolioArchived
	}
	if portfolio.status != PortfolioStatusActive || name.IsZero() || updatedAt.IsZero() || updatedAt.Before(portfolio.updatedAt) {
		return Portfolio{}, ErrInvalidPortfolio
	}
	portfolio.name = name
	portfolio.updatedAt = updatedAt
	return portfolio, nil
}

func (portfolio Portfolio) Archive(archivedAt time.Time) (Portfolio, error) {
	if portfolio.status == PortfolioStatusArchived {
		return portfolio, nil
	}
	if portfolio.status != PortfolioStatusActive || archivedAt.IsZero() || archivedAt.Before(portfolio.updatedAt) {
		return Portfolio{}, ErrInvalidPortfolio
	}
	portfolio.status = PortfolioStatusArchived
	portfolio.archivedAt = archivedAt
	portfolio.hasArchivedAt = true
	portfolio.updatedAt = archivedAt
	return portfolio, nil
}

func (portfolio Portfolio) ID() PortfolioID                { return portfolio.id }
func (portfolio Portfolio) OwnerID() identitydomain.UserID { return portfolio.ownerID }
func (portfolio Portfolio) Name() PortfolioName            { return portfolio.name }
func (portfolio Portfolio) BaseCurrency() BaseCurrency     { return portfolio.baseCurrency }
func (portfolio Portfolio) Status() PortfolioStatus        { return portfolio.status }
func (portfolio Portfolio) CreatedAt() time.Time           { return portfolio.createdAt }
func (portfolio Portfolio) UpdatedAt() time.Time           { return portfolio.updatedAt }
func (portfolio Portfolio) IsActive() bool                 { return portfolio.status == PortfolioStatusActive }
func (portfolio Portfolio) IsArchived() bool               { return portfolio.status == PortfolioStatusArchived }
func (portfolio Portfolio) ArchivedAt() (time.Time, bool) {
	return portfolio.archivedAt, portfolio.hasArchivedAt
}
