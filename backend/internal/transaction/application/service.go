package application

import (
	"context"
	"errors"
	"time"

	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/audit"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/platform/outbox"
	portfolioapp "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
)

type Dependencies struct {
	Transactor Transactor
	Clock      Clock
	IDs        IDs
	Rejections audit.Store
}
type Service struct{ dependencies Dependencies }

func NewService(dependencies Dependencies) (*Service, error) {
	if dependencies.Transactor == nil || dependencies.Clock == nil || dependencies.IDs == nil || dependencies.Rejections == nil {
		return nil, ErrInvalidInput
	}
	return &Service{dependencies}, nil
}
func (s *Service) now() time.Time { return s.dependencies.Clock.Now().UTC().Truncate(time.Microsecond) }

func owned(ctx context.Context, u UnitOfWork, principal identity.Principal, id portfolio.PortfolioID) (portfolio.Portfolio, error) {
	if !principal.IsAuthenticated() {
		return portfolio.Portfolio{}, ErrUnauthenticated
	}
	if id.IsZero() {
		return portfolio.Portfolio{}, ErrPortfolioNotFound
	}
	p, err := u.Portfolios.GetPortfolio(ctx, principal, id)
	if errors.Is(err, portfolioapp.ErrPortfolioNotFound) || errors.Is(err, portfolioapp.ErrInvalidPortfolioInput) {
		return portfolio.Portfolio{}, ErrPortfolioNotFound
	}
	if err != nil {
		return portfolio.Portfolio{}, err
	}
	owner, _ := principal.UserID()
	if p.ID() != id || p.OwnerID() != owner {
		return portfolio.Portfolio{}, ErrPortfolioNotFound
	}
	return p, nil
}

func (s *Service) Create(ctx context.Context, principal identity.Principal, portfolioID portfolio.PortfolioID, input CommandInput, metadata CommandMetadata) (Result, error) {
	return s.execute(ctx, principal, portfolioID, domain.TransactionID{}, input, metadata, false)
}
func (s *Service) Correct(ctx context.Context, principal identity.Principal, portfolioID portfolio.PortfolioID, original domain.TransactionID, replacement CommandInput, metadata CommandMetadata) (Result, error) {
	return s.execute(ctx, principal, portfolioID, original, replacement, metadata, true)
}

func (s *Service) execute(ctx context.Context, principal identity.Principal, portfolioID portfolio.PortfolioID, target domain.TransactionID, input CommandInput, metadata CommandMetadata, correction bool) (Result, error) {
	if !principal.IsAuthenticated() {
		return Result{}, ErrUnauthenticated
	}
	if !correlationPattern.MatchString(metadata.CorrelationID) {
		return Result{}, ErrInvalidInput
	}
	var result Result
	err := s.dependencies.Transactor.WithinTransaction(ctx, func(ctx context.Context, u UnitOfWork) error {
		p, err := owned(ctx, u, principal, portfolioID)
		if err != nil {
			return err
		}
		if correction && target.IsZero() {
			return ErrTransactionNotFound
		}
		if !keyPattern.MatchString(metadata.IdempotencyKey) {
			return ErrInvalidIdempotencyKey
		}
		now := s.now()
		normalized, err := normalize(input, now)
		if err != nil {
			return err
		}
		scope := CreateScope
		if correction {
			scope = CorrectScope
		}
		digest := fingerprint(scope, target, normalized)
		if err := u.Repository.LockKey(ctx, portfolioID, scope, metadata.IdempotencyKey); err != nil {
			return err
		}
		previous, found, err := u.Repository.Idempotency(ctx, portfolioID, scope, metadata.IdempotencyKey, s.now())
		if err != nil {
			return err
		}
		if found {
			if previous.Fingerprint != digest {
				return ErrIdempotencyConflict
			}
			result.Transaction, err = u.Repository.Get(ctx, portfolioID, previous.PrimaryID)
			if err != nil {
				return err
			}
			if !previous.CorrectionID.IsZero() {
				facts, err := u.Repository.Correction(ctx, portfolioID, previous.CorrectionID)
				if err != nil {
					return err
				}
				result.Correction = &facts
			}
			return s.appendAudit(ctx, u.Audit, principal, portfolioID, result.Transaction.ID(), previous.CorrectionID, metadata.CorrelationID, audit.ActionTransactionIdempotentReplay, true)
		}
		if p.IsArchived() {
			return ErrPortfolioArchived
		}
		command, err := s.resolveCommand(ctx, u.Assets, principal, normalized, now)
		if err != nil {
			return err
		}
		first, second, err := u.Repository.Allocate(ctx, portfolioID, correction)
		if err != nil {
			return err
		}
		actor, _ := principal.UserID()
		createdAt := s.now()
		if !correction {
			id, err := s.newTransactionID()
			if err != nil {
				return err
			}
			fact, err := domain.NewTransactionFromCommand(id, portfolioID, actor, first, createdAt, command)
			if err != nil {
				return err
			}
			if err := validateReplay(ctx, u.Repository, portfolioID, []domain.Transaction{fact}); err != nil {
				return err
			}
			if err := u.Repository.Insert(ctx, fact); err != nil {
				return err
			}
			result.Transaction = fact
			if err := s.appendAudit(ctx, u.Audit, principal, portfolioID, id, domain.CorrectionID{}, metadata.CorrelationID, audit.ActionTransactionCreateSuccess, true); err != nil {
				return err
			}
			if err := s.appendEvent(ctx, u.Outbox, portfolioID, metadata.CorrelationID, "transaction.recorded.v1", []outbox.Reference{{Role: "transaction", ID: id.Bytes()}}); err != nil {
				return err
			}
		} else {
			original, err := u.Repository.Get(ctx, portfolioID, target)
			if err != nil {
				return err
			}
			direct, err := u.Repository.DirectCorrection(ctx, portfolioID, target)
			if err != nil {
				return err
			}
			if err := domain.ValidateCorrectionTarget(original, direct != nil); err != nil {
				return err
			}
			reversalID, err := s.newTransactionID()
			if err != nil {
				return err
			}
			replacementID, err := s.newTransactionID()
			if err != nil {
				return err
			}
			bytes, err := s.dependencies.IDs.NewID()
			if err != nil {
				return ErrPersistence
			}
			correctionID, err := domain.NewCorrectionID(bytes)
			if err != nil {
				return err
			}
			facts, err := domain.BuildCorrection(domain.CorrectionInput{Original: original, Replacement: command, CorrectionID: correctionID, ReversalID: reversalID, ReplacementID: replacementID, Actor: actor, ReversalSequence: first, ReplacementSequence: second, CreatedAt: createdAt})
			if err != nil {
				return err
			}
			if err := validateReplay(ctx, u.Repository, portfolioID, []domain.Transaction{facts.Reversal, facts.Replacement}, original); err != nil {
				return err
			}
			if err := u.Repository.InsertCorrection(ctx, facts.Relationship); err != nil {
				return err
			}
			for _, fact := range []domain.Transaction{facts.Reversal, facts.Replacement} {
				if err := u.Repository.Insert(ctx, fact); err != nil {
					return err
				}
			}
			for _, entry := range []struct {
				action audit.Action
				id     domain.TransactionID
			}{
				{audit.ActionTransactionCorrectionInitiated, target}, {audit.ActionTransactionReversalCreated, reversalID}, {audit.ActionTransactionReplacementCreated, replacementID}, {audit.ActionTransactionCorrectionCompleted, target},
			} {
				if err := s.appendAudit(ctx, u.Audit, principal, portfolioID, entry.id, correctionID, metadata.CorrelationID, entry.action, true); err != nil {
					return err
				}
			}
			if err := s.appendEvent(ctx, u.Outbox, portfolioID, metadata.CorrelationID, "transaction.corrected.v1", []outbox.Reference{{Role: "original_transaction", ID: target.Bytes()}, {Role: "reversal_transaction", ID: reversalID.Bytes()}, {Role: "replacement_transaction", ID: replacementID.Bytes()}, {Role: "correction", ID: correctionID.Bytes()}}); err != nil {
				return err
			}
			result = Result{Transaction: facts.Replacement, Correction: &facts}
		}
		return u.Repository.Complete(ctx, portfolioID, scope, metadata.IdempotencyKey, digest, target, result, s.now())
	})
	if err != nil {
		if isRejection(err) {
			action := audit.ActionTransactionCreateFailure
			if correction {
				action = audit.ActionTransactionCorrectionRejected
			}
			if errors.Is(err, ErrIdempotencyConflict) {
				action = audit.ActionTransactionIdempotencyConflict
			}
			if errors.Is(err, ErrPortfolioNotFound) {
				action = audit.ActionTransactionOwnershipRejection
			}
			// Only a rejected-command outcome is appended after rollback. Success
			// evidence and events never escape their authoritative transaction.
			if auditErr := s.appendAudit(ctx, s.dependencies.Rejections, principal, portfolioID, target, domain.CorrectionID{}, metadata.CorrelationID, action, false); auditErr != nil {
				return Result{}, ErrPersistence
			}
		}
		return Result{}, err
	}
	return result, nil
}

func validateReplay(ctx context.Context, repository Repository, portfolioID portfolio.PortfolioID, candidates []domain.Transaction, originals ...domain.Transaction) error {
	var existing []domain.Transaction
	seen := map[asset.AssetID]bool{}
	for _, fact := range candidates {
		item, present := fact.Asset()
		if !present || seen[item.ID()] {
			continue
		}
		seen[item.ID()] = true
		rows, err := repository.Replay(ctx, portfolioID, item.ID())
		if err != nil {
			return err
		}
		existing = append(existing, rows...)
	}
	for _, original := range originals {
		found := false
		for _, fact := range existing {
			if fact.ID() == original.ID() {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, original)
		}
	}
	err := domain.ReplayLedger(existing, candidates)
	var violation *domain.LedgerViolation
	if errors.As(err, &violation) && violation.LaterExisting {
		return ErrInvalidBackdatedLedger
	}
	return err
}

func (s *Service) newTransactionID() (domain.TransactionID, error) {
	id, err := s.dependencies.IDs.NewID()
	if err != nil {
		return domain.TransactionID{}, ErrPersistence
	}
	return domain.NewTransactionID(id)
}
func (s *Service) appendAudit(ctx context.Context, store audit.Store, principal identity.Principal, pid portfolio.PortfolioID, tid domain.TransactionID, cid domain.CorrectionID, correlation string, action audit.Action, success bool) error {
	id, err := s.dependencies.IDs.NewID()
	if err != nil {
		return ErrPersistence
	}
	actor, _ := principal.UserID()
	actorBytes := actor.Bytes()
	optional := func(id [16]byte) *[16]byte {
		if id == [16]byte{} {
			return nil
		}
		return &id
	}
	record := audit.Record{EventID: id, OccurredAt: s.now(), Action: action, Result: audit.ResultSuccess, Severity: audit.SeverityInfo, ActorUserID: &actorBytes, PortfolioID: optional(pid.Bytes()), TransactionID: optional(tid.Bytes()), CorrectionID: optional(cid.Bytes()), CorrelationID: correlation}
	if !success {
		record.Result = audit.ResultFailure
		record.Severity = audit.SeverityWarning
	}
	if err := store.Append(ctx, record); err != nil {
		return ErrPersistence
	}
	return nil
}
func (s *Service) appendEvent(ctx context.Context, store outbox.Appender, pid portfolio.PortfolioID, correlation, eventType string, refs []outbox.Reference) error {
	id, err := s.dependencies.IDs.NewID()
	if err != nil {
		return ErrPersistence
	}
	now := s.now()
	if err := store.Append(ctx, outbox.Event{ID: id, Type: eventType, Version: 1, AggregateType: "portfolio", AggregateID: pid.Bytes(), PortfolioID: pid.Bytes(), OccurredAt: now, CorrelationID: correlation, Payload: outbox.Payload{SchemaVersion: 1, References: refs}, NextAttemptAt: now}); err != nil {
		return ErrPersistence
	}
	return nil
}
func isRejection(err error) bool {
	for _, candidate := range []error{ErrPortfolioNotFound, ErrTransactionNotFound, ErrPortfolioArchived, ErrAssetNotFound, ErrInvalidInput, ErrInvalidIdempotencyKey, ErrIdempotencyConflict, ErrInvalidBackdatedLedger, domain.ErrInvalidTransactionKind, domain.ErrInvalidTransactionFields, domain.ErrInvalidCurrency, domain.ErrInvalidEffectiveAt, domain.ErrAssetFinanciallyIneligible, domain.ErrTransactionNotCorrectable, domain.ErrTransactionAlreadyCorrected, domain.ErrInsufficientOrderedQuantity} {
		if errors.Is(err, candidate) {
			return true
		}
	}
	return false
}
