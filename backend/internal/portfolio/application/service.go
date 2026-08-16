package application

import (
	"context"
	"errors"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

type ServiceDependencies struct {
	Repository PortfolioRepository
	Clock      Clock
	IDs        IDGenerator
}

type Service struct{ dependencies ServiceDependencies }

func NewService(dependencies ServiceDependencies) (*Service, error) {
	if dependencies.Repository == nil || dependencies.Clock == nil || dependencies.IDs == nil {
		return nil, ErrPortfolioService
	}
	return &Service{dependencies: dependencies}, nil
}

type CreatePortfolioInput struct {
	Name         string
	BaseCurrency string
}

type UpdatePortfolioInput struct{ Name string }

func (service *Service) CreatePortfolio(ctx context.Context, principal identitydomain.Principal, input CreatePortfolioInput) (domain.Portfolio, error) {
	ownerID, err := ownerFromPrincipal(principal)
	if err != nil {
		return domain.Portfolio{}, err
	}
	name, err := domain.NewPortfolioName(input.Name)
	if err != nil {
		return domain.Portfolio{}, ErrInvalidPortfolioInput
	}
	currency, err := domain.ParseBaseCurrency(input.BaseCurrency)
	if err != nil {
		return domain.Portfolio{}, ErrInvalidPortfolioInput
	}
	id, err := service.dependencies.IDs.PortfolioID()
	if err != nil {
		return domain.Portfolio{}, ErrPortfolioService
	}
	portfolio, err := domain.NewPortfolio(id, ownerID, name, currency, service.dependencies.Clock.Now().UTC())
	if err != nil {
		return domain.Portfolio{}, ErrInvalidPortfolioInput
	}
	return service.dependencies.Repository.Create(ctx, portfolio)
}

func (service *Service) ListPortfolios(ctx context.Context, principal identitydomain.Principal, status domain.PortfolioStatus) ([]domain.Portfolio, error) {
	ownerID, err := ownerFromPrincipal(principal)
	if err != nil {
		return nil, err
	}
	if _, err := domain.ParsePortfolioStatus(string(status)); err != nil {
		return nil, ErrInvalidPortfolioInput
	}
	return service.dependencies.Repository.ListOwnedByStatus(ctx, ownerID, status)
}

func (service *Service) GetPortfolio(ctx context.Context, principal identitydomain.Principal, id domain.PortfolioID) (domain.Portfolio, error) {
	ownerID, err := ownerFromPrincipal(principal)
	if err != nil {
		return domain.Portfolio{}, err
	}
	if id.IsZero() {
		return domain.Portfolio{}, ErrInvalidPortfolioInput
	}
	return service.dependencies.Repository.FindOwnedByID(ctx, ownerID, id)
}

func (service *Service) UpdatePortfolio(ctx context.Context, principal identitydomain.Principal, id domain.PortfolioID, input UpdatePortfolioInput) (domain.Portfolio, error) {
	ownerID, err := ownerFromPrincipal(principal)
	if err != nil {
		return domain.Portfolio{}, err
	}
	if id.IsZero() {
		return domain.Portfolio{}, ErrInvalidPortfolioInput
	}
	name, err := domain.NewPortfolioName(input.Name)
	if err != nil {
		return domain.Portfolio{}, ErrInvalidPortfolioInput
	}
	updated, err := service.dependencies.Repository.UpdateOwnedActiveName(ctx, ownerID, id, name, service.dependencies.Clock.Now().UTC())
	if !errors.Is(err, ErrPortfolioNotFound) {
		return updated, err
	}
	existing, findErr := service.dependencies.Repository.FindOwnedByID(ctx, ownerID, id)
	if findErr != nil {
		return domain.Portfolio{}, findErr
	}
	if existing.IsArchived() {
		return domain.Portfolio{}, ErrPortfolioArchived
	}
	return domain.Portfolio{}, ErrPersistenceConflict
}

func (service *Service) ArchivePortfolio(ctx context.Context, principal identitydomain.Principal, id domain.PortfolioID) (domain.Portfolio, error) {
	ownerID, err := ownerFromPrincipal(principal)
	if err != nil {
		return domain.Portfolio{}, err
	}
	if id.IsZero() {
		return domain.Portfolio{}, ErrInvalidPortfolioInput
	}
	archived, err := service.dependencies.Repository.ArchiveOwnedActive(ctx, ownerID, id, service.dependencies.Clock.Now().UTC())
	if !errors.Is(err, ErrPortfolioNotFound) {
		return archived, err
	}
	existing, findErr := service.dependencies.Repository.FindOwnedByID(ctx, ownerID, id)
	if findErr != nil {
		return domain.Portfolio{}, findErr
	}
	if existing.IsArchived() {
		return existing, nil
	}
	return domain.Portfolio{}, ErrPersistenceConflict
}

func ownerFromPrincipal(principal identitydomain.Principal) (identitydomain.UserID, error) {
	ownerID, ok := principal.UserID()
	if !ok {
		return identitydomain.UserID{}, ErrUnauthenticated
	}
	return ownerID, nil
}
