// Package application orchestrates immutable financial commands. HTTP, SQL,
// authentication transport, and financial projections are outside this package.
package application

import (
	"context"
	"errors"
	"time"

	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/audit"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
)

var (
	ErrUnauthenticated        = errors.New("authenticated principal required")
	ErrPortfolioNotFound      = errors.New("portfolio not found")
	ErrTransactionNotFound    = errors.New("transaction not found")
	ErrPortfolioArchived      = errors.New("portfolio archived")
	ErrAssetNotFound          = errors.New("asset not found")
	ErrInvalidInput           = errors.New("invalid transaction input")
	ErrInvalidIdempotencyKey  = errors.New("invalid idempotency key")
	ErrIdempotencyConflict    = errors.New("idempotency conflict")
	ErrInvalidBackdatedLedger = errors.New("invalid backdated ledger")
	ErrPersistence            = errors.New("transaction persistence failed")
)

const CreateScope = "transaction.create.v1"
const CorrectScope = "transaction.correct.v1"

type PortfolioReader interface {
	GetPortfolio(context.Context, identity.Principal, portfolio.PortfolioID) (portfolio.Portfolio, error)
}
type AssetReader interface {
	GetAsset(context.Context, identity.Principal, asset.AssetID) (asset.Asset, error)
}
type Clock interface{ Now() time.Time }
type IDs interface{ NewID() ([16]byte, error) }

// CommandInput contains application values, not an HTTP DTO. The zero Decimal
// means absent; trade fee alone defaults to zero. Text absence is preserved.
type CommandInput struct {
	Kind                             domain.Kind
	AssetID                          asset.AssetID
	Quantity, UnitPrice, Fee, Amount domain.Decimal
	Currency                         domain.Currency
	EffectiveAt                      time.Time
	Note, ExternalReference          domain.OptionalText
}
type CommandMetadata struct{ IdempotencyKey, CorrelationID string }
type Position struct {
	EffectiveAt time.Time
	Sequence    int64
	ID          domain.TransactionID
}
type HistoryInput struct {
	Kind             *domain.Kind
	From, To         *time.Time
	IncludeReversals *bool
	After            *Position
	Limit            int
}
type Record struct {
	Transaction domain.Transaction
	// Incoming creation links live on Transaction; outgoing links remain separate
	// so a replacement can itself become the original of another correction.
	DirectCorrection *domain.Correction
}
type History struct {
	Records []Record
	Next    *Position
}
type Result struct {
	Transaction domain.Transaction
	Correction  *domain.CorrectionFacts
}
type Idempotency struct {
	Fingerprint  [32]byte
	PrimaryID    domain.TransactionID
	CorrectionID domain.CorrectionID
}
type Repository interface {
	LockKey(context.Context, portfolio.PortfolioID, string, string) error
	Idempotency(context.Context, portfolio.PortfolioID, string, string, time.Time) (Idempotency, bool, error)
	Complete(context.Context, portfolio.PortfolioID, string, string, [32]byte, domain.TransactionID, Result, time.Time) error
	Allocate(context.Context, portfolio.PortfolioID, bool) (int64, int64, error)
	Replay(context.Context, portfolio.PortfolioID, asset.AssetID) ([]domain.Transaction, error)
	Get(context.Context, portfolio.PortfolioID, domain.TransactionID) (domain.Transaction, error)
	List(context.Context, portfolio.PortfolioID, HistoryInput) ([]domain.Transaction, error)
	DirectCorrection(context.Context, portfolio.PortfolioID, domain.TransactionID) (*domain.Correction, error)
	Correction(context.Context, portfolio.PortfolioID, domain.CorrectionID) (domain.CorrectionFacts, error)
	Insert(context.Context, domain.Transaction) error
	InsertCorrection(context.Context, domain.Correction) error
}
type UnitOfWork struct {
	Repository Repository
	// A transaction-bound Portfolio public reader holds a shared row lock until
	// commit, making an archive and an accepted financial command serializable.
	Portfolios PortfolioReader
	Assets     AssetReader
	Audit      audit.Store
	Outbox     outbox.Appender
}
type Transactor interface {
	WithinTransaction(context.Context, func(context.Context, UnitOfWork) error) error
}
