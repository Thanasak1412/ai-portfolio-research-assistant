# Transaction Ledger Domain Foundation

Task: `M3-BE-001`. This package defines pure immutable Transaction facts and
transient ledger validity. It does not activate the Transaction API; application
orchestration remains `M3-BE-002`.

## Exact values and commands

`Decimal` stores a canonical base-ten string, with `math/big` used only for
exact arithmetic. Public input accepts unsigned non-exponent decimal strings
with at most 12 fractional digits. Positive values are required for quantity,
unit price, and amount; trade fee is non-negative. Trailing fractional zeroes
are removed canonically, so `1`, `1.0`, and `1.00` are the same semantic value.
No float or rounding is involved. Integer magnitude is not artificially capped.

BUY and SELL require an eligible Asset snapshot, quantity, unit price, and
explicit non-negative fee; an omitted fee is converted to exact zero before
constructing the command. DIVIDEND requires an Asset and amount. DEPOSIT,
WITHDRAWAL, and standalone FEE require only amount. All commands require USD
and a nonfuture microsecond-precision effective instant. Note and external
reference distinguish absent from empty and preserve exact Unicode bytes.

The Asset snapshot contains only canonical Asset ID, type, exchange, and
currency. Financial eligibility is EQUITY/ETF, USD, and NYSE/NASDAQ/NYSEARCA/
AMEX. Asset lookup and ownership remain application responsibilities.

## Immutable facts and corrections

`Transaction` has private state, strict rehydration, and read-only accessors.
The six public kinds are BUY, SELL, DIVIDEND, DEPOSIT, WITHDRAWAL, and FEE.
REVERSAL is internal and created only by `BuildCorrection`. ADJUSTMENT is not
accepted. No financial fact is edited or deleted.

A correction creates a reversal (copied positive financial dimensions, original
effective time, no note/reference), a complete public replacement (its command's
effective time), and one immutable relationship. The original is unchanged. A
replacement can later become the original of another correction. Persistence
must atomically commit the three facts; this domain package performs no writes.

## Ordered validity replay

`ReplayLedger` sorts a copy of immutable facts by effective time and Portfolio
sequence, resolving every reversal against its original fact. BUY adds quantity,
SELL subtracts quantity, and reversal negates the original kind's effect. It
rejects any negative running asset quantity at any ordered position, including
later positions invalidated by a backdated candidate. Cash events have no
quantity effect and no cash-sufficiency rule. The result is validation only:
no persisted Holding, lot, FIFO cost basis, cash projection, price, or valuation.
