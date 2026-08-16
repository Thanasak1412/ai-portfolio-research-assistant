//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

func TestPostgresPortfolioRepositoryOwnershipLifecycleAndErrorClassification(t *testing.T) {
	pool := openPortfolioTestPool(t)
	repository := NewPostgresPortfolioRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	owner := repositoryOwner(t, insertPortfolioTestUser(t, pool, now))
	otherOwner := repositoryOwner(t, insertPortfolioTestUser(t, pool, now))

	created := repositoryPortfolio(t, owner, " Growth ", now)
	created, err := repository.Create(ctx, created)
	if err != nil || created.Name().String() != "Growth" || created.BaseCurrency() != domain.BaseCurrencyUSD || !created.IsActive() {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	if _, err := repository.Create(ctx, repositoryPortfolio(t, owner, "growth", now)); !errors.Is(err, application.ErrPortfolioNameConflict) {
		t.Fatalf("case-equivalent create error=%v", err)
	}
	if _, err := repository.Create(ctx, repositoryPortfolio(t, otherOwner, "growth", now)); err != nil {
		t.Fatalf("other owner same name error=%v", err)
	}

	got, err := repository.FindOwnedByID(ctx, owner, created.ID())
	if err != nil || got.ID() != created.ID() {
		t.Fatalf("owned get=%+v err=%v", got, err)
	}
	if _, err := repository.FindOwnedByID(ctx, otherOwner, created.ID()); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("cross-owner get error=%v", err)
	}
	if _, err := repository.FindOwnedByID(ctx, owner, repositoryID(t)); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("missing get error=%v", err)
	}

	renamed, err := repository.UpdateOwnedActiveName(ctx, owner, created.ID(), repositoryName(t, "Updated"), now.Add(time.Minute))
	if err != nil || renamed.Name().String() != "Updated" || renamed.ID() != created.ID() || renamed.OwnerID() != owner {
		t.Fatalf("rename=%+v err=%v", renamed, err)
	}
	conflict := repositoryPortfolio(t, owner, "Conflict", now)
	if _, err := repository.Create(ctx, conflict); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.UpdateOwnedActiveName(ctx, owner, renamed.ID(), repositoryName(t, "conflict"), now.Add(2*time.Minute)); !errors.Is(err, application.ErrPortfolioNameConflict) {
		t.Fatalf("rename conflict error=%v", err)
	}

	archived, err := repository.ArchiveOwnedActive(ctx, owner, renamed.ID(), now.Add(3*time.Minute))
	if err != nil || !archived.IsArchived() {
		t.Fatalf("archive=%+v err=%v", archived, err)
	}
	archivedAt, _ := archived.ArchivedAt()
	if _, err := repository.UpdateOwnedActiveName(ctx, owner, archived.ID(), repositoryName(t, "No"), now.Add(4*time.Minute)); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("archived direct update error=%v", err)
	}
	if _, err := repository.ArchiveOwnedActive(ctx, owner, archived.ID(), now.Add(4*time.Minute)); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("archived direct archive error=%v", err)
	}
	stored, err := repository.FindOwnedByID(ctx, owner, archived.ID())
	storedArchivedAt, _ := stored.ArchivedAt()
	if err != nil || !storedArchivedAt.Equal(archivedAt) {
		t.Fatalf("archive retry changed stored timestamp: %v %v", stored, err)
	}
	if _, err := repository.Create(ctx, repositoryPortfolio(t, owner, "updated", now.Add(5*time.Minute))); err != nil {
		t.Fatalf("archived name was not reusable: %v", err)
	}

	if _, err := repository.UpdateOwnedActiveName(ctx, otherOwner, conflict.ID(), repositoryName(t, "No"), now); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("cross-owner update error=%v", err)
	}
	if _, err := repository.ArchiveOwnedActive(ctx, otherOwner, conflict.ID(), now); !errors.Is(err, application.ErrPortfolioNotFound) {
		t.Fatalf("cross-owner archive error=%v", err)
	}
}

func TestPostgresPortfolioRepositoryListPreservesOwnerStatusAndOrder(t *testing.T) {
	pool := openPortfolioTestPool(t)
	repository := NewPostgresPortfolioRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	owner := repositoryOwner(t, insertPortfolioTestUser(t, pool, now))
	otherOwner := repositoryOwner(t, insertPortfolioTestUser(t, pool, now))
	first, err := repository.Create(ctx, repositoryPortfolio(t, owner, "First", now))
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.Create(ctx, repositoryPortfolio(t, owner, "Second", now.Add(time.Second)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Create(ctx, repositoryPortfolio(t, otherOwner, "Other", now.Add(2*time.Second))); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.UpdateOwnedActiveName(ctx, owner, first.ID(), repositoryName(t, "First"), now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	active, err := repository.ListOwnedByStatus(ctx, owner, domain.PortfolioStatusActive)
	if err != nil || len(active) != 2 || active[0].ID() != first.ID() || active[1].ID() != second.ID() {
		t.Fatalf("owner/status ordering rows=%+v err=%v", active, err)
	}
	if _, err := repository.ArchiveOwnedActive(ctx, owner, second.ID(), now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	archived, err := repository.ListOwnedByStatus(ctx, owner, domain.PortfolioStatusArchived)
	if err != nil || len(archived) != 1 || archived[0].ID() != second.ID() {
		t.Fatalf("archived rows=%+v err=%v", archived, err)
	}
}

func repositoryPortfolio(t *testing.T, owner identitydomain.UserID, name string, createdAt time.Time) domain.Portfolio {
	t.Helper()
	portfolio, err := domain.NewPortfolio(repositoryID(t), owner, repositoryName(t, name), domain.BaseCurrencyUSD, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	return portfolio
}

func repositoryOwner(t *testing.T, value pgtype.UUID) identitydomain.UserID {
	t.Helper()
	owner, err := identitydomain.NewUserID(uuid.UUID(value.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	return owner
}

func repositoryID(t *testing.T) domain.PortfolioID {
	t.Helper()
	id, err := domain.NewPortfolioID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func repositoryName(t *testing.T, value string) domain.PortfolioName {
	t.Helper()
	name, err := domain.NewPortfolioName(value)
	if err != nil {
		t.Fatal(err)
	}
	return name
}
