package http

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/gofiber/fiber/v2"
)

func encodeCursor(position application.Position) string {
	payload, _ := json.Marshal([3]string{position.EffectiveAt.UTC().Format(time.RFC3339Nano), strconv.FormatInt(position.Sequence, 10), position.ID.String()})
	return "v1." + base64.RawURLEncoding.EncodeToString(payload)
}

func decodeCursor(value string) (*application.Position, error) {
	invalid := application.ErrInvalidInput
	if len(value) > 512 || !strings.HasPrefix(value, "v1.") {
		return nil, invalid
	}
	payload, err := base64.RawURLEncoding.Strict().DecodeString(strings.TrimPrefix(value, "v1."))
	if err != nil {
		return nil, invalid
	}
	var tuple []string
	if json.Unmarshal(payload, &tuple) != nil || len(tuple) != 3 {
		return nil, invalid
	}
	timestamp, err := parseTime(tuple[0])
	if err != nil {
		return nil, invalid
	}
	sequence, err := strconv.ParseInt(tuple[1], 10, 64)
	if err != nil || sequence <= 0 {
		return nil, invalid
	}
	id, err := domain.ParseTransactionID(tuple[2])
	if err != nil {
		return nil, invalid
	}
	position := application.Position{EffectiveAt: timestamp, Sequence: sequence, ID: id}
	if encodeCursor(position) != value {
		return nil, invalid
	}
	return &position, nil
}

func historyInput(ctx *fiber.Ctx) (application.HistoryInput, error) {
	input := application.HistoryInput{Limit: 50}
	values := make(map[string]string)
	valid := true
	ctx.Context().QueryArgs().VisitAll(func(key, value []byte) {
		name := string(key)
		if _, duplicate := values[name]; duplicate {
			valid = false
		}
		switch name {
		case "kind", "effectiveAtFrom", "effectiveAtTo", "includeReversals", "limit", "cursor":
		default:
			valid = false
		}
		values[name] = string(value)
	})
	if !valid {
		return input, application.ErrInvalidInput
	}
	if value, ok := values["kind"]; ok {
		kind, err := domain.ParseKind(value)
		if err != nil {
			return input, application.ErrInvalidInput
		}
		input.Kind = &kind
	}
	for name, destination := range map[string]**time.Time{"effectiveAtFrom": &input.From, "effectiveAtTo": &input.To} {
		if value, ok := values[name]; ok {
			timestamp, err := parseTime(value)
			if err != nil {
				return input, err
			}
			*destination = &timestamp
		}
	}
	if input.From != nil && input.To != nil && input.From.After(*input.To) {
		return input, application.ErrInvalidInput
	}
	if value, ok := values["includeReversals"]; ok {
		if value != "true" && value != "false" {
			return input, application.ErrInvalidInput
		}
		include := value == "true"
		input.IncludeReversals = &include
	}
	if value, ok := values["limit"]; ok {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 || strconv.Itoa(limit) != value {
			return input, application.ErrInvalidInput
		}
		input.Limit = limit
	}
	if value, ok := values["cursor"]; ok {
		position, err := decodeCursor(value)
		if err != nil {
			return input, err
		}
		input.After = position
	}
	return input, nil
}
