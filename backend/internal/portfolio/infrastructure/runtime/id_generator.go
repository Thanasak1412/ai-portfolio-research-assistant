package runtime

import (
	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

// IDGenerator creates cryptographically random opaque Portfolio IDs.
type IDGenerator struct{}

func (IDGenerator) PortfolioID() (domain.PortfolioID, error) {
	value, err := uuid.NewRandom()
	if err != nil {
		return domain.PortfolioID{}, err
	}
	return domain.NewPortfolioID(value)
}
