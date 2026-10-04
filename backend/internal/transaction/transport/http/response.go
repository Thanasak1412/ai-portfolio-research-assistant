package http

import (
	"strconv"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
)

type correctionLinks struct {
	Reverses    *string `json:"reversesTransactionId"`
	Replaces    *string `json:"replacesTransactionId"`
	Reversal    *string `json:"reversalTransactionId"`
	Replacement *string `json:"replacementTransactionId"`
}
type transactionResponse struct {
	ID                string          `json:"id"`
	Kind              domain.Kind     `json:"kind"`
	AssetID           *string         `json:"assetId"`
	Quantity          *string         `json:"quantity"`
	UnitPrice         *string         `json:"unitPrice"`
	Fee               *string         `json:"fee"`
	Amount            *string         `json:"amount"`
	Currency          domain.Currency `json:"currency"`
	EffectiveAt       time.Time       `json:"effectiveAt"`
	PortfolioSequence string          `json:"portfolioSequence"`
	Note              *string         `json:"note"`
	ExternalReference *string         `json:"externalReference"`
	CorrectionLinks   correctionLinks `json:"correctionLinks"`
	CreatedAt         time.Time       `json:"createdAt"`
}
type correctionResponse struct {
	Original    transactionResponse `json:"original"`
	Reversal    transactionResponse `json:"reversal"`
	Replacement transactionResponse `json:"replacement"`
}
type historyResponse struct {
	Items      []transactionResponse `json:"items"`
	NextCursor *string               `json:"nextCursor"`
}

func optional(value string, present bool) *string {
	if !present {
		return nil
	}
	return &value
}
func decimal(value domain.Decimal, present bool) *string { return optional(value.String(), present) }
func idLink(id domain.TransactionID) *string             { return optional(id.String(), !id.IsZero()) }

func responseFromRecord(record application.Record) transactionResponse {
	fact := record.Transaction
	result := transactionResponse{
		ID: fact.ID().String(), Kind: fact.Kind(), Currency: fact.Currency(), EffectiveAt: fact.EffectiveAt(),
		PortfolioSequence: strconv.FormatInt(fact.PortfolioSequence(), 10), CreatedAt: fact.CreatedAt(),
		Quantity: decimal(fact.Quantity()), UnitPrice: decimal(fact.UnitPrice()), Fee: decimal(fact.Fee()), Amount: decimal(fact.Amount()),
		Note: optional(fact.Note()), ExternalReference: optional(fact.ExternalReference()),
		CorrectionLinks: correctionLinks{Reverses: idLink(fact.ReversalOf()), Replaces: idLink(fact.CorrectionOf())},
	}
	if asset, ok := fact.Asset(); ok {
		id := asset.ID().String()
		result.AssetID = &id
	}
	if correction := record.DirectCorrection; correction != nil {
		result.CorrectionLinks.Reversal = idLink(correction.ReversalID())
		result.CorrectionLinks.Replacement = idLink(correction.ReplacementID())
	}
	return result
}
