// Package composition wires Asset read dependencies for the API composition root.
package composition

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	assetdatabase "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/infrastructure/database"
	assethttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/transport/http"
	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

func BuildHTTP(pool *pgxpool.Pool, bearer fiber.Handler, principal func(*fiber.Ctx) (identitydomain.Principal, bool)) (*assethttp.Handler, error) {
	service, err := application.NewService(assetdatabase.NewPostgresAssetRepository(pool))
	if err != nil {
		return nil, err
	}
	return assethttp.NewHandler(service, bearer, principal)
}
