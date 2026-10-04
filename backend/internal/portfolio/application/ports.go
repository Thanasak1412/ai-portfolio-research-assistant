package application

import (
	"context"
	"time"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

type PortfolioRepository interface {
	Create(context.Context, domain.Portfolio) (domain.Portfolio, error)
	FindOwnedByID(context.Context, identitydomain.UserID, domain.PortfolioID) (domain.Portfolio, error)
	ListOwnedByStatus(context.Context, identitydomain.UserID, domain.PortfolioStatus) ([]domain.Portfolio, error)
	UpdateOwnedActiveName(context.Context, identitydomain.UserID, domain.PortfolioID, domain.PortfolioName, time.Time) (domain.Portfolio, error)
	ArchiveOwnedActive(context.Context, identitydomain.UserID, domain.PortfolioID, time.Time) (domain.Portfolio, error)
}

type Clock interface{ Now() time.Time }

// OwnershipReader is the public owner-scoped read boundary. A transaction-bound
// implementation may hold a shared lock until its caller's financial write ends.
type OwnershipReader interface {
	GetPortfolio(context.Context, identitydomain.Principal, domain.PortfolioID) (domain.Portfolio, error)
}

type IDGenerator interface {
	PortfolioID() (domain.PortfolioID, error)
}
