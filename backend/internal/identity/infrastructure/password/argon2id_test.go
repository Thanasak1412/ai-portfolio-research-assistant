package password

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

func TestArgon2idHashAndVerify(t *testing.T) {
	adapter := New()
	ctx := context.Background()
	password := "correct horse battery staple"
	first, err := adapter.Hash(ctx, password)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Hash(ctx, password)
	if err != nil {
		t.Fatal(err)
	}
	if first.Encoded() == second.Encoded() {
		t.Fatal("random salts did not produce distinct hashes")
	}
	if !strings.HasPrefix(first.Encoded(), "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("unexpected PHC policy: %s", first)
	}
	result, err := adapter.Verify(ctx, password, first)
	if err != nil || !result.Verified || result.NeedsRehash {
		t.Fatalf("verify: result=%+v err=%v", result, err)
	}
	if _, err := adapter.Verify(ctx, "incorrect password value", first); !errors.Is(err, application.ErrCredentialRejected) {
		t.Fatalf("wrong password: %v", err)
	}
}

func TestArgon2idInputAndMalformedBounds(t *testing.T) {
	adapter := New()
	ctx := context.Background()
	if _, err := adapter.Hash(ctx, strings.Repeat("a", 11)); !errors.Is(err, application.ErrCredentialRejected) {
		t.Fatal("minimum was not enforced")
	}
	if _, err := adapter.Hash(ctx, strings.Repeat("a", MaximumBytes)); err != nil {
		t.Fatalf("maximum rejected: %v", err)
	}
	if _, err := adapter.Hash(ctx, strings.Repeat("a", MaximumBytes+1)); !errors.Is(err, application.ErrCredentialRejected) {
		t.Fatal("over maximum accepted")
	}
	for _, malformed := range []string{"", "$argon2i$v=19$m=65536,t=3,p=2$bad$bad", "$argon2id$v=20$m=65536,t=3,p=2$bad$bad", "$argon2id$v=19$m=999999,t=3,p=2$bad$bad", "$argon2id$v=19$m=65536,t=3,p=2$%%%$%%%"} {
		hash, _ := domain.NewPasswordHash(malformed)
		if _, err := adapter.Verify(ctx, "correct horse battery staple", hash); !errors.Is(err, application.ErrCredentialRejected) || (malformed != "" && strings.Contains(err.Error(), malformed)) {
			t.Fatalf("unsafe malformed result for %q: %v", malformed, err)
		}
	}
}

func TestArgon2idRehashEvaluationAndUnapprovedParameters(t *testing.T) {
	if needsRehash(phcParameters{memory: MemoryKiB, iterations: Iterations, parallelism: Parallelism}) {
		t.Fatal("current parameters require rehash")
	}
	if !needsRehash(phcParameters{memory: MemoryKiB, iterations: Iterations, parallelism: 1}) {
		t.Fatal("outdated parameters did not require rehash")
	}
	current, err := New().Hash(context.Background(), "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	unapproved := strings.Replace(current.Encoded(), "m=65536,t=3,p=2", "m=65536,t=3,p=1", 1)
	hash, _ := domain.NewPasswordHash(unapproved)
	if _, err := New().Verify(context.Background(), "correct horse battery staple", hash); !errors.Is(err, application.ErrCredentialRejected) {
		t.Fatalf("unapproved legacy parameters accepted: %v", err)
	}
}
