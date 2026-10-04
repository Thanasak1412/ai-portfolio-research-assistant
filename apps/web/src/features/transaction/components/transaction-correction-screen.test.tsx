import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { TransactionScreen } from "@/features/transaction/components/transaction-screen";
import {
  correction,
  portfolio,
  transaction,
} from "@/features/transaction/test-fixtures";

const mocks = vi.hoisted(() => ({
  portfolio: vi.fn(),
  history: vi.fn(),
  correct: vi.fn(),
}));
vi.mock("@/features/portfolio/model/portfolio-queries", () => ({
  usePortfolio: mocks.portfolio,
}));
vi.mock("@/features/transaction/model/transaction-queries", () => ({
  useTransactionHistory: mocks.history,
  useCreateTransaction: () => ({ mutateAsync: vi.fn(), reset: vi.fn() }),
  useCorrectTransaction: () => ({ mutateAsync: mocks.correct, reset: vi.fn() }),
}));
vi.mock("@/features/asset/model/asset-queries", () => ({
  useAssets: () => ({ isSuccess: true, data: { pages: [{ items: [] }] } }),
}));
beforeEach(() => {
  vi.resetAllMocks();
  mocks.portfolio.mockReturnValue({ data: portfolio });
  mocks.history.mockReturnValue({
    isSuccess: true,
    data: { pages: [{ items: [transaction] }] },
  });
});
it("connects history to correction, retains create draft, returns focus, and removes correction on archival", () => {
  const client = new QueryClient();
  const view = (
    <QueryClientProvider client={client}>
      <TransactionScreen portfolioId="p" />
    </QueryClientProvider>
  );
  const { rerender } = render(view);
  fireEvent.change(screen.getByLabelText("Quantity"), {
    target: { value: "1.23" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Correct" }));
  expect(
    screen.getByRole("heading", { name: "Correct Transaction" }),
  ).toHaveFocus();
  expect(
    screen.queryByRole("form", { name: "Transaction entry" }),
  ).not.toBeInTheDocument();
  expect(
    screen.getByRole("form", { name: "Complete replacement" }),
  ).toBeVisible();
  expect(screen.getByRole("button", { name: "Correct" })).toBeDisabled();
  fireEvent.click(screen.getByRole("button", { name: "Return to history" }));
  expect(screen.getByRole("heading", { name: "Ledger history" })).toHaveFocus();
  expect(screen.getByLabelText("Quantity")).toHaveValue("1.23");
  fireEvent.click(screen.getByRole("button", { name: "Correct" }));
  mocks.portfolio.mockReturnValue({
    data: { ...portfolio, status: "ARCHIVED" },
  });
  mocks.history.mockReturnValue({
    isSuccess: true,
    data: {
      pages: [
        {
          items: [
            correction.original,
            correction.reversal,
            correction.replacement,
          ],
        },
      ],
    },
  });
  rerender(
    <QueryClientProvider client={client}>
      <TransactionScreen portfolioId="p" />
    </QueryClientProvider>,
  );
  expect(
    screen.queryByRole("heading", { name: "Correct Transaction" }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: "Correct" }),
  ).not.toBeInTheDocument();
  expect(
    screen.getByRole("link", {
      name: `Replacement: ${correction.replacement.id}`,
    }),
  ).toBeVisible();
  expect(screen.getByText(/archived. History is read-only/)).toBeVisible();
});
it("discards target and draft state when moving to another Portfolio", () => {
  const client = new QueryClient();
  const { rerender } = render(
    <QueryClientProvider client={client}>
      <TransactionScreen portfolioId="p" />
    </QueryClientProvider>,
  );
  fireEvent.click(screen.getByRole("button", { name: "Correct" }));
  rerender(
    <QueryClientProvider client={client}>
      <TransactionScreen portfolioId="other" />
    </QueryClientProvider>,
  );
  expect(
    screen.queryByRole("heading", { name: "Correct Transaction" }),
  ).not.toBeInTheDocument();
  expect(
    within(
      screen.getByRole("form", { name: "Transaction entry" }),
    ).getByLabelText("Quantity"),
  ).toHaveValue("");
});
