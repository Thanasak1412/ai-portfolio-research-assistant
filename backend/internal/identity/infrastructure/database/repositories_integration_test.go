//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

func TestPostgresUserRepositoryMapsDomainAndPersistenceErrors(t *testing.T) {
	pool := openTestPool(t)
	repository := NewPostgresUserRepository(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := repositoryUserFixture(t, now)

	created, err := repository.Create(ctx, user)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	byEmail, err := repository.FindByNormalizedEmail(ctx, user.Email())
	if err != nil || byEmail.ID() != created.ID() {
		t.Fatalf("find by email: user=%v err=%v", byEmail, err)
	}
	byID, err := repository.FindByID(ctx, user.ID())
	if err != nil || byID.ID() != created.ID() {
		t.Fatalf("find by ID: user=%v err=%v", byID, err)
	}

	missingID := mustDomainUserID(t, uuid.New())
	if _, err := repository.FindByID(ctx, missingID); !errors.Is(err, application.ErrUserNotFound) {
		t.Fatalf("expected user-not-found classification, got %v", err)
	}
	duplicateID := mustDomainUserID(t, uuid.New())
	duplicate, err := domain.NewActiveUser(duplicateID, user.Email(), user.PasswordHash(), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Create(ctx, duplicate); !errors.Is(err, application.ErrDuplicateIdentity) {
		t.Fatalf("expected duplicate-identity classification, got %v", err)
	}

	replacementHash, _ := domain.NewPasswordHash(encodedHash("repository-rehash"))
	updated, err := repository.CompareAndSwapPasswordHash(ctx, user.ID(), user.PasswordHash(), replacementHash, now.Add(time.Second))
	if err != nil || updated.PasswordHash().Encoded() != replacementHash.Encoded() {
		t.Fatalf("compare-and-swap: user=%v err=%v", updated, err)
	}
	if _, err := repository.CompareAndSwapPasswordHash(ctx, user.ID(), user.PasswordHash(), replacementHash, now.Add(2*time.Second)); !errors.Is(err, application.ErrPersistenceConflict) {
		t.Fatalf("expected stale compare-and-swap conflict, got %v", err)
	}

	disabledAt := now.Add(3 * time.Second)
	if _, err := pool.Exec(ctx, `
		UPDATE users
		SET account_status = 'disabled', disabled_at = $2, updated_at = $2
		WHERE user_id = $1`, pgUserID(user.ID()), disabledAt); err != nil {
		t.Fatalf("disable fixture user: %v", err)
	}
	disabled, err := repository.FindByID(ctx, user.ID())
	if err != nil || !disabled.IsDisabled() {
		t.Fatalf("map disabled user: user=%v err=%v", disabled, err)
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repository.FindByID(canceled, user.ID()); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestPostgresRefreshRepositoryRotatesAndRevokesWithinTransaction(t *testing.T) {
	pool := openTestPool(t)
	userRepository := NewPostgresUserRepository(pool)
	sessionRepository := NewPostgresRefreshSessionRepository(pool)
	transactor := NewPostgresTransactor(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := repositoryUserFixture(t, now)
	if _, err := userRepository.Create(ctx, user); err != nil {
		t.Fatalf("create owner: %v", err)
	}
	original := repositorySessionFixture(t, user.ID(), now, 1)
	if _, err := sessionRepository.CreateInitial(ctx, original); err != nil {
		t.Fatalf("create initial session: %v", err)
	}

	var replacement domain.RefreshSession
	err := transactor.WithinTransaction(ctx, func(txContext context.Context, repositories application.TransactionRepositories) error {
		locked, lockErr := repositories.RefreshSessions().LockByDigest(txContext, original.TokenDigest())
		if lockErr != nil {
			return lockErr
		}
		replaced, plannedReplacement, planErr := locked.PlanReplacement(domain.ReplacementInput{
			SessionID: mustDomainSessionID(t, uuid.New()), TokenDigest: repositoryDigest(t, 2),
			CreatedAt: now.Add(time.Minute), IdleExpiresAt: now.Add(30 * 24 * time.Hour),
			NetworkIdentityHash: "network-hash", UserAgent: "integration-browser",
		})
		if planErr != nil {
			return planErr
		}
		if _, replaceErr := repositories.RefreshSessions().MarkReplaced(txContext, replaced); replaceErr != nil {
			return replaceErr
		}
		inserted, insertErr := repositories.RefreshSessions().InsertReplacement(txContext, plannedReplacement)
		if insertErr != nil {
			return insertErr
		}
		replacement = inserted
		return nil
	})
	if err != nil {
		t.Fatalf("rotate through repository transaction: %v", err)
	}
	family, err := sessionRepository.ListFamily(ctx, original.FamilyID())
	if err != nil || len(family) != 2 {
		t.Fatalf("list rotated family: count=%d err=%v", len(family), err)
	}
	if replacement.FamilyID() != original.FamilyID() || replacement.UserID() != original.UserID() || !replacement.AbsoluteExpiresAt().Equal(original.AbsoluteExpiresAt()) {
		t.Fatal("repository rotation changed family invariants")
	}

	unrelated := repositorySessionFixture(t, user.ID(), now.Add(time.Second), 3)
	if _, err := sessionRepository.CreateInitial(ctx, unrelated); err != nil {
		t.Fatalf("create unrelated family: %v", err)
	}
	err = transactor.WithinTransaction(ctx, func(txContext context.Context, repositories application.TransactionRepositories) error {
		revoked, revokeErr := repositories.RefreshSessions().RevokeFamily(txContext, original.FamilyID(), now.Add(2*time.Minute), "logout")
		if revokeErr != nil {
			return revokeErr
		}
		if len(revoked) != 2 {
			t.Fatalf("expected complete current family revocation, got %d rows", len(revoked))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("revoke current session family: %v", err)
	}
	unrelatedAfter, err := sessionRepository.FindByID(ctx, unrelated.ID())
	if err != nil || unrelatedAfter.State() != domain.RefreshSessionStateActive {
		t.Fatalf("unrelated family changed: state=%q err=%v", unrelatedAfter.State(), err)
	}
}

func TestPostgresRefreshRepositoryRollbackAndConflict(t *testing.T) {
	pool := openTestPool(t)
	userRepository := NewPostgresUserRepository(pool)
	sessionRepository := NewPostgresRefreshSessionRepository(pool)
	transactor := NewPostgresTransactor(pool)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := repositoryUserFixture(t, now)
	if _, err := userRepository.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	original := repositorySessionFixture(t, user.ID(), now, 10)
	if _, err := sessionRepository.CreateInitial(ctx, original); err != nil {
		t.Fatal(err)
	}
	conflicting, err := domain.NewActiveRefreshSession(
		mustDomainSessionID(t, uuid.New()), original.FamilyID(), user.ID(), repositoryDigest(t, 11),
		now.Add(time.Second), now.Add(24*time.Hour), original.AbsoluteExpiresAt(), "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sessionRepository.CreateInitial(ctx, conflicting); !errors.Is(err, application.ErrPersistenceConflict) {
		t.Fatalf("expected active-family persistence conflict, got %v", err)
	}

	rollbackCause := errors.New("force rollback")
	err = transactor.WithinTransaction(ctx, func(txContext context.Context, repositories application.TransactionRepositories) error {
		locked, lockErr := repositories.RefreshSessions().LockByDigest(txContext, original.TokenDigest())
		if lockErr != nil {
			return lockErr
		}
		replaced, replacement, planErr := locked.PlanReplacement(domain.ReplacementInput{
			SessionID: mustDomainSessionID(t, uuid.New()), TokenDigest: repositoryDigest(t, 12),
			CreatedAt: now.Add(time.Minute), IdleExpiresAt: now.Add(24 * time.Hour),
		})
		if planErr != nil {
			return planErr
		}
		if _, replaceErr := repositories.RefreshSessions().MarkReplaced(txContext, replaced); replaceErr != nil {
			return replaceErr
		}
		if _, insertErr := repositories.RefreshSessions().InsertReplacement(txContext, replacement); insertErr != nil {
			return insertErr
		}
		return rollbackCause
	})
	if !errors.Is(err, rollbackCause) {
		t.Fatalf("expected callback error after rollback, got %v", err)
	}
	after, err := sessionRepository.FindByID(ctx, original.ID())
	if err != nil || after.State() != domain.RefreshSessionStateActive {
		t.Fatalf("rollback changed original: state=%q err=%v", after.State(), err)
	}
}

func TestPostgresTransactorSerializesRepositoryLocksAcrossConnections(t *testing.T) {
	poolOne := openTestPool(t)
	poolTwo := openTestPool(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := repositoryUserFixture(t, now)
	if _, err := NewPostgresUserRepository(poolOne).Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	session := repositorySessionFixture(t, user.ID(), now, 20)
	if _, err := NewPostgresRefreshSessionRepository(poolOne).CreateInitial(ctx, session); err != nil {
		t.Fatal(err)
	}
	replacementID := mustDomainSessionID(t, uuid.New())
	replacementDigest := repositoryDigest(t, 21)

	locked := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- NewPostgresTransactor(poolOne).WithinTransaction(ctx, func(txContext context.Context, repositories application.TransactionRepositories) error {
			current, err := repositories.RefreshSessions().LockByDigest(txContext, session.TokenDigest())
			if err != nil {
				return err
			}
			close(locked)
			<-release
			replaced, replacement, err := current.PlanReplacement(domain.ReplacementInput{
				SessionID: replacementID, TokenDigest: replacementDigest,
				CreatedAt: now.Add(time.Minute), IdleExpiresAt: now.Add(24 * time.Hour),
			})
			if err != nil {
				return err
			}
			if _, err := repositories.RefreshSessions().MarkReplaced(txContext, replaced); err != nil {
				return err
			}
			_, err = repositories.RefreshSessions().InsertReplacement(txContext, replacement)
			return err
		})
	}()
	<-locked

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- NewPostgresTransactor(poolTwo).WithinTransaction(ctx, func(txContext context.Context, repositories application.TransactionRepositories) error {
			current, err := repositories.RefreshSessions().LockByDigest(txContext, session.TokenDigest())
			if err != nil {
				return err
			}
			if err := current.ReplacementEligibility(now.Add(2 * time.Minute)); !errors.Is(err, domain.ErrSessionReplaced) {
				return errors.New("competing transaction observed generation as independently active")
			}
			return nil
		})
	}()
	select {
	case err := <-secondDone:
		t.Fatalf("second repository lock completed before release: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first transaction: %v", err)
	}
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second transaction after release: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second repository lock did not resume")
	}
	family, err := NewPostgresRefreshSessionRepository(poolOne).ListFamily(ctx, session.FamilyID())
	if err != nil || len(family) != 2 {
		t.Fatalf("final serialized family state: count=%d err=%v", len(family), err)
	}
	active := 0
	for _, generation := range family {
		if generation.State() == domain.RefreshSessionStateActive {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("expected exactly one active generation, got %d", active)
	}
}

func repositoryUserFixture(t *testing.T, now time.Time) domain.User {
	t.Helper()
	id := mustDomainUserID(t, uuid.New())
	email, err := domain.NormalizeEmail(uniqueEmail("repository"))
	if err != nil {
		t.Fatal(err)
	}
	hash, err := domain.NewPasswordHash(encodedHash("repository"))
	if err != nil {
		t.Fatal(err)
	}
	user, err := domain.NewActiveUser(id, email, hash, now)
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func repositorySessionFixture(t *testing.T, userID domain.UserID, now time.Time, seed byte) domain.RefreshSession {
	t.Helper()
	session, err := domain.NewActiveRefreshSession(
		mustDomainSessionID(t, uuid.New()), mustDomainFamilyID(t, uuid.New()), userID, repositoryDigest(t, seed),
		now, now.Add(30*24*time.Hour), now.Add(90*24*time.Hour), "network-hash", "integration-browser",
	)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func mustDomainUserID(t *testing.T, value uuid.UUID) domain.UserID {
	t.Helper()
	id, err := domain.NewUserID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustDomainSessionID(t *testing.T, value uuid.UUID) domain.SessionID {
	t.Helper()
	id, err := domain.NewSessionID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustDomainFamilyID(t *testing.T, value uuid.UUID) domain.TokenFamilyID {
	t.Helper()
	id, err := domain.NewTokenFamilyID(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func repositoryDigest(t *testing.T, seed byte) domain.TokenDigest {
	t.Helper()
	value := make([]byte, domain.TokenDigestLength)
	randomPart := uuid.New()
	copy(value, randomPart[:])
	value[len(value)-1] = seed
	digest, err := domain.NewTokenDigest(value)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
