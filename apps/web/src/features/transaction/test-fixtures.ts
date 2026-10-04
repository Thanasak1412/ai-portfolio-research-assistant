import type { Asset } from "@/features/asset/api/asset-api";
import type {
  Transaction,
  TransactionCommand,
} from "@/features/transaction/api/transaction-api";

export const command = {
  kind: "DEPOSIT",
  amount: "10.000",
  currency: "USD",
  effectiveAt: "2020-01-02T12:30:00.123456Z",
} satisfies TransactionCommand;
export const transaction: Transaction = {
  id: "transaction-1",
  kind: "DEPOSIT",
  assetId: null,
  quantity: null,
  unitPrice: null,
  fee: null,
  amount: "10",
  currency: "USD",
  effectiveAt: command.effectiveAt,
  portfolioSequence: "12345678901234567890",
  note: null,
  externalReference: null,
  createdAt: "2020-01-02T12:30:01Z",
  correctionLinks: {
    reversesTransactionId: null,
    replacesTransactionId: null,
    reversalTransactionId: null,
    replacementTransactionId: null,
  },
};
export const asset: Asset = {
  id: "asset-1",
  symbol: "SYNTH",
  name: "Synthetic Equity",
  assetType: "EQUITY",
  exchange: "NYSE",
  currency: "USD",
};
export const portfolio = {
  id: "portfolio-1",
  name: "Synthetic Portfolio",
  baseCurrency: "USD",
  status: "ACTIVE",
  archivedAt: null,
  createdAt: "2020-01-01T00:00:00Z",
  updatedAt: "2020-01-01T00:00:00Z",
} as const;
