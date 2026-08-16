package http

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
)

const cursorVersion = 1

type cursorPayload struct {
	Version  int    `json:"v"`
	Symbol   string `json:"s"`
	Exchange string `json:"e"`
	AssetID  string `json:"i"`
}

func encodeCursor(position application.CursorPosition) (string, error) {
	if position.AssetID.IsZero() || position.Symbol == "" || position.Exchange == "" {
		return "", errInvalidRequest
	}
	payload, err := json.Marshal(cursorPayload{Version: cursorVersion, Symbol: position.Symbol, Exchange: position.Exchange, AssetID: position.AssetID.String()})
	if err != nil {
		return "", errInvalidRequest
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(value string) (application.CursorPosition, error) {
	if value == "" || len(value) > 512 {
		return application.CursorPosition{}, errInvalidRequest
	}
	encoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return application.CursorPosition{}, errInvalidRequest
	}
	if base64.RawURLEncoding.EncodeToString(encoded) != value {
		return application.CursorPosition{}, errInvalidRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var payload cursorPayload
	if err := decoder.Decode(&payload); err != nil {
		return application.CursorPosition{}, errInvalidRequest
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return application.CursorPosition{}, errInvalidRequest
	}
	if payload.Version != cursorVersion || payload.Symbol == "" || payload.Exchange == "" || payload.AssetID == "" {
		return application.CursorPosition{}, errInvalidRequest
	}
	id, err := domain.ParseAssetID(payload.AssetID)
	if err != nil {
		return application.CursorPosition{}, errInvalidRequest
	}
	if id.String() != payload.AssetID {
		return application.CursorPosition{}, errInvalidRequest
	}
	return application.CursorPosition{Symbol: payload.Symbol, Exchange: payload.Exchange, AssetID: id}, nil
}
