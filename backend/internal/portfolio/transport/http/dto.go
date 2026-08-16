package http

import (
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/portfolio/domain"
)

type createPortfolioRequest struct {
	Name         *string `json:"name"`
	BaseCurrency *string `json:"baseCurrency"`
}

type updatePortfolioRequest struct {
	Name *string `json:"name"`
}

type portfolioResponse struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	BaseCurrency string     `json:"baseCurrency"`
	Status       string     `json:"status"`
	ArchivedAt   *time.Time `json:"archivedAt"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type portfolioListResponse struct {
	Items []portfolioResponse `json:"items"`
}

func responseFromPortfolio(portfolio domain.Portfolio) portfolioResponse {
	response := portfolioResponse{
		ID:           portfolio.ID().String(),
		Name:         portfolio.Name().String(),
		BaseCurrency: string(portfolio.BaseCurrency()),
		Status:       string(portfolio.Status()),
		CreatedAt:    portfolio.CreatedAt().UTC(),
		UpdatedAt:    portfolio.UpdatedAt().UTC(),
	}
	if archivedAt, ok := portfolio.ArchivedAt(); ok {
		utc := archivedAt.UTC()
		response.ArchivedAt = &utc
	}
	return response
}

func responseFromPortfolios(portfolios []domain.Portfolio) portfolioListResponse {
	items := make([]portfolioResponse, 0, len(portfolios))
	for _, portfolio := range portfolios {
		items = append(items, responseFromPortfolio(portfolio))
	}
	return portfolioListResponse{Items: items}
}
