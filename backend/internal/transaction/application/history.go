package application

import (
	"context"
	"time"

	identity "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	portfolio "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
)

func (s *Service) Get(ctx context.Context, principal identity.Principal, pid portfolio.PortfolioID, id domain.TransactionID) (Record, error) {
	var result Record
	err := s.dependencies.Transactor.WithinTransaction(ctx, func(ctx context.Context, u UnitOfWork) error {
		if _, err := owned(ctx, u, principal, pid); err != nil {
			return err
		}
		if id.IsZero() {
			return ErrTransactionNotFound
		}
		fact, err := u.Repository.Get(ctx, pid, id)
		if err != nil {
			return err
		}
		direct, err := u.Repository.DirectCorrection(ctx, pid, id)
		if err != nil {
			return err
		}
		result = Record{fact, direct}
		return nil
	})
	if err != nil {
		return Record{}, err
	}
	return result, nil
}
func validHistoryTime(value time.Time) bool { return !value.IsZero() && value.Nanosecond()%1000 == 0 }

func (s *Service) List(ctx context.Context, principal identity.Principal, pid portfolio.PortfolioID, input HistoryInput) (History, error) {
	var result History
	err := s.dependencies.Transactor.WithinTransaction(ctx, func(ctx context.Context, u UnitOfWork) error {
		if _, err := owned(ctx, u, principal, pid); err != nil {
			return err
		}
		if input.Limit == 0 {
			input.Limit = 50
		}
		if input.Limit < 1 || input.Limit > 100 {
			return ErrInvalidInput
		}
		if input.Kind != nil {
			if _, err := domain.ParseKind(string(*input.Kind)); err != nil {
				return ErrInvalidInput
			}
		}
		for _, timestamp := range []*time.Time{input.From, input.To} {
			if timestamp != nil && !validHistoryTime(*timestamp) {
				return ErrInvalidInput
			}
		}
		if input.From != nil && input.To != nil && input.From.After(*input.To) {
			return ErrInvalidInput
		}
		if input.After != nil && (!validHistoryTime(input.After.EffectiveAt) || input.After.Sequence < 1 || input.After.ID.IsZero()) {
			return ErrInvalidInput
		}
		limit := input.Limit
		input.Limit++
		facts, err := u.Repository.List(ctx, pid, input)
		if err != nil {
			return err
		}
		if len(facts) > limit {
			facts = facts[:limit]
			last := facts[limit-1]
			result.Next = &Position{last.EffectiveAt(), last.PortfolioSequence(), last.ID()}
		}
		result.Records = make([]Record, 0, len(facts))
		for _, fact := range facts {
			direct, err := u.Repository.DirectCorrection(ctx, pid, fact.ID())
			if err != nil {
				return err
			}
			result.Records = append(result.Records, Record{fact, direct})
		}
		return nil
	})
	if err != nil {
		return History{}, err
	}
	return result, nil
}
