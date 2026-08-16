package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

func TestCreatePortfolioUsesOnlyTrustedPrincipalClockAndID(t *testing.T) {
	fixture := newPortfolioServiceFixture(t)
	if _, err := fixture.service.CreatePortfolio(context.Background(), identitydomain.Principal{}, CreatePortfolioInput{Name: "Growth", BaseCurrency: "USD"}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("unauthenticated create error=%v", err)
	}
	if fixture.repository.creates != 0 {
		t.Fatal("unauthenticated create reached repository")
	}
	created, err := fixture.service.CreatePortfolio(context.Background(), fixture.principal, CreatePortfolioInput{Name: " Growth ", BaseCurrency: "USD"})
	if err != nil {
		t.Fatal(err)
	}
	if created.OwnerID() != fixture.owner || created.Name().String() != "Growth" || !created.CreatedAt().Equal(fixture.clock.now) || created.ID() != fixture.ids.nextID {
		t.Fatalf("created=%+v", created)
	}
	if _, err := fixture.service.CreatePortfolio(context.Background(), fixture.principal, CreatePortfolioInput{Name: "Growth", BaseCurrency: "THB"}); !errors.Is(err, ErrInvalidPortfolioInput) {
		t.Fatalf("non-USD create error=%v", err)
	}
	fixture.repository.createErr = ErrPortfolioNameConflict
	if _, err := fixture.service.CreatePortfolio(context.Background(), fixture.principal, CreatePortfolioInput{Name: "Conflict", BaseCurrency: "USD"}); !errors.Is(err, ErrPortfolioNameConflict) {
		t.Fatalf("conflict propagation error=%v", err)
	}
}

func TestPrincipalScopedGetListUpdateAndArchive(t *testing.T) {
	fixture := newPortfolioServiceFixture(t)
	active := fixture.portfolio(t, "Growth", domain.PortfolioStatusActive, nil)
	archivedAt := fixture.clock.now.Add(-time.Hour)
	archived := fixture.portfolio(t, "Archived", domain.PortfolioStatusArchived, &archivedAt)
	fixture.repository.rows[active.ID().String()] = active
	fixture.repository.rows[archived.ID().String()] = archived

	list, err := fixture.service.ListPortfolios(context.Background(), fixture.principal, domain.PortfolioStatusActive)
	if err != nil || len(list) != 1 || list[0].ID() != active.ID() {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	if _, err := fixture.service.ListPortfolios(context.Background(), identitydomain.Principal{}, domain.PortfolioStatusActive); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("unauthenticated list error=%v", err)
	}
	if _, err := fixture.service.GetPortfolio(context.Background(), fixture.principal, active.ID()); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.GetPortfolio(context.Background(), fixture.principal, mustApplicationPortfolioID(t)); !errors.Is(err, ErrPortfolioNotFound) {
		t.Fatalf("missing get error=%v", err)
	}

	updated, err := fixture.service.UpdatePortfolio(context.Background(), fixture.principal, active.ID(), UpdatePortfolioInput{Name: " Updated "})
	if err != nil || updated.Name().String() != "Updated" || !updated.UpdatedAt().Equal(fixture.clock.now) {
		t.Fatalf("update=%+v err=%v", updated, err)
	}
	if _, err := fixture.service.UpdatePortfolio(context.Background(), fixture.principal, archived.ID(), UpdatePortfolioInput{Name: "No"}); !errors.Is(err, ErrPortfolioArchived) {
		t.Fatalf("archived update error=%v", err)
	}
	if _, err := fixture.service.UpdatePortfolio(context.Background(), fixture.principal, mustApplicationPortfolioID(t), UpdatePortfolioInput{Name: "No"}); !errors.Is(err, ErrPortfolioNotFound) {
		t.Fatalf("missing update error=%v", err)
	}

	result, err := fixture.service.ArchivePortfolio(context.Background(), fixture.principal, active.ID())
	if err != nil || !result.IsArchived() {
		t.Fatalf("archive=%+v err=%v", result, err)
	}
	retry, err := fixture.service.ArchivePortfolio(context.Background(), fixture.principal, active.ID())
	if err != nil || !retry.IsArchived() {
		t.Fatalf("archive retry=%+v err=%v", retry, err)
	}
	firstArchivedAt, _ := result.ArchivedAt()
	secondArchivedAt, _ := retry.ArchivedAt()
	if !firstArchivedAt.Equal(secondArchivedAt) {
		t.Fatalf("archive retry changed archivedAt: %v -> %v", firstArchivedAt, secondArchivedAt)
	}
}

type portfolioServiceFixture struct {
	service    *Service
	repository *fakePortfolioRepository
	clock      fixedPortfolioClock
	ids        fixedPortfolioIDs
	owner      identitydomain.UserID
	principal  identitydomain.Principal
}

func newPortfolioServiceFixture(t *testing.T) *portfolioServiceFixture {
	t.Helper()
	owner := mustApplicationOwner(t)
	principal, err := identitydomain.NewPrincipal(owner)
	if err != nil {
		t.Fatal(err)
	}
	ids := fixedPortfolioIDs{nextID: mustApplicationPortfolioID(t)}
	fixture := &portfolioServiceFixture{
		repository: &fakePortfolioRepository{rows: map[string]domain.Portfolio{}},
		clock:      fixedPortfolioClock{now: time.Date(2026, time.August, 16, 14, 0, 0, 0, time.UTC)},
		ids:        ids, owner: owner, principal: principal,
	}
	service, err := NewService(ServiceDependencies{Repository: fixture.repository, Clock: fixture.clock, IDs: fixture.ids})
	if err != nil {
		t.Fatal(err)
	}
	fixture.service = service
	return fixture
}

func (fixture *portfolioServiceFixture) portfolio(t *testing.T, name string, status domain.PortfolioStatus, archivedAt *time.Time) domain.Portfolio {
	t.Helper()
	value, err := domain.RehydratePortfolio(mustApplicationPortfolioID(t), fixture.owner, mustApplicationName(t, name), domain.BaseCurrencyUSD, status, archivedAt, fixture.clock.now.Add(-2*time.Hour), fixture.clock.now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

type fixedPortfolioClock struct{ now time.Time }

func (clock fixedPortfolioClock) Now() time.Time { return clock.now }

type fixedPortfolioIDs struct{ nextID domain.PortfolioID }

func (ids fixedPortfolioIDs) PortfolioID() (domain.PortfolioID, error) { return ids.nextID, nil }

type fakePortfolioRepository struct {
	rows      map[string]domain.Portfolio
	creates   int
	createErr error
}

func (repository *fakePortfolioRepository) Create(_ context.Context, portfolio domain.Portfolio) (domain.Portfolio, error) {
	repository.creates++
	if repository.createErr != nil {
		return domain.Portfolio{}, repository.createErr
	}
	repository.rows[portfolio.ID().String()] = portfolio
	return portfolio, nil
}
func (repository *fakePortfolioRepository) FindOwnedByID(_ context.Context, owner identitydomain.UserID, id domain.PortfolioID) (domain.Portfolio, error) {
	portfolio, ok := repository.rows[id.String()]
	if !ok || portfolio.OwnerID() != owner {
		return domain.Portfolio{}, ErrPortfolioNotFound
	}
	return portfolio, nil
}
func (repository *fakePortfolioRepository) ListOwnedByStatus(_ context.Context, owner identitydomain.UserID, status domain.PortfolioStatus) ([]domain.Portfolio, error) {
	result := []domain.Portfolio{}
	for _, portfolio := range repository.rows {
		if portfolio.OwnerID() == owner && portfolio.Status() == status {
			result = append(result, portfolio)
		}
	}
	return result, nil
}
func (repository *fakePortfolioRepository) UpdateOwnedActiveName(_ context.Context, owner identitydomain.UserID, id domain.PortfolioID, name domain.PortfolioName, updatedAt time.Time) (domain.Portfolio, error) {
	portfolio, err := repository.FindOwnedByID(context.Background(), owner, id)
	if err != nil || !portfolio.IsActive() {
		return domain.Portfolio{}, ErrPortfolioNotFound
	}
	updated, err := portfolio.Rename(name, updatedAt)
	if err != nil {
		return domain.Portfolio{}, err
	}
	repository.rows[id.String()] = updated
	return updated, nil
}
func (repository *fakePortfolioRepository) ArchiveOwnedActive(_ context.Context, owner identitydomain.UserID, id domain.PortfolioID, archivedAt time.Time) (domain.Portfolio, error) {
	portfolio, err := repository.FindOwnedByID(context.Background(), owner, id)
	if err != nil || !portfolio.IsActive() {
		return domain.Portfolio{}, ErrPortfolioNotFound
	}
	archived, err := portfolio.Archive(archivedAt)
	if err != nil {
		return domain.Portfolio{}, err
	}
	repository.rows[id.String()] = archived
	return archived, nil
}

func mustApplicationPortfolioID(t *testing.T) domain.PortfolioID {
	t.Helper()
	id, err := domain.NewPortfolioID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func mustApplicationOwner(t *testing.T) identitydomain.UserID {
	t.Helper()
	id, err := identitydomain.NewUserID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func mustApplicationName(t *testing.T, value string) domain.PortfolioName {
	t.Helper()
	name, err := domain.NewPortfolioName(value)
	if err != nil {
		t.Fatal(err)
	}
	return name
}
