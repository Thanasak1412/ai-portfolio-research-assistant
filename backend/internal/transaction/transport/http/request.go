package http

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"regexp"
	"time"
	"unicode/utf8"

	asset "github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/asset/domain"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/transaction/domain"
	"github.com/gofiber/fiber/v2"
)

const maxCommandBytes = 8192

var utcTimestamp = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$`)
var keyGrammar = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._~-]{15,127}$`)

// object rejects duplicate keys, non-objects and trailing JSON, rather than
// allowing encoding/json's last-key-wins or case-insensitive struct matching.
func object(body []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return nil, application.ErrInvalidInput
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, application.ErrInvalidInput
		}
		name, ok := key.(string)
		if !ok {
			return nil, application.ErrInvalidInput
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, application.ErrInvalidInput
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return nil, application.ErrInvalidInput
		}
		fields[name] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, application.ErrInvalidInput
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, application.ErrInvalidInput
	}
	return fields, nil
}

func parseTime(value string) (time.Time, error) {
	if !utcTimestamp.MatchString(value) {
		return time.Time{}, application.ErrInvalidInput
	}
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || timestamp.IsZero() {
		return time.Time{}, application.ErrInvalidInput
	}
	return timestamp, nil
}

func decodeCommand(ctx *fiber.Ctx, correction bool) (application.CommandInput, error) {
	body := ctx.Body()
	media, _, err := mime.ParseMediaType(ctx.Get(fiber.HeaderContentType))
	if err != nil || media != "application/json" || len(body) > maxCommandBytes || !utf8.Valid(body) {
		return application.CommandInput{}, application.ErrInvalidInput
	}
	return parseCommand(body, correction)
}

func parseCommand(body []byte, correction bool) (application.CommandInput, error) {
	var input application.CommandInput
	fields, err := object(body)
	if err != nil {
		return input, err
	}
	if correction {
		if len(fields) != 1 || fields["replacement"] == nil {
			return input, application.ErrInvalidInput
		}
		fields, err = object(fields["replacement"])
		if err != nil {
			return input, err
		}
	}
	values := make(map[string]string, len(fields))
	for name, raw := range fields {
		switch name {
		case "kind", "currency", "effectiveAt", "assetId", "quantity", "unitPrice", "fee", "amount", "note", "externalReference":
		default:
			return input, domain.ErrInvalidTransactionFields
		}
		var value string
		if bytes.Equal(raw, []byte("null")) || json.Unmarshal(raw, &value) != nil {
			return input, application.ErrInvalidInput
		}
		values[name] = value
	}
	for _, name := range []string{"kind", "currency", "effectiveAt"} {
		if _, ok := values[name]; !ok {
			return input, domain.ErrInvalidTransactionFields
		}
	}
	input.Kind, err = domain.ParseKind(values["kind"])
	if err != nil || !input.Kind.IsPublicCreatable() {
		return input, domain.ErrInvalidTransactionKind
	}
	input.Currency, err = domain.ParseCurrency(values["currency"])
	if err != nil {
		return input, err
	}
	input.EffectiveAt, err = parseTime(values["effectiveAt"])
	if err != nil {
		return input, err
	}
	trade := input.Kind == domain.KindBuy || input.Kind == domain.KindSell
	assetBacked := trade || input.Kind == domain.KindDividend
	for _, name := range []string{"assetId", "quantity", "unitPrice", "fee", "amount"} {
		_, present := values[name]
		required := (name == "assetId" && assetBacked) || ((name == "quantity" || name == "unitPrice") && trade) || (name == "amount" && !trade)
		allowed := required || (name == "fee" && trade)
		if required && !present || !allowed && present {
			return input, domain.ErrInvalidTransactionFields
		}
	}
	if assetBacked {
		if utf8.RuneCountInString(values["assetId"]) < 1 || utf8.RuneCountInString(values["assetId"]) > 128 {
			return input, application.ErrInvalidInput
		}
		input.AssetID, err = asset.ParseAssetID(values["assetId"])
		if err != nil {
			return input, application.ErrAssetNotFound
		}
	}
	for name, destination := range map[string]*domain.Decimal{"quantity": &input.Quantity, "unitPrice": &input.UnitPrice, "fee": &input.Fee, "amount": &input.Amount} {
		if value, present := values[name]; present {
			if name == "fee" {
				*destination, err = domain.ParseNonNegativeDecimal(value)
			} else {
				*destination, err = domain.ParsePositiveDecimal(value)
			}
			if err != nil {
				return input, err
			}
		}
	}
	if value, present := values["note"]; present {
		input.Note, err = domain.NewNote(value)
		if err != nil {
			return input, err
		}
	}
	if value, present := values["externalReference"]; present {
		input.ExternalReference, err = domain.NewExternalReference(value)
	}
	return input, err
}

func metadata(ctx *fiber.Ctx, correlation string) (application.CommandMetadata, error) {
	count, key := 0, ""
	ctx.Request().Header.VisitAll(func(name, value []byte) {
		if bytes.EqualFold(name, []byte("Idempotency-Key")) {
			count++
			key = string(value)
		}
	})
	if count != 1 || !keyGrammar.MatchString(key) {
		return application.CommandMetadata{}, application.ErrInvalidIdempotencyKey
	}
	return application.CommandMetadata{IdempotencyKey: key, CorrelationID: correlation}, nil
}
