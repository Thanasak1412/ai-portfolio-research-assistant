// Package composition constructs Transaction application operations and HTTP transport.
package composition

import (
	"time"

	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/database"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/infrastructure/database"
	transactionhttp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/transport/http"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type clock struct{}

func (clock) Now() time.Time { return time.Now() }

type identifiers struct{}

func (identifiers) NewID() ([16]byte, error) { id, err := uuid.NewRandom(); return id, err }

func Build(pool *pgxpool.Pool, portfolios database.OwnershipBinder, assets database.AssetBinder) (*application.Service, error) {
	if pool == nil || portfolios == nil || assets == nil {
		return nil, application.ErrInvalidInput
	}
	return application.NewService(application.Dependencies{Transactor: database.NewPostgresTransactor(pool, portfolios, assets), Clock: clock{}, IDs: identifiers{}, Rejections: platform.NewPlatformAuditStore(pool)})
}

func BuildHTTP(pool *pgxpool.Pool, portfolios database.OwnershipBinder, assets database.AssetBinder, bearer fiber.Handler, principal func(*fiber.Ctx) (identity.Principal, bool)) (*transactionhttp.Handler, error) {
	service, err := Build(pool, portfolios, assets)
	if err != nil {
		return nil, err
	}
	return transactionhttp.NewHandler(service, bearer, principal)
}
