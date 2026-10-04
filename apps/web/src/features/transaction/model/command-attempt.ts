import type { TransactionCommand } from "@/features/transaction/api/transaction-api";
import { canonicalTime } from "@/features/transaction/model/transaction-validation";

const canonicalDecimal = (value: string) =>
  value.includes(".") ? value.replace(/0+$/, "").replace(/\.$/, "") : value;
function identity(command: TransactionCommand): string {
  return JSON.stringify({
    kind: command.kind,
    assetId: "assetId" in command ? command.assetId : null,
    quantity: "quantity" in command ? canonicalDecimal(command.quantity) : null,
    unitPrice:
      "unitPrice" in command ? canonicalDecimal(command.unitPrice) : null,
    fee: "quantity" in command ? canonicalDecimal(command.fee ?? "0") : null,
    amount: "amount" in command ? canonicalDecimal(command.amount) : null,
    currency: command.currency,
    effectiveAt: canonicalTime(command.effectiveAt),
    note: command.note ?? null,
    externalReference: command.externalReference ?? null,
  });
}

// One instance per mounted entry form; no browser persistence or token access.
export class CommandAttempt {
  private attempt: { identity: string; key: string } | null = null;
  keyFor(command: TransactionCommand): string {
    const next = identity(command);
    if (this.attempt?.identity !== next)
      this.attempt = { identity: next, key: crypto.randomUUID() };
    return this.attempt.key;
  }
  clear() {
    this.attempt = null;
  }
}
