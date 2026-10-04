import { ApiError } from "@/platform/api/api-error";

const messages: Record<string, string> = {
  TRANSACTION_NOT_FOUND: "Transaction not found.",
  TRANSACTION_ALREADY_CORRECTED:
    "This Transaction already has a direct correction. Refresh ledger history to see its reversal and replacement. To correct again, select the replacement Transaction.",
  TRANSACTION_NOT_CORRECTABLE:
    "Internal reversals cannot be corrected. The original and reversal remain immutable in history.",
  PORTFOLIO_NOT_FOUND: "Portfolio not found.",
  PORTFOLIO_ARCHIVED:
    "This Portfolio is archived. New entries and corrections are unavailable.",
  ASSET_NOT_FOUND: "The selected Asset is unavailable. Choose another Asset.",
  ASSET_FINANCIALLY_INELIGIBLE:
    "This Asset is not eligible for USD transactions.",
  INVALID_EFFECTIVE_AT:
    "Enter a valid UTC effective time that is not in the future.",
  INSUFFICIENT_ORDERED_ASSET_QUANTITY:
    "The entry exceeds available asset quantity in ledger order.",
  INVALID_BACKDATED_LEDGER:
    "This backdated entry would make the ordered ledger invalid.",
  IDEMPOTENCY_CONFLICT:
    "This attempt conflicts with an earlier command. Review the ledger before starting a new command.",
  INVALID_TRANSACTION_FIELDS:
    "Check the fields required for this transaction kind.",
  INVALID_DECIMAL:
    "Check the decimal values. Use at most 12 fractional digits.",
  INVALID_REQUEST: "Check the request fields and try again.",
  UNSUPPORTED_TRANSACTION_KIND: "Choose a supported transaction kind.",
  UNSUPPORTED_TRANSACTION_CURRENCY: "Transactions must use USD.",
  ACCESS_TOKEN_INVALID: "Your session is unavailable. Sign in again.",
};

export function TransactionError({ error }: Readonly<{ error: unknown }>) {
  return (
    <div role="alert" className="text-sm text-red-700">
      <p>
        {error instanceof ApiError
          ? (messages[error.code] ??
            "Transaction service is unavailable. Please try again.")
          : "Transaction service is unavailable. Please try again."}
      </p>
      {error instanceof ApiError && error.correlationId && (
        <p>Support reference: {error.correlationId}</p>
      )}
    </div>
  );
}
