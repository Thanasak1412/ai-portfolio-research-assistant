package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

func TestPortfolioNameUsesOnlyApprovedEdgeTrim(t *testing.T) {
	name, err := NewPortfolioName(" \tGrowth  Fund\n")
	if err != nil || name.String() != "Growth  Fund" {
		t.Fatalf("name=%q err=%v", name.String(), err)
	}
	punctuation, err := NewPortfolioName("Growth-Fund")
	if err != nil || punctuation.String() != "Growth-Fund" {
		t.Fatalf("punctuation=%q err=%v", punctuation.String(), err)
	}
	unicode, err := NewPortfolioName("\u00a0Growth\u00a0")
	if err != nil || unicode.String() != "\u00a0Growth\u00a0" {
		t.Fatalf("unicode whitespace was unexpectedly normalized: %q err=%v", unicode.String(), err)
	}
	for _, value := range []string{"", " \t\n", strings.Repeat("a", 201)} {
		if _, err := NewPortfolioName(value); !errors.Is(err, ErrInvalidPortfolioName) {
			t.Fatalf("value %q error=%v", value, err)
		}
	}
	if _, err := NewPortfolioName(strings.Repeat("é", 200)); err != nil {
		t.Fatalf("200 Unicode code points should be accepted: %v", err)
	}
}

func TestPortfolioConstructionLifecycleAndRehydration(t *testing.T) {
	now := time.Date(2026, time.August, 16, 12, 0, 0, 0, time.UTC)
	portfolio := portfolioFixture(t, now)
	if !portfolio.IsActive() || portfolio.IsArchived() {
		t.Fatalf("new Portfolio lifecycle = %q", portfolio.Status())
	}
	if _, ok := portfolio.ArchivedAt(); ok {
		t.Fatal("new Portfolio has archived timestamp")
	}
	renamed, err := portfolio.Rename(mustName(t, "Growth  Fund"), now.Add(time.Minute))
	if err != nil || renamed.Name().String() != "Growth  Fund" || renamed.ID() != portfolio.ID() || renamed.OwnerID() != portfolio.OwnerID() || renamed.BaseCurrency() != BaseCurrencyUSD {
		t.Fatalf("rename=%+v err=%v", renamed, err)
	}
	archived, err := renamed.Archive(now.Add(2 * time.Minute))
	if err != nil || !archived.IsArchived() {
		t.Fatalf("archive=%+v err=%v", archived, err)
	}
	archivedAt, ok := archived.ArchivedAt()
	if !ok || !archivedAt.Equal(now.Add(2*time.Minute)) || !archived.UpdatedAt().Equal(archivedAt) {
		t.Fatalf("archive timestamp=%v present=%v updated=%v", archivedAt, ok, archived.UpdatedAt())
	}
	retried, err := archived.Archive(now.Add(3 * time.Minute))
	if err != nil || !retried.UpdatedAt().Equal(archived.UpdatedAt()) {
		t.Fatalf("archive retry=%+v err=%v", retried, err)
	}
	if _, err := archived.Rename(mustName(t, "No"), now.Add(4*time.Minute)); !errors.Is(err, ErrPortfolioArchived) {
		t.Fatalf("archived rename error=%v", err)
	}

	owner := mustOwner(t)
	id := mustPortfolioID(t)
	name := mustName(t, "Rehydrated")
	if _, err := RehydratePortfolio(id, owner, name, BaseCurrencyUSD, PortfolioStatusActive, &now, now, now); !errors.Is(err, ErrInvalidPortfolio) {
		t.Fatalf("active with archive time error=%v", err)
	}
	if _, err := RehydratePortfolio(id, owner, name, BaseCurrencyUSD, PortfolioStatusArchived, nil, now, now); !errors.Is(err, ErrInvalidPortfolio) {
		t.Fatalf("archived without archive time error=%v", err)
	}
	if _, err := RehydratePortfolio(id, owner, name, BaseCurrencyUSD, PortfolioStatus("UNKNOWN"), nil, now, now); !errors.Is(err, ErrInvalidPortfolioStatus) {
		t.Fatalf("unknown status error=%v", err)
	}
	if _, err := RehydratePortfolio(id, owner, name, BaseCurrency("THB"), PortfolioStatusActive, nil, now, now); !errors.Is(err, ErrInvalidBaseCurrency) {
		t.Fatalf("non-USD error=%v", err)
	}
}

func TestPortfolioIDAndClosedValues(t *testing.T) {
	if _, err := NewPortfolioID(uuid.Nil); !errors.Is(err, ErrInvalidPortfolioID) {
		t.Fatalf("zero ID error=%v", err)
	}
	if _, err := ParsePortfolioID("not-a-uuid"); !errors.Is(err, ErrInvalidPortfolioID) {
		t.Fatalf("invalid ID error=%v", err)
	}
	if _, err := ParseBaseCurrency("USD"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseBaseCurrency("THB"); !errors.Is(err, ErrInvalidBaseCurrency) {
		t.Fatalf("currency error=%v", err)
	}
	for _, value := range []string{"ACTIVE", "ARCHIVED"} {
		if _, err := ParsePortfolioStatus(value); err != nil {
			t.Fatalf("status %q error=%v", value, err)
		}
	}
}

func portfolioFixture(t *testing.T, now time.Time) Portfolio {
	t.Helper()
	portfolio, err := NewPortfolio(mustPortfolioID(t), mustOwner(t), mustName(t, "Growth"), BaseCurrencyUSD, now)
	if err != nil {
		t.Fatal(err)
	}
	return portfolio
}

func mustPortfolioID(t *testing.T) PortfolioID {
	t.Helper()
	value, err := NewPortfolioID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mustOwner(t *testing.T) identitydomain.UserID {
	t.Helper()
	value, err := identitydomain.NewUserID(uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mustName(t *testing.T, value string) PortfolioName {
	t.Helper()
	name, err := NewPortfolioName(value)
	if err != nil {
		t.Fatal(err)
	}
	return name
}
