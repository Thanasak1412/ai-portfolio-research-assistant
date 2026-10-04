import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { transactionApi } from "@/features/transaction/api/transaction-api";
import { transactionKeys } from "@/features/transaction/model/transaction-query-keys";
import {
  useCreateTransaction,
  useCorrectTransaction,
  useTransaction,
  useTransactionHistory,
} from "@/features/transaction/model/transaction-queries";
import {
  command,
  correction,
  transaction,
} from "@/features/transaction/test-fixtures";
import { ApiError } from "@/platform/api/api-error";

const auth = vi.hoisted(() => ({
  runAuthenticated: vi.fn(),
  state: { status: "authenticated" },
}));
vi.mock("@/features/auth/model/auth-session-provider", () => ({
  useAuthSession: () => auth,
}));
vi.mock("@/features/transaction/api/transaction-api", () => ({
  transactionApi: {
    list: vi.fn(),
    create: vi.fn(),
    correct: vi.fn(),
    get: vi.fn(),
  },
}));
const wrapper = (client: QueryClient) =>
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    );
  };
beforeEach(() => {
  vi.clearAllMocks();
  auth.state.status = "authenticated";
  auth.runAuthenticated.mockImplementation((operation) =>
    operation("memory-access"),
  );
});

describe("Transaction query model", () => {
  it("loads related facts with Portfolio-scoped detail keys and authenticated reads", async () => {
    vi.mocked(transactionApi.get).mockResolvedValue(transaction);
    const client = new QueryClient();
    const { result, unmount } = renderHook(
      () => useTransaction("p", transaction.id),
      { wrapper: wrapper(client) },
    );
    await waitFor(() => expect(result.current.data).toEqual(transaction));
    expect(transactionApi.get).toHaveBeenCalledWith(
      "memory-access",
      "p",
      transaction.id,
    );
    expect(transactionKeys.detail("p", transaction.id)).toEqual([
      "transactions",
      "p",
      "detail",
      transaction.id,
    ]);
    expect(transactionKeys.detail("other", transaction.id)).not.toEqual(
      transactionKeys.detail("p", transaction.id),
    );
    unmount();
    client.clear();
    auth.state.status = "unauthenticated";
    vi.mocked(transactionApi.get).mockClear();
    renderHook(() => useTransaction("p", transaction.id), {
      wrapper: wrapper(client),
    });
    expect(transactionApi.get).not.toHaveBeenCalled();
    client.clear();
  });
  it("shows correction success without waiting on background history/detail invalidation", async () => {
    vi.mocked(transactionApi.correct).mockResolvedValue(correction);
    const client = new QueryClient();
    const invalidate = vi
      .spyOn(client, "invalidateQueries")
      .mockReturnValue(new Promise(() => {}));
    const { result } = renderHook(() => useCorrectTransaction("p"), {
      wrapper: wrapper(client),
    });
    await act(async () => {
      await expect(
        result.current.mutateAsync({
          transactionId: transaction.id,
          command: { replacement: command },
          key: "correction-attempt-key",
        }),
      ).resolves.toEqual(correction);
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: transactionKeys.histories("p"),
    });
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: transactionKeys.details("p"),
    });
    client.clear();
  });
  it.each(["INTERNAL_ERROR", "TRANSACTION_ALREADY_CORRECTED"])(
    "never automatically retries %s; refreshes already-corrected history",
    async (code) => {
      vi.mocked(transactionApi.correct).mockRejectedValue(
        new ApiError(code === "INTERNAL_ERROR" ? 500 : 409, code, "private"),
      );
      const client = new QueryClient({
        defaultOptions: { mutations: { retry: 3, retryDelay: 0 } },
      });
      const invalidate = vi
        .spyOn(client, "invalidateQueries")
        .mockResolvedValue();
      const { result } = renderHook(() => useCorrectTransaction("p"), {
        wrapper: wrapper(client),
      });
      await act(async () => {
        await expect(
          result.current.mutateAsync({
            transactionId: transaction.id,
            command: { replacement: command },
            key: "correction-attempt-key",
          }),
        ).rejects.toMatchObject({ code });
      });
      expect(transactionApi.correct).toHaveBeenCalledTimes(1);
      if (code === "TRANSACTION_ALREADY_CORRECTED")
        expect(invalidate).toHaveBeenCalledWith({
          queryKey: transactionKeys.histories("p"),
        });
      else expect(invalidate).not.toHaveBeenCalled();
      client.clear();
    },
  );
  it("normalizes default keys and includes every filter and Portfolio scope", () => {
    expect(transactionKeys.history("p", {})).toEqual(
      transactionKeys.history("p", { limit: 50, includeReversals: true }),
    );
    for (const filter of [
      { kind: "BUY" as const },
      { effectiveAtFrom: "2020-01-01T00:00:00Z" },
      { effectiveAtTo: "2999-01-01T00:00:00Z" },
      { includeReversals: false },
      { limit: 100 },
    ])
      expect(transactionKeys.history("p", filter)).not.toEqual(
        transactionKeys.history("p", {}),
      );
    expect(transactionKeys.history("other", {})).not.toEqual(
      transactionKeys.history("p", {}),
    );
  });
  it("keeps exact cursor/server order and resets pages when filters change", async () => {
    vi.mocked(transactionApi.list)
      .mockResolvedValueOnce({
        items: [transaction],
        nextCursor: "opaque-cursor",
      })
      .mockResolvedValueOnce({
        items: [{ ...transaction, id: "older" }],
        nextCursor: null,
      })
      .mockResolvedValue({ items: [], nextCursor: null });
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result, rerender } = renderHook(
      ({ kind }: { kind?: "BUY" }) => useTransactionHistory("p", { kind }),
      { wrapper: wrapper(client), initialProps: {} },
    );
    await waitFor(() => expect(result.current.data?.pages).toHaveLength(1));
    await act(async () => {
      await result.current.fetchNextPage();
    });
    expect(transactionApi.list).toHaveBeenNthCalledWith(
      2,
      "memory-access",
      "p",
      { kind: undefined, limit: 50, cursor: "opaque-cursor" },
    );
    await waitFor(() =>
      expect(
        result.current.data?.pages.flatMap((page) =>
          page.items.map((row) => row.id),
        ),
      ).toEqual(["transaction-1", "older"]),
    );
    expect(result.current.hasNextPage).toBe(false);
    rerender({ kind: "BUY" });
    await waitFor(() =>
      expect(transactionApi.list).toHaveBeenLastCalledWith(
        "memory-access",
        "p",
        { kind: "BUY", limit: 50, cursor: undefined },
      ),
    );
    client.clear();
  });
  it("disables reads before authentication", () => {
    auth.state.status = "unauthenticated";
    const client = new QueryClient();
    renderHook(() => useTransactionHistory("p", {}), {
      wrapper: wrapper(client),
    });
    expect(transactionApi.list).not.toHaveBeenCalled();
    client.clear();
  });
  it("keeps the key through authenticated retry and does not wait for invalidation", async () => {
    auth.runAuthenticated.mockImplementation(async (operation) => {
      try {
        return await operation("expired");
      } catch {
        return operation("refreshed");
      }
    });
    vi.mocked(transactionApi.create)
      .mockRejectedValueOnce(new Error("expired"))
      .mockResolvedValue(transaction);
    const client = new QueryClient();
    const invalidate = vi
      .spyOn(client, "invalidateQueries")
      .mockReturnValue(new Promise(() => {}));
    const { result } = renderHook(() => useCreateTransaction("p"), {
      wrapper: wrapper(client),
    });
    await act(async () => {
      expect(
        await result.current.mutateAsync({
          command,
          key: "one-memory-attempt",
        }),
      ).toEqual(transaction);
    });
    expect(transactionApi.create).toHaveBeenNthCalledWith(
      1,
      "expired",
      "p",
      command,
      "one-memory-attempt",
    );
    expect(transactionApi.create).toHaveBeenNthCalledWith(
      2,
      "refreshed",
      "p",
      command,
      "one-memory-attempt",
    );
    expect(invalidate).toHaveBeenCalledWith({
      queryKey: transactionKeys.histories("p"),
    });
    client.clear();
  });
  it("overrides automatic mutation retry defaults", async () => {
    vi.mocked(transactionApi.create).mockRejectedValue(new Error("uncertain"));
    const client = new QueryClient({
      defaultOptions: { mutations: { retry: 3, retryDelay: 0 } },
    });
    const { result } = renderHook(() => useCreateTransaction("p"), {
      wrapper: wrapper(client),
    });
    await act(async () => {
      await expect(
        result.current.mutateAsync({ command, key: "one-memory-attempt" }),
      ).rejects.toThrow("uncertain");
    });
    expect(transactionApi.create).toHaveBeenCalledTimes(1);
    client.clear();
  });
});
