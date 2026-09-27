package domain

import (
	"time"

	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
	portfoliodomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

type Correction struct {
	id          CorrectionID
	portfolioID portfoliodomain.PortfolioID
	original    TransactionID
	reversal    TransactionID
	replacement TransactionID
	actor       identitydomain.UserID
	createdAt   time.Time
}

func RehydrateCorrection(id CorrectionID, portfolioID portfoliodomain.PortfolioID, original, reversal, replacement TransactionID, actor identitydomain.UserID, createdAt time.Time) (Correction, error) {
	if id.IsZero() || portfolioID.IsZero() || original.IsZero() || reversal.IsZero() || replacement.IsZero() || actor.IsZero() || createdAt.IsZero() || createdAt.Nanosecond()%1000 != 0 || original == reversal || original == replacement || reversal == replacement {
		return Correction{}, ErrInvalidCorrection
	}
	return Correction{id, portfolioID, original, reversal, replacement, actor, createdAt.UTC()}, nil
}
func (correction Correction) ID() CorrectionID                         { return correction.id }
func (correction Correction) PortfolioID() portfoliodomain.PortfolioID { return correction.portfolioID }
func (correction Correction) OriginalID() TransactionID                { return correction.original }
func (correction Correction) ReversalID() TransactionID                { return correction.reversal }
func (correction Correction) ReplacementID() TransactionID             { return correction.replacement }
func (correction Correction) Actor() identitydomain.UserID             { return correction.actor }
func (correction Correction) CreatedAt() time.Time                     { return correction.createdAt }

func ValidateCorrectionTarget(original Transaction, hasDirectCorrection bool) error {
	if original.ID().IsZero() {
		return ErrInvalidTransaction
	}
	if original.Kind() == KindReversal {
		return ErrTransactionNotCorrectable
	}
	if hasDirectCorrection {
		return ErrTransactionAlreadyCorrected
	}
	return nil
}

type CorrectionFacts struct {
	Reversal     Transaction
	Replacement  Transaction
	Relationship Correction
}
type CorrectionInput struct {
	Original            Transaction
	HasDirectCorrection bool
	Replacement         Command
	CorrectionID        CorrectionID
	ReversalID          TransactionID
	ReplacementID       TransactionID
	Actor               identitydomain.UserID
	ReversalSequence    int64
	ReplacementSequence int64
	CreatedAt           time.Time
}

// BuildCorrection builds three immutable facts; the application owns atomic persistence.
func BuildCorrection(input CorrectionInput) (CorrectionFacts, error) {
	if err := ValidateCorrectionTarget(input.Original, input.HasDirectCorrection); err != nil {
		return CorrectionFacts{}, err
	}
	if !input.Replacement.IsValid() || input.ReversalSequence <= 0 || input.ReplacementSequence <= input.ReversalSequence || input.ReversalID == input.Original.ID() || input.ReplacementID == input.Original.ID() {
		return CorrectionFacts{}, ErrInvalidCorrection
	}
	original := input.Original.state
	relationship, err := RehydrateCorrection(input.CorrectionID, original.PortfolioID, original.ID, input.ReversalID, input.ReplacementID, input.Actor, input.CreatedAt)
	if err != nil {
		return CorrectionFacts{}, err
	}
	reversalState := original
	reversalState.ID = input.ReversalID
	reversalState.CreatedBy = input.Actor
	reversalState.Kind = KindReversal
	reversalState.PortfolioSequence = input.ReversalSequence
	reversalState.Note = OptionalText{}
	reversalState.ExternalReference = OptionalText{}
	reversalState.ReversalOf = original.ID
	reversalState.CorrectionOf = TransactionID{}
	reversalState.OriginatingCorrection = input.CorrectionID
	reversalState.CreatedAt = input.CreatedAt
	reversal, err := RehydrateTransaction(reversalState)
	if err != nil {
		return CorrectionFacts{}, err
	}
	replacement, err := NewTransactionFromCommand(input.ReplacementID, original.PortfolioID, input.Actor, input.ReplacementSequence, input.CreatedAt, input.Replacement)
	if err != nil {
		return CorrectionFacts{}, err
	}
	replacementState := replacement.state
	replacementState.CorrectionOf = original.ID
	replacementState.OriginatingCorrection = input.CorrectionID
	replacement, err = RehydrateTransaction(replacementState)
	if err != nil {
		return CorrectionFacts{}, err
	}
	return CorrectionFacts{reversal, replacement, relationship}, nil
}
