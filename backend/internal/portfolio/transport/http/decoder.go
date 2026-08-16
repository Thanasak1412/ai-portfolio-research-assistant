package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"

	"github.com/gofiber/fiber/v2"
)

var errInvalidJSON = errors.New("invalid JSON request")

func decodeJSON(ctx *fiber.Ctx, target any) error {
	mediaType, _, err := mime.ParseMediaType(ctx.Get(fiber.HeaderContentType))
	if err != nil || mediaType != fiber.MIMEApplicationJSON || len(ctx.Body()) == 0 {
		return errInvalidJSON
	}
	decoder := json.NewDecoder(bytes.NewReader(ctx.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return errInvalidJSON
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errInvalidJSON
	}
	return nil
}
