package http

import (
	"errors"

	platform "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/httpserver"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/gofiber/fiber/v2"
)

func writeError(ctx *fiber.Ctx, err error, individual bool) error {
	status, code := 500, "INTERNAL_ERROR"
	switch {
	case errors.Is(err, application.ErrUnauthenticated):
		status, code = 401, "ACCESS_TOKEN_INVALID"
	case errors.Is(err, application.ErrPortfolioNotFound):
		status, code = 404, "PORTFOLIO_NOT_FOUND"
		if individual {
			code = "TRANSACTION_NOT_FOUND"
		}
	case errors.Is(err, application.ErrTransactionNotFound):
		status, code = 404, "TRANSACTION_NOT_FOUND"
	case errors.Is(err, application.ErrInvalidInput):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(err, application.ErrInvalidIdempotencyKey):
		status, code = 400, "INVALID_IDEMPOTENCY_KEY"
	case errors.Is(err, domain.ErrInvalidTransactionKind):
		status, code = 400, "UNSUPPORTED_TRANSACTION_KIND"
	case errors.Is(err, domain.ErrInvalidCurrency):
		status, code = 400, "UNSUPPORTED_TRANSACTION_CURRENCY"
	case errors.Is(err, domain.ErrInvalidTransactionFields):
		status, code = 400, "INVALID_TRANSACTION_FIELDS"
	case errors.Is(err, domain.ErrInvalidDecimal):
		status, code = 400, "INVALID_DECIMAL"
	case errors.Is(err, application.ErrIdempotencyConflict):
		status, code = 409, "IDEMPOTENCY_CONFLICT"
	case errors.Is(err, domain.ErrTransactionAlreadyCorrected):
		status, code = 409, "TRANSACTION_ALREADY_CORRECTED"
	case errors.Is(err, application.ErrAssetNotFound):
		status, code = 422, "ASSET_NOT_FOUND"
	case errors.Is(err, domain.ErrAssetFinanciallyIneligible):
		status, code = 422, "ASSET_FINANCIALLY_INELIGIBLE"
	case errors.Is(err, domain.ErrInvalidEffectiveAt):
		status, code = 422, "INVALID_EFFECTIVE_AT"
	case errors.Is(err, domain.ErrInsufficientOrderedQuantity):
		status, code = 422, "INSUFFICIENT_ORDERED_ASSET_QUANTITY"
	case errors.Is(err, application.ErrInvalidBackdatedLedger):
		status, code = 422, "INVALID_BACKDATED_LEDGER"
	case errors.Is(err, application.ErrPortfolioArchived):
		status, code = 422, "PORTFOLIO_ARCHIVED"
	case errors.Is(err, domain.ErrTransactionNotCorrectable):
		status, code = 422, "TRANSACTION_NOT_CORRECTABLE"
	}
	// Never serialize domain/persistence error text or submitted financial data.
	message := "The Transaction request was rejected"
	if status == 500 {
		message = "An internal error occurred"
	}
	return ctx.Status(status).JSON(platform.ErrorEnvelope{Error: platform.ErrorDetail{Code: code, Message: message, CorrelationID: platform.CorrelationID(ctx)}})
}
