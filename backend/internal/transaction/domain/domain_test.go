package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"

	assetdomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	portfoliodomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
	"github.com/google/uuid"
)

var testNow = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

func positive(t *testing.T, value string) Decimal {
	t.Helper()
	d, err := ParsePositiveDecimal(value)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func nonnegative(t *testing.T, value string) Decimal {
	t.Helper()
	d, err := ParseNonNegativeDecimal(value)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func portfolioID(t *testing.T) portfoliodomain.PortfolioID {
	t.Helper()
	id, err := portfoliodomain.NewPortfolioID(uuid.MustParse("11111111-1111-4111-8111-111111111111"))
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func actorID(t *testing.T) identitydomain.UserID {
	t.Helper()
	id, err := identitydomain.NewUserID(uuid.MustParse("22222222-2222-4222-8222-222222222222"))
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func transactionID(t *testing.T, suffix byte) TransactionID {
	t.Helper()
	raw := uuid.MustParse("33333333-3333-4333-8333-333333333300")
	raw[15] = suffix
	id, err := NewTransactionID(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func correctionID(t *testing.T, suffix byte) CorrectionID {
	t.Helper()
	raw := uuid.MustParse("44444444-4444-4444-8444-444444444400")
	raw[15] = suffix
	id, err := NewCorrectionID(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func assetSnapshot(t *testing.T, suffix byte) AssetSnapshot {
	t.Helper()
	raw := uuid.MustParse("55555555-5555-4555-8555-555555555500")
	raw[15] = suffix
	id, err := assetdomain.NewAssetID(raw)
	if err != nil {
		t.Fatal(err)
	}
	asset, err := NewAssetSnapshot(id, assetdomain.AssetTypeEquity, "NYSE", assetdomain.CurrencyUSD)
	if err != nil {
		t.Fatal(err)
	}
	return asset
}
func context(at time.Time) CommandContext {
	return CommandContext{Currency: CurrencyUSD, EffectiveAt: at, Now: testNow}
}
func buy(t *testing.T, asset AssetSnapshot, qty string, at time.Time) Command {
	t.Helper()
	c, err := NewBuyCommand(asset, positive(t, qty), positive(t, "10"), nonnegative(t, "0"), context(at))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func sell(t *testing.T, asset AssetSnapshot, qty string, at time.Time) Command {
	t.Helper()
	c, err := NewSellCommand(asset, positive(t, qty), positive(t, "10"), nonnegative(t, "0"), context(at))
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func fact(t *testing.T, id byte, seq int64, command Command) Transaction {
	t.Helper()
	result, err := NewTransactionFromCommand(transactionID(t, id), portfolioID(t), actorID(t), seq, testNow, command)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestDecimalGrammarCanonicalAndExactArithmetic(t *testing.T) {
	for _, value := range []string{"1", "1.0", "1.000000000000"} {
		if got := positive(t, value).String(); got != "1" {
			t.Fatalf("%q canonical = %q", value, got)
		}
	}
	for _, value := range []string{"0", "0.0", "0.000"} {
		if got := nonnegative(t, value).String(); got != "0" {
			t.Fatalf("%q canonical = %q", value, got)
		}
	}
	for _, value := range []string{"", "0", "-1", "+1", "01", "1.", ".1", "1e3", "NaN", "Infinity", "0.0000000000001", "0.000000000000"} {
		if _, err := ParsePositiveDecimal(value); !errors.Is(err, ErrInvalidDecimal) {
			t.Errorf("accepted positive %q", value)
		}
	}
	for _, value := range []string{"-0", "+0", "01", "1e3", "0.0000000000001"} {
		if _, err := ParseNonNegativeDecimal(value); !errors.Is(err, ErrInvalidDecimal) {
			t.Errorf("accepted nonnegative %q", value)
		}
	}
	a := positive(t, "1234567890123456789012345678901234567890.123456789012")
	b := positive(t, "0.000000000001")
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if sum.String() != "1234567890123456789012345678901234567890.123456789013" {
		t.Fatal(sum)
	}
	back, err := sum.Sub(b)
	if err != nil || back.String() != a.String() {
		t.Fatalf("sub = %v, %v", back, err)
	}
	cmp, err := a.Compare(back)
	if err != nil || cmp != 0 {
		t.Fatalf("compare = %d, %v", cmp, err)
	}
	if a.String() != "1234567890123456789012345678901234567890.123456789012" || b.String() != "0.000000000001" {
		t.Fatal("operand mutated")
	}
	negative, err := b.Sub(a)
	if err != nil || negative.IsNonNegative() {
		t.Fatal("negative arithmetic invalid")
	}
}

func TestIdentifiersKindsAndAssetEligibility(t *testing.T) {
	if _, err := ParseTransactionID("not-a-uuid"); !errors.Is(err, ErrInvalidTransactionID) {
		t.Fatal(err)
	}
	if _, err := ParseCorrectionID(uuid.Nil.String()); !errors.Is(err, ErrInvalidCorrectionID) {
		t.Fatal(err)
	}
	if _, err := ParseTransactionID("33333333-3333-4333-8333-3333333333AA"); !errors.Is(err, ErrInvalidTransactionID) {
		t.Fatal("noncanonical UUID accepted")
	}
	if transactionID(t, 1).IsZero() || correctionID(t, 1).IsZero() {
		t.Fatal("nonzero ID rejected")
	}
	for _, kind := range []Kind{KindBuy, KindSell, KindDividend, KindDeposit, KindWithdrawal, KindFee} {
		if !kind.IsPublicCreatable() {
			t.Fatal(kind)
		}
	}
	if KindReversal.IsPublicCreatable() {
		t.Fatal("reversal public")
	}
	if _, err := ParseKind("ADJUSTMENT"); !errors.Is(err, ErrInvalidTransactionKind) {
		t.Fatal(err)
	}
	asset := assetSnapshot(t, 1)
	if _, err := NewAssetSnapshot(asset.ID(), assetdomain.AssetTypeETF, "NASDAQ", assetdomain.CurrencyUSD); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []struct {
		kind     assetdomain.AssetType
		exchange string
		currency assetdomain.Currency
	}{{assetdomain.AssetTypeCrypto, "CRYPTO", assetdomain.CurrencyUSD}, {assetdomain.AssetTypeEquity, "LSE", assetdomain.CurrencyUSD}, {assetdomain.AssetTypeEquity, "NYSE", "EUR"}} {
		if _, err := NewAssetSnapshot(asset.ID(), bad.kind, bad.exchange, bad.currency); !errors.Is(err, ErrAssetFinanciallyIneligible) {
			t.Fatal("ineligible asset accepted")
		}
	}
}

func TestCommandsTimeTextAndFieldMatrix(t *testing.T) {
	asset := assetSnapshot(t, 1)
	note, err := NewNote(" value ")
	if err != nil {
		t.Fatal(err)
	}
	reference, err := NewExternalReference("")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context(testNow.Add(-time.Microsecond))
	ctx.Note, ctx.ExternalReference = note, reference
	trade, err := NewBuyCommand(asset, positive(t, "1"), positive(t, "2"), nonnegative(t, "0"), ctx)
	if err != nil || !trade.IsValid() {
		t.Fatal(err)
	}
	if got, present := trade.Note(); !present || got != " value " {
		t.Fatal("note changed")
	}
	if got, present := trade.ExternalReference(); !present || got != "" {
		t.Fatal("empty ref lost")
	}
	if _, present := buy(t, asset, "1", testNow).Note(); present {
		t.Fatal("absent collapsed into empty")
	}
	for _, constructor := range []func(Decimal, CommandContext) (Command, error){NewDepositCommand, NewWithdrawalCommand, NewFeeCommand} {
		c, err := constructor(positive(t, "1"), context(testNow))
		if err != nil || !c.IsValid() {
			t.Fatal(err)
		}
		if _, ok := c.Asset(); ok {
			t.Fatal("cash has asset")
		}
	}
	dividend, err := NewDividendCommand(asset, positive(t, "1"), context(testNow))
	if err != nil || !dividend.IsValid() {
		t.Fatal(err)
	}
	if _, ok := dividend.Quantity(); ok {
		t.Fatal("dividend has quantity")
	}
	if _, err := NewSellCommand(asset, nonnegative(t, "0"), positive(t, "1"), nonnegative(t, "0"), ctx); !errors.Is(err, ErrInvalidTransactionFields) {
		t.Fatal(err)
	}
	if _, err := NewBuyCommand(asset, positive(t, "1"), nonnegative(t, "0"), nonnegative(t, "0"), ctx); !errors.Is(err, ErrInvalidTransactionFields) {
		t.Fatal(err)
	}
	badCurrency := context(testNow)
	badCurrency.Currency = "EUR"
	if _, err := NewDepositCommand(positive(t, "1"), badCurrency); !errors.Is(err, ErrInvalidCurrency) {
		t.Fatal(err)
	}
	for _, value := range []time.Time{time.Time{}, testNow.Add(time.Microsecond), testNow.Add(-time.Nanosecond)} {
		if _, err := ValidateEffectiveAt(value, testNow); !errors.Is(err, ErrInvalidEffectiveAt) {
			t.Fatalf("accepted %v", value)
		}
	}
	if at, err := ValidateEffectiveAt(testNow, testNow); err != nil || !at.Equal(testNow) {
		t.Fatal(err)
	}
	if _, err := NewNote(string(make([]rune, 2001))); !errors.Is(err, ErrInvalidTransactionFields) {
		t.Fatal("overlong note")
	}
	if _, err := NewExternalReference(string(make([]rune, 257))); !errors.Is(err, ErrInvalidTransactionFields) {
		t.Fatal("overlong reference")
	}
	decomposed, _ := NewNote("e\u0301")
	composed, _ := NewNote("é")
	if a, _ := decomposed.Value(); a == "é" {
		t.Fatal("normalized text")
	}
	if a, _ := composed.Value(); a != "é" {
		t.Fatal("text changed")
	}
}

func TestCorrectionFactsAndChain(t *testing.T) {
	asset := assetSnapshot(t, 1)
	before := fact(t, 1, 1, buy(t, asset, "10", testNow.Add(-time.Hour)))
	stateBefore := before.state
	input := CorrectionInput{Original: before, Replacement: sell(t, asset, "2", testNow), CorrectionID: correctionID(t, 1), ReversalID: transactionID(t, 2), ReplacementID: transactionID(t, 3), Actor: actorID(t), ReversalSequence: 2, ReplacementSequence: 3, CreatedAt: testNow}
	facts, err := BuildCorrection(input)
	if err != nil {
		t.Fatal(err)
	}
	if facts.Reversal.Kind() != KindReversal || facts.Reversal.EffectiveAt() != before.EffectiveAt() || facts.Reversal.ReversalOf() != before.ID() || facts.Replacement.CorrectionOf() != before.ID() || facts.Replacement.OriginatingCorrection() != input.CorrectionID {
		t.Fatal("incorrect links")
	}
	if _, present := facts.Reversal.Note(); present {
		t.Fatal("reversal retained note")
	}
	if !reflect.DeepEqual(before.state, stateBefore) {
		t.Fatal("original mutated")
	}
	if err := ValidateCorrectionTarget(facts.Replacement, false); err != nil {
		t.Fatal("replacement cannot be corrected")
	}
	if err := ValidateCorrectionTarget(facts.Reversal, false); !errors.Is(err, ErrTransactionNotCorrectable) {
		t.Fatal(err)
	}
	if err := ValidateCorrectionTarget(before, true); !errors.Is(err, ErrTransactionAlreadyCorrected) {
		t.Fatal(err)
	}
	chainInput := input
	chainInput.Original = facts.Replacement
	chainInput.CorrectionID = correctionID(t, 2)
	chainInput.ReversalID = transactionID(t, 4)
	chainInput.ReplacementID = transactionID(t, 5)
	chainInput.ReversalSequence = 4
	chainInput.ReplacementSequence = 5
	chain, err := BuildCorrection(chainInput)
	if err != nil || chain.Replacement.CorrectionOf() != facts.Replacement.ID() {
		t.Fatal(err)
	}
	input.ReplacementSequence = input.ReversalSequence
	if _, err := BuildCorrection(input); !errors.Is(err, ErrInvalidCorrection) {
		t.Fatal(err)
	}
	bad := before.state
	bad.Kind = KindReversal
	if _, err := RehydrateTransaction(bad); !errors.Is(err, ErrInvalidTransaction) {
		t.Fatal(err)
	}
}

func TestStrictRehydration(t *testing.T) {
	asset := assetSnapshot(t, 1)
	valid := fact(t, 20, 1, buy(t, asset, "1", testNow))
	for name, change := range map[string]func(*TransactionState){
		"zero id":            func(s *TransactionState) { s.ID = TransactionID{} },
		"zero sequence":      func(s *TransactionState) { s.PortfolioSequence = 0 },
		"bad kind":           func(s *TransactionState) { s.Kind = "ADJUSTMENT" },
		"amount on buy":      func(s *TransactionState) { s.Amount = positive(t, "1") },
		"missing asset":      func(s *TransactionState) { s.HasAsset = false },
		"future at creation": func(s *TransactionState) { s.EffectiveAt = s.CreatedAt.Add(time.Microsecond) },
		"submicrosecond":     func(s *TransactionState) { s.EffectiveAt = s.EffectiveAt.Add(time.Nanosecond) },
		"orphan correction":  func(s *TransactionState) { s.CorrectionOf = transactionID(t, 21) },
	} {
		t.Run(name, func(t *testing.T) {
			state := valid.state
			change(&state)
			if _, err := RehydrateTransaction(state); !errors.Is(err, ErrInvalidTransaction) {
				t.Fatal(err)
			}
		})
	}
}

func TestReplayOrderingBackdatingReversalAndIsolation(t *testing.T) {
	asset := assetSnapshot(t, 1)
	t1 := testNow.Add(-3 * time.Hour)
	t2 := testNow.Add(-2 * time.Hour)
	t3 := testNow.Add(-time.Hour)
	b := fact(t, 1, 1, buy(t, asset, "10", t1))
	s := fact(t, 2, 2, sell(t, asset, "4", t3))
	if err := ReplayLedger([]Transaction{b, s}, nil); err != nil {
		t.Fatal(err)
	}
	close := fact(t, 3, 3, sell(t, asset, "6", t3))
	if err := ReplayLedger([]Transaction{b, s, close}, nil); err != nil {
		t.Fatal(err)
	}
	oversell := fact(t, 4, 4, sell(t, asset, "7", t3))
	if !errors.Is(ReplayLedger([]Transaction{b, s, oversell}, nil), ErrInsufficientOrderedQuantity) {
		t.Fatal("oversell accepted")
	}
	backdated := fact(t, 5, 5, sell(t, asset, "7", t2))
	before := []Transaction{s, b}
	snapshot := append([]Transaction(nil), before...)
	err := ReplayLedger(before, []Transaction{backdated})
	var violation *LedgerViolation
	if !errors.As(err, &violation) || !violation.LaterExisting || violation.TransactionID != s.ID() {
		t.Fatalf("unexpected violation: %v", err)
	}
	if !reflect.DeepEqual(before, snapshot) {
		t.Fatal("caller slice mutated")
	}
	correction, err := BuildCorrection(CorrectionInput{Original: b, Replacement: buy(t, asset, "8", t1), CorrectionID: correctionID(t, 1), ReversalID: transactionID(t, 6), ReplacementID: transactionID(t, 7), Actor: actorID(t), ReversalSequence: 6, ReplacementSequence: 7, CreatedAt: testNow})
	if err != nil {
		t.Fatal(err)
	}
	if err := ReplayLedger([]Transaction{b}, []Transaction{correction.Reversal, correction.Replacement}); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(ReplayLedger(nil, []Transaction{correction.Reversal}), ErrInvalidLedger) {
		t.Fatal("orphan reversal accepted")
	}
	otherAsset := assetSnapshot(t, 2)
	tampered := correction.Reversal.state
	tampered.Asset = otherAsset
	malformed, _ := RehydrateTransaction(tampered)
	if !errors.Is(ReplayLedger([]Transaction{b}, []Transaction{malformed}), ErrInvalidLedger) {
		t.Fatal("cross-asset reversal accepted")
	}
	dividend, _ := NewDividendCommand(asset, positive(t, "100"), context(t2))
	cash, _ := NewWithdrawalCommand(positive(t, "1000"), context(t2))
	if err := ReplayLedger([]Transaction{b}, []Transaction{fact(t, 8, 8, dividend), fact(t, 9, 9, cash)}); err != nil {
		t.Fatal(err)
	}
	// A reversal of SELL restores quantity, not subtracts it again.
	sellCorrection, err := BuildCorrection(CorrectionInput{Original: s, Replacement: sell(t, asset, "2", t3), CorrectionID: correctionID(t, 2), ReversalID: transactionID(t, 10), ReplacementID: transactionID(t, 11), Actor: actorID(t), ReversalSequence: 10, ReplacementSequence: 11, CreatedAt: testNow})
	if err != nil {
		t.Fatal(err)
	}
	if err := ReplayLedger([]Transaction{b, s}, []Transaction{sellCorrection.Reversal, sellCorrection.Replacement}); err != nil {
		t.Fatal(err)
	}
	// A later buy may make the final sum positive but cannot repair an earlier oversell.
	laterBuy := fact(t, 12, 12, buy(t, asset, "100", testNow))
	if !errors.Is(ReplayLedger([]Transaction{b, s, laterBuy}, []Transaction{backdated}), ErrInsufficientOrderedQuantity) {
		t.Fatal("later buy masked backdated oversell")
	}
	// An Asset-changing correction must be checked independently in both streams.
	assetChanging, err := BuildCorrection(CorrectionInput{Original: b, Replacement: buy(t, otherAsset, "3", t2), CorrectionID: correctionID(t, 3), ReversalID: transactionID(t, 13), ReplacementID: transactionID(t, 14), Actor: actorID(t), ReversalSequence: 13, ReplacementSequence: 14, CreatedAt: testNow})
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(ReplayLedger([]Transaction{b, s}, []Transaction{assetChanging.Reversal, assetChanging.Replacement}), ErrInsufficientOrderedQuantity) {
		t.Fatal("asset-changing correction failed to validate original stream")
	}
}
