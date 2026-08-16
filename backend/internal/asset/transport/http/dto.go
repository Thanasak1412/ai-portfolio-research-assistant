package http

import "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"

type assetResponse struct {
	ID        string `json:"id"`
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	AssetType string `json:"assetType"`
	Exchange  string `json:"exchange"`
	Currency  string `json:"currency"`
}

type assetListResponse struct {
	Items      []assetResponse `json:"items"`
	NextCursor *string         `json:"nextCursor"`
}

func responseFromAsset(asset domain.Asset) assetResponse {
	return assetResponse{ID: asset.ID().String(), Symbol: asset.Symbol(), Name: asset.Name(), AssetType: string(asset.AssetType()), Exchange: asset.Exchange(), Currency: string(asset.Currency())}
}
