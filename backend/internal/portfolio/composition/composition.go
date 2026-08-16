// Package composition wires Portfolio runtime dependencies without exposing
// infrastructure details to the HTTP transport.
package composition

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	portfoliodatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/infrastructure/database"
	portfolioruntime "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/infrastructure/runtime"
	portfoliohttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/transport/http"
)

func BuildHTTP(pool *pgxpool.Pool, bearer fiber.Handler, principal func(*fiber.Ctx) (identitydomain.Principal, bool)) (*portfoliohttp.Handler, error) {
	repository := portfoliodatabase.NewPostgresPortfolioRepository(pool)
	service, err := application.NewService(application.ServiceDependencies{Repository: repository, Clock: portfolioruntime.Clock{}, IDs: portfolioruntime.IDGenerator{}})
	if err != nil {
		return nil, err
	}
	return portfoliohttp.NewHandler(service, bearer, principal)
}
