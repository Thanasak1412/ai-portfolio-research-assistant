// Package http exposes the frozen read-only M2 Asset contract.
package http

import (
	"context"
	"strconv"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	identitydomain "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

type Operations interface {
	GetAsset(context.Context, identitydomain.Principal, domain.AssetID) (domain.Asset, error)
	SearchAssets(context.Context, identitydomain.Principal, application.SearchInput) (application.SearchResult, error)
}

type PrincipalExtractor func(*fiber.Ctx) (identitydomain.Principal, bool)

type Handler struct {
	operations Operations
	bearer     fiber.Handler
	principal  PrincipalExtractor
}

func NewHandler(operations Operations, bearer fiber.Handler, principal PrincipalExtractor) (*Handler, error) {
	if operations == nil || bearer == nil || principal == nil {
		return nil, application.ErrAssetService
	}
	return &Handler{operations: operations, bearer: bearer, principal: principal}, nil
}

func (handler *Handler) Mount(router fiber.Router) {
	assets := router.Group("/assets", handler.bearer)
	assets.Get("/", handler.list)
	assets.Get("/:assetId", handler.get)
}

func (handler *Handler) get(ctx *fiber.Ctx) error {
	id, err := domain.ParseAssetID(ctx.Params("assetId"))
	if err != nil {
		return writeError(ctx, application.ErrAssetNotFound)
	}
	asset, err := handler.operations.GetAsset(ctx.UserContext(), handler.validPrincipal(ctx), id)
	if err != nil {
		return writeError(ctx, err)
	}
	return ctx.JSON(responseFromAsset(asset))
}

func (handler *Handler) list(ctx *fiber.Ctx) error {
	input, err := parseSearchInput(ctx)
	if err != nil {
		return writeError(ctx, err)
	}
	result, err := handler.operations.SearchAssets(ctx.UserContext(), handler.validPrincipal(ctx), input)
	if err != nil {
		return writeError(ctx, err)
	}
	items := make([]assetResponse, 0, len(result.Assets))
	for _, asset := range result.Assets {
		items = append(items, responseFromAsset(asset))
	}
	response := assetListResponse{Items: items}
	if result.Next != nil {
		cursor, err := encodeCursor(*result.Next)
		if err != nil {
			return writeError(ctx, err)
		}
		response.NextCursor = &cursor
	}
	return ctx.JSON(response)
}

func parseSearchInput(ctx *fiber.Ctx) (application.SearchInput, error) {
	input := application.SearchInput{Limit: application.DefaultSearchLimit}
	arguments := ctx.Context().QueryArgs()
	if arguments.Has("search") {
		input.Search, input.HasSearch = ctx.Query("search"), true
		if input.Search == "" || !utf8.ValidString(input.Search) || utf8.RuneCountInString(input.Search) > 100 {
			return application.SearchInput{}, errInvalidRequest
		}
	}
	if arguments.Has("type") {
		typeValue, err := domain.ParseAssetType(ctx.Query("type"))
		if err != nil {
			return application.SearchInput{}, errInvalidRequest
		}
		input.AssetType = &typeValue
	}
	if arguments.Has("limit") {
		limit, err := strconv.Atoi(ctx.Query("limit"))
		if err != nil || limit < 1 || limit > application.MaximumSearchLimit {
			return application.SearchInput{}, errInvalidRequest
		}
		input.Limit = limit
	}
	if arguments.Has("cursor") {
		cursor, err := decodeCursor(ctx.Query("cursor"))
		if err != nil {
			return application.SearchInput{}, errInvalidRequest
		}
		input.Cursor = &cursor
	}
	return input, nil
}

func (handler *Handler) validPrincipal(ctx *fiber.Ctx) identitydomain.Principal {
	principal, ok := handler.principal(ctx)
	if !ok {
		return identitydomain.Principal{}
	}
	return principal
}

var _ interface{ Mount(fiber.Router) } = (*Handler)(nil)
