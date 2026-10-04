package application

import (
	"errors"
	"testing"
	"time"

	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/google/uuid"
)

func TestSemanticFingerprint(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	id, _ := asset.NewAssetID(uuid.New())
	quantity, _ := domain.ParsePositiveDecimal("1.00")
	price, _ := domain.ParsePositiveDecimal("10.000")
	input := CommandInput{Kind: domain.KindBuy, AssetID: id, Quantity: quantity, UnitPrice: price, Currency: domain.CurrencyUSD, EffectiveAt: now}
	a, err := normalize(input, now)
	if err != nil {
		t.Fatal(err)
	}
	input.Quantity, _ = domain.ParsePositiveDecimal("1")
	input.Fee, _ = domain.ParseNonNegativeDecimal("0.0")
	input.EffectiveAt = now.In(time.FixedZone("offset", 3600))
	b, err := normalize(input, now)
	if err != nil {
		t.Fatal(err)
	}
	base := fingerprint(CreateScope, domain.TransactionID{}, a)
	if base != fingerprint(CreateScope, domain.TransactionID{}, b) {
		t.Fatal("canonical decimal, missing fee, and timezone equivalence")
	}
	for _, change := range []func(*CommandInput){
		func(i *CommandInput) { i.Note, _ = domain.NewNote("") },
		func(i *CommandInput) { i.ExternalReference, _ = domain.NewExternalReference("") },
		func(i *CommandInput) { i.Note, _ = domain.NewNote(" x ") },
		func(i *CommandInput) { i.Quantity, _ = domain.ParsePositiveDecimal("2") },
		func(i *CommandInput) { i.EffectiveAt = i.EffectiveAt.Add(time.Microsecond) },
	} {
		c := a
		change(&c)
		if base == fingerprint(CreateScope, domain.TransactionID{}, c) {
			t.Fatal("distinct semantic command collided")
		}
	}
	if base == fingerprint(CorrectScope, domain.TransactionID{}, a) {
		t.Fatal("scope omitted")
	}
	target, _ := domain.NewTransactionID(uuid.New())
	if fingerprint(CorrectScope, target, a) == fingerprint(CorrectScope, domain.TransactionID{}, a) {
		t.Fatal("target omitted")
	}
	c := a
	c.Note, _ = domain.NewNote("é")
	d := a
	d.Note, _ = domain.NewNote("e\u0301")
	if fingerprint(CreateScope, domain.TransactionID{}, c) == fingerprint(CreateScope, domain.TransactionID{}, d) {
		t.Fatal("Unicode normalized unexpectedly")
	}
}

func TestCommandValidation(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	amount, _ := domain.ParsePositiveDecimal("1")
	base := CommandInput{Kind: domain.KindDeposit, Currency: domain.CurrencyUSD, Amount: amount, EffectiveAt: now}
	for _, test := range []struct {
		name   string
		change func(*CommandInput)
		want   error
	}{
		{"internal reversal", func(i *CommandInput) { i.Kind = domain.KindReversal }, domain.ErrInvalidTransactionKind},
		{"currency", func(i *CommandInput) { i.Currency = "EUR" }, domain.ErrInvalidCurrency},
		{"future", func(i *CommandInput) { i.EffectiveAt = now.Add(time.Microsecond) }, domain.ErrInvalidEffectiveAt},
		{"submicrosecond", func(i *CommandInput) { i.EffectiveAt = now.Add(-time.Nanosecond) }, domain.ErrInvalidEffectiveAt},
		{"forbidden quantity", func(i *CommandInput) { i.Quantity = amount }, domain.ErrInvalidTransactionFields},
		{"missing amount", func(i *CommandInput) { i.Amount = domain.Decimal{} }, domain.ErrInvalidTransactionFields},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := base
			test.change(&input)
			if _, err := normalize(input, now); !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestIdempotencyKeyGrammar(t *testing.T) {
	for _, value := range []string{"0123456789abcdef", "A0123456789._~-zz"} {
		if !keyPattern.MatchString(value) {
			t.Fatal("valid key rejected")
		}
	}
	for _, value := range []string{"short", " 0123456789abcdef", "0123456789abcdef ", "_0123456789abcdef", "0123456789abcde/", "0123456789abcdeé"} {
		if keyPattern.MatchString(value) {
			t.Fatal("invalid key accepted")
		}
	}
}
