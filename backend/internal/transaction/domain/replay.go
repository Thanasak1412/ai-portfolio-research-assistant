package domain

import (
	"fmt"
	"sort"

	assetdomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	portfoliodomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

// LedgerViolation identifies the first negative ordered position without
// exposing transport or persistence policy. LaterExisting distinguishes a
// backdated candidate's effect from an immediately insufficient candidate.
type LedgerViolation struct {
	TransactionID     TransactionID
	PortfolioSequence int64
	Candidate         bool
	LaterExisting     bool
	NegativeQuantity  Decimal
}

func (violation *LedgerViolation) Error() string { return ErrInsufficientOrderedQuantity.Error() }
func (violation *LedgerViolation) Unwrap() error { return ErrInsufficientOrderedQuantity }

type assetStreamKey struct {
	portfolio portfoliodomain.PortfolioID
	asset     assetdomain.AssetID
}

// ReplayLedger validates the whole ordered stream. It sorts a copy, never the
// caller's slices, and persists no holding, lot, cash, or cost-basis state.
func ReplayLedger(existing, candidates []Transaction) error {
	all := make([]Transaction, 0, len(existing)+len(candidates))
	all = append(all, existing...)
	all = append(all, candidates...)
	byID := make(map[TransactionID]Transaction, len(all))
	reversed := make(map[TransactionID]bool)
	sequences := make(map[portfoliodomain.PortfolioID]map[int64]bool)
	candidateIDs := make(map[TransactionID]bool, len(candidates))
	for _, candidate := range candidates {
		candidateIDs[candidate.ID()] = true
	}
	for _, fact := range all {
		if _, err := RehydrateTransaction(fact.state); err != nil {
			return fmt.Errorf("%w: invalid fact", ErrInvalidLedger)
		}
		if _, exists := byID[fact.ID()]; exists {
			return fmt.Errorf("%w: duplicate fact", ErrInvalidLedger)
		}
		byID[fact.ID()] = fact
		if sequences[fact.PortfolioID()] == nil {
			sequences[fact.PortfolioID()] = make(map[int64]bool)
		}
		if sequences[fact.PortfolioID()][fact.PortfolioSequence()] {
			return fmt.Errorf("%w: duplicate sequence", ErrInvalidLedger)
		}
		sequences[fact.PortfolioID()][fact.PortfolioSequence()] = true
	}
	for _, fact := range all {
		if fact.Kind() != KindReversal {
			continue
		}
		if reversed[fact.ReversalOf()] {
			return fmt.Errorf("%w: duplicate direct reversal", ErrInvalidLedger)
		}
		reversed[fact.ReversalOf()] = true
		original, exists := byID[fact.ReversalOf()]
		if !exists || original.Kind() == KindReversal || original.PortfolioID() != fact.PortfolioID() || !sameFinancialDimensions(original, fact) || !original.EffectiveAt().Equal(fact.EffectiveAt()) {
			return fmt.Errorf("%w: inconsistent reversal", ErrInvalidLedger)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if !a.EffectiveAt().Equal(b.EffectiveAt()) {
			return a.EffectiveAt().Before(b.EffectiveAt())
		}
		if a.PortfolioSequence() != b.PortfolioSequence() {
			return a.PortfolioSequence() < b.PortfolioSequence()
		}
		return a.ID().String() < b.ID().String()
	})
	balances := make(map[assetStreamKey]Decimal)
	seenCandidate := make(map[assetStreamKey]bool)
	zero, _ := ParseNonNegativeDecimal("0")
	for _, fact := range all {
		asset, hasAsset := fact.Asset()
		if !hasAsset {
			continue
		}
		key := assetStreamKey{fact.PortfolioID(), asset.ID()}
		if candidateIDs[fact.ID()] {
			seenCandidate[key] = true
		}
		quantity, hasQuantity := fact.Quantity()
		if !hasQuantity {
			continue
		}
		balance, exists := balances[key]
		if !exists {
			balance = zero
		}
		var next Decimal
		var err error
		switch fact.Kind() {
		case KindBuy:
			next, err = balance.Add(quantity)
		case KindSell:
			next, err = balance.Sub(quantity)
		case KindReversal:
			if byID[fact.ReversalOf()].Kind() == KindBuy {
				next, err = balance.Sub(quantity)
			} else {
				next, err = balance.Add(quantity)
			}
		default:
			return fmt.Errorf("%w: quantity on non-trade", ErrInvalidLedger)
		}
		if err != nil {
			return err
		}
		if !next.IsNonNegative() {
			return &LedgerViolation{fact.ID(), fact.PortfolioSequence(), candidateIDs[fact.ID()], !candidateIDs[fact.ID()] && seenCandidate[key], next}
		}
		balances[key] = next
	}
	return nil
}

func sameFinancialDimensions(a, b Transaction) bool {
	x, xAsset := a.Asset()
	y, yAsset := b.Asset()
	return xAsset == yAsset && x == y && a.state.Quantity == b.state.Quantity && a.state.UnitPrice == b.state.UnitPrice && a.state.Fee == b.state.Fee && a.state.Amount == b.state.Amount && a.Currency() == b.Currency()
}
