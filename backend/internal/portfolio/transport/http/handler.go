// Package http exposes the frozen M2 Portfolio HTTP contract.
package http

import (
	"context"

	"github.com/gofiber/fiber/v2"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

// Operations is the application boundary used by the HTTP transport.
type Operations interface {
	CreatePortfolio(context.Context, identitydomain.Principal, application.CreatePortfolioInput) (domain.Portfolio, error)
	ListPortfolios(context.Context, identitydomain.Principal, domain.PortfolioStatus) ([]domain.Portfolio, error)
	GetPortfolio(context.Context, identitydomain.Principal, domain.PortfolioID) (domain.Portfolio, error)
	UpdatePortfolio(context.Context, identitydomain.Principal, domain.PortfolioID, application.UpdatePortfolioInput) (domain.Portfolio, error)
	ArchivePortfolio(context.Context, identitydomain.Principal, domain.PortfolioID) (domain.Portfolio, error)
}

// PrincipalExtractor reads only an Identity-validated principal supplied by
// the injected bearer middleware.
type PrincipalExtractor func(*fiber.Ctx) (identitydomain.Principal, bool)

type Handler struct {
	operations Operations
	bearer     fiber.Handler
	principal  PrincipalExtractor
}

func NewHandler(operations Operations, bearer fiber.Handler, principal PrincipalExtractor) (*Handler, error) {
	if operations == nil || bearer == nil || principal == nil {
		return nil, application.ErrPortfolioService
	}
	return &Handler{operations: operations, bearer: bearer, principal: principal}, nil
}

func (handler *Handler) Mount(router fiber.Router) {
	portfolios := router.Group("/portfolios", handler.bearer)
	portfolios.Post("/", handler.create)
	portfolios.Get("/", handler.list)
	portfolios.Get("/:portfolioId", handler.get)
	portfolios.Patch("/:portfolioId", handler.update)
	portfolios.Post("/:portfolioId/archive", handler.archive)
}

func (handler *Handler) create(ctx *fiber.Ctx) error {
	var request createPortfolioRequest
	if err := decodeJSON(ctx, &request); err != nil || request.Name == nil || request.BaseCurrency == nil || *request.BaseCurrency != string(domain.BaseCurrencyUSD) {
		return writeError(ctx, errInvalidJSON)
	}
	portfolio, err := handler.operations.CreatePortfolio(ctx.UserContext(), handler.validPrincipal(ctx), application.CreatePortfolioInput{Name: *request.Name, BaseCurrency: *request.BaseCurrency})
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.Status(fiber.StatusCreated).JSON(responseFromPortfolio(portfolio))
}

func (handler *Handler) list(ctx *fiber.Ctx) error {
	status := domain.PortfolioStatusActive
	if ctx.Context().QueryArgs().Has("status") {
		parsed, err := domain.ParsePortfolioStatus(ctx.Query("status"))
		if err != nil {
			return writeError(ctx, errInvalidJSON)
		}
		status = parsed
	}
	portfolios, err := handler.operations.ListPortfolios(ctx.UserContext(), handler.validPrincipal(ctx), status)
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.JSON(responseFromPortfolios(portfolios))
}

func (handler *Handler) get(ctx *fiber.Ctx) error {
	id, err := domain.ParsePortfolioID(ctx.Params("portfolioId"))
	if err != nil {
		return writeError(ctx, application.ErrPortfolioNotFound)
	}
	portfolio, err := handler.operations.GetPortfolio(ctx.UserContext(), handler.validPrincipal(ctx), id)
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.JSON(responseFromPortfolio(portfolio))
}

func (handler *Handler) update(ctx *fiber.Ctx) error {
	id, err := domain.ParsePortfolioID(ctx.Params("portfolioId"))
	if err != nil {
		return writeError(ctx, application.ErrPortfolioNotFound)
	}
	var request updatePortfolioRequest
	if err := decodeJSON(ctx, &request); err != nil || request.Name == nil {
		return writeError(ctx, errInvalidJSON)
	}
	portfolio, err := handler.operations.UpdatePortfolio(ctx.UserContext(), handler.validPrincipal(ctx), id, application.UpdatePortfolioInput{Name: *request.Name})
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.JSON(responseFromPortfolio(portfolio))
}

func (handler *Handler) archive(ctx *fiber.Ctx) error {
	id, err := domain.ParsePortfolioID(ctx.Params("portfolioId"))
	if err != nil {
		return writeError(ctx, application.ErrPortfolioNotFound)
	}
	portfolio, err := handler.operations.ArchivePortfolio(ctx.UserContext(), handler.validPrincipal(ctx), id)
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.JSON(responseFromPortfolio(portfolio))
}

func (handler *Handler) validPrincipal(ctx *fiber.Ctx) identitydomain.Principal {
	principal, ok := handler.principal(ctx)
	if !ok {
		return identitydomain.Principal{}
	}
	return principal
}

var _ interface{ Mount(fiber.Router) } = (*Handler)(nil)
