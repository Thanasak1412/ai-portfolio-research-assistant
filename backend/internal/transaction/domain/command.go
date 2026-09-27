package domain

import "time"

// Command is a validated user-creatable fact; its financial shape is private.
type Command struct {
	kind              Kind
	asset             AssetSnapshot
	hasAsset          bool
	quantity          Decimal
	unitPrice         Decimal
	fee               Decimal
	amount            Decimal
	currency          Currency
	effectiveAt       time.Time
	note              OptionalText
	externalReference OptionalText
}

type CommandContext struct {
	Currency          Currency
	EffectiveAt       time.Time
	Now               time.Time
	Note              OptionalText
	ExternalReference OptionalText
}

func validatedContext(context CommandContext) (time.Time, error) {
	if context.Currency != CurrencyUSD {
		return time.Time{}, ErrInvalidCurrency
	}
	if err := validateOptional(context.Note, context.ExternalReference); err != nil {
		return time.Time{}, err
	}
	return ValidateEffectiveAt(context.EffectiveAt, context.Now)
}
func validateOptional(note, reference OptionalText) error {
	if note.present {
		if _, err := NewNote(note.value); err != nil {
			return err
		}
	}
	if reference.present {
		if _, err := NewExternalReference(reference.value); err != nil {
			return err
		}
	}
	return nil
}

func NewBuyCommand(asset AssetSnapshot, quantity, unitPrice, fee Decimal, context CommandContext) (Command, error) {
	return newTradeCommand(KindBuy, asset, quantity, unitPrice, fee, context)
}
func NewSellCommand(asset AssetSnapshot, quantity, unitPrice, fee Decimal, context CommandContext) (Command, error) {
	return newTradeCommand(KindSell, asset, quantity, unitPrice, fee, context)
}
func newTradeCommand(kind Kind, asset AssetSnapshot, quantity, unitPrice, fee Decimal, context CommandContext) (Command, error) {
	if !asset.IsValid() || !quantity.IsPositive() || !unitPrice.IsPositive() || !fee.IsNonNegative() {
		return Command{}, ErrInvalidTransactionFields
	}
	effectiveAt, err := validatedContext(context)
	if err != nil {
		return Command{}, err
	}
	return Command{kind: kind, asset: asset, hasAsset: true, quantity: quantity, unitPrice: unitPrice, fee: fee, currency: context.Currency, effectiveAt: effectiveAt, note: context.Note, externalReference: context.ExternalReference}, nil
}
func NewDividendCommand(asset AssetSnapshot, amount Decimal, context CommandContext) (Command, error) {
	if !asset.IsValid() || !amount.IsPositive() {
		return Command{}, ErrInvalidTransactionFields
	}
	effectiveAt, err := validatedContext(context)
	if err != nil {
		return Command{}, err
	}
	return Command{kind: KindDividend, asset: asset, hasAsset: true, amount: amount, currency: context.Currency, effectiveAt: effectiveAt, note: context.Note, externalReference: context.ExternalReference}, nil
}
func NewDepositCommand(amount Decimal, context CommandContext) (Command, error) {
	return newCashCommand(KindDeposit, amount, context)
}
func NewWithdrawalCommand(amount Decimal, context CommandContext) (Command, error) {
	return newCashCommand(KindWithdrawal, amount, context)
}
func NewFeeCommand(amount Decimal, context CommandContext) (Command, error) {
	return newCashCommand(KindFee, amount, context)
}
func newCashCommand(kind Kind, amount Decimal, context CommandContext) (Command, error) {
	if !amount.IsPositive() {
		return Command{}, ErrInvalidTransactionFields
	}
	effectiveAt, err := validatedContext(context)
	if err != nil {
		return Command{}, err
	}
	return Command{kind: kind, amount: amount, currency: context.Currency, effectiveAt: effectiveAt, note: context.Note, externalReference: context.ExternalReference}, nil
}
func (command Command) IsValid() bool {
	if !command.kind.IsPublicCreatable() || command.currency != CurrencyUSD || command.effectiveAt.IsZero() || command.effectiveAt.Nanosecond()%1000 != 0 || validateOptional(command.note, command.externalReference) != nil {
		return false
	}
	switch command.kind {
	case KindBuy, KindSell:
		return command.hasAsset && command.asset.IsValid() && command.quantity.IsPositive() && command.unitPrice.IsPositive() && command.fee.IsNonNegative() && !command.amount.IsValid()
	case KindDividend:
		return command.hasAsset && command.asset.IsValid() && command.amount.IsPositive() && !command.quantity.IsValid() && !command.unitPrice.IsValid() && !command.fee.IsValid()
	default:
		return !command.hasAsset && command.amount.IsPositive() && !command.quantity.IsValid() && !command.unitPrice.IsValid() && !command.fee.IsValid()
	}
}
func (command Command) Kind() Kind                   { return command.kind }
func (command Command) Asset() (AssetSnapshot, bool) { return command.asset, command.hasAsset }
func (command Command) Quantity() (Decimal, bool) {
	return command.quantity, command.quantity.IsValid()
}
func (command Command) UnitPrice() (Decimal, bool) {
	return command.unitPrice, command.unitPrice.IsValid()
}
func (command Command) Fee() (Decimal, bool)              { return command.fee, command.fee.IsValid() }
func (command Command) Amount() (Decimal, bool)           { return command.amount, command.amount.IsValid() }
func (command Command) Currency() Currency                { return command.currency }
func (command Command) EffectiveAt() time.Time            { return command.effectiveAt }
func (command Command) Note() (string, bool)              { return command.note.Value() }
func (command Command) ExternalReference() (string, bool) { return command.externalReference.Value() }
