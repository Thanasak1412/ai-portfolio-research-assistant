// Package http implements the frozen Transaction contract using Identity's
// validated principal and the existing Transaction application boundary.
package http

import (
	"context"

	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/gofiber/fiber/v2"
)

type Operations interface {
	Create(context.Context, identity.Principal, portfolio.PortfolioID, application.CommandInput, application.CommandMetadata) (application.Result, error)
	Correct(context.Context, identity.Principal, portfolio.PortfolioID, domain.TransactionID, application.CommandInput, application.CommandMetadata) (application.Result, error)
	Get(context.Context, identity.Principal, portfolio.PortfolioID, domain.TransactionID) (application.Record, error)
	List(context.Context, identity.Principal, portfolio.PortfolioID, application.HistoryInput) (application.History, error)
}
type PrincipalExtractor func(*fiber.Ctx) (identity.Principal, bool)
type Handler struct {
	operations Operations
	bearer     fiber.Handler
	principal  PrincipalExtractor
}

func NewHandler(operations Operations, bearer fiber.Handler, principal PrincipalExtractor) (*Handler, error) {
	if operations == nil || bearer == nil || principal == nil {
		return nil, application.ErrInvalidInput
	}
	return &Handler{operations: operations, bearer: bearer, principal: principal}, nil
}
func (handler *Handler) Mount(router fiber.Router) {
	// Route-local middleware avoids authenticating unrelated Portfolio paths.
	base := "/portfolios/:portfolioId/transactions"
	router.Post(base, handler.bearer, handler.create)
	router.Get(base, handler.bearer, handler.list)
	router.Get(base+"/:transactionId", handler.bearer, handler.get)
	router.Post(base+"/:transactionId/corrections", handler.bearer, handler.correct)
}
func (handler *Handler) identity(ctx *fiber.Ctx) (identity.Principal, error) {
	principal, ok := handler.principal(ctx)
	if !ok {
		return identity.Principal{}, application.ErrUnauthenticated
	}
	return principal, nil
}
func portfolioID(ctx *fiber.Ctx) (portfolio.PortfolioID, error) {
	id, err := portfolio.ParsePortfolioID(ctx.Params("portfolioId"))
	if err != nil {
		return id, application.ErrPortfolioNotFound
	}
	return id, nil
}
func transactionID(ctx *fiber.Ctx) (domain.TransactionID, error) {
	id, err := domain.ParseTransactionID(ctx.Params("transactionId"))
	if err != nil {
		return id, application.ErrTransactionNotFound
	}
	return id, nil
}
func (handler *Handler) create(ctx *fiber.Ctx) error  { return handler.command(ctx, false) }
func (handler *Handler) correct(ctx *fiber.Ctx) error { return handler.command(ctx, true) }
func (handler *Handler) command(ctx *fiber.Ctx, correction bool) error {
	principal, err := handler.identity(ctx)
	if err != nil {
		return writeError(ctx, err, correction)
	}
	pid, err := portfolioID(ctx)
	if err != nil {
		return writeError(ctx, err, correction)
	}
	var target domain.TransactionID
	if correction {
		target, err = transactionID(ctx)
		if err != nil {
			return writeError(ctx, err, true)
		}
	}
	meta, err := metadata(ctx, platform.CorrelationID(ctx))
	if err != nil {
		return writeError(ctx, err, correction)
	}
	input, err := decodeCommand(ctx, correction)
	if err != nil {
		return writeError(ctx, err, correction)
	}
	if !correction {
		result, err := handler.operations.Create(ctx.UserContext(), principal, pid, input, meta)
		if err != nil {
			return writeError(ctx, err, false)
		}
		return ctx.Status(201).JSON(responseFromRecord(application.Record{Transaction: result.Transaction}))
	}
	result, err := handler.operations.Correct(ctx.UserContext(), principal, pid, target, input, meta)
	if err != nil {
		return writeError(ctx, err, true)
	}
	if result.Correction == nil {
		return writeError(ctx, application.ErrPersistence, true)
	}
	// The committed result owns the correction relationship. Read only the
	// immutable original fact; later corrections must not change replay output.
	original, err := handler.operations.Get(ctx.UserContext(), principal, pid, target)
	if err != nil {
		return writeError(ctx, application.ErrPersistence, true)
	}
	original.DirectCorrection = &result.Correction.Relationship
	return ctx.Status(201).JSON(correctionResponse{
		Original:    responseFromRecord(original),
		Reversal:    responseFromRecord(application.Record{Transaction: result.Correction.Reversal}),
		Replacement: responseFromRecord(application.Record{Transaction: result.Correction.Replacement}),
	})
}
func (handler *Handler) get(ctx *fiber.Ctx) error {
	principal, err := handler.identity(ctx)
	if err != nil {
		return writeError(ctx, err, true)
	}
	pid, err := portfolioID(ctx)
	if err != nil {
		return writeError(ctx, err, true)
	}
	id, err := transactionID(ctx)
	if err != nil {
		return writeError(ctx, err, true)
	}
	record, err := handler.operations.Get(ctx.UserContext(), principal, pid, id)
	if err != nil {
		return writeError(ctx, err, true)
	}
	return ctx.JSON(responseFromRecord(record))
}
func (handler *Handler) list(ctx *fiber.Ctx) error {
	principal, err := handler.identity(ctx)
	if err != nil {
		return writeError(ctx, err, false)
	}
	pid, err := portfolioID(ctx)
	if err != nil {
		return writeError(ctx, err, false)
	}
	input, err := historyInput(ctx)
	if err != nil {
		return writeError(ctx, err, false)
	}
	history, err := handler.operations.List(ctx.UserContext(), principal, pid, input)
	if err != nil {
		return writeError(ctx, err, false)
	}
	response := historyResponse{Items: make([]transactionResponse, 0, len(history.Records))}
	for _, record := range history.Records {
		response.Items = append(response.Items, responseFromRecord(record))
	}
	if history.Next != nil {
		cursor := encodeCursor(*history.Next)
		response.NextCursor = &cursor
	}
	return ctx.JSON(response)
}
