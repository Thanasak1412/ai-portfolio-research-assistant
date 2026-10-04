import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { expect, it, vi } from "vitest";
import type { AuthApi } from "@/features/auth/api/auth-api";
import {
  AuthSessionProvider,
  useAuthSession,
} from "@/features/auth/model/auth-session-provider";
import { transactionApi } from "@/features/transaction/api/transaction-api";
import { useCreateTransaction } from "@/features/transaction/model/transaction-queries";
import { command, transaction } from "@/features/transaction/test-fixtures";
import { ApiError } from "@/platform/api/api-error";

it("retains the financial command/key through real AuthSessionProvider single refresh retry", async () => {
  const access = {
    accessToken: "initial-memory-access",
    tokenType: "Bearer" as const,
    expiresIn: 900 as const,
  };
  const api: AuthApi = {
    register: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
    refresh: vi
      .fn()
      .mockResolvedValueOnce(access)
      .mockResolvedValue({ ...access, accessToken: "renewed-memory-access" }),
    me: vi.fn().mockResolvedValue({
      id: "user-1",
      email: "person@example.test",
      status: "active",
      createdAt: "2020-01-01T00:00:00Z",
      updatedAt: "2020-01-01T00:00:00Z",
    }),
  };
  const create = vi
    .spyOn(transactionApi, "create")
    .mockRejectedValueOnce(new ApiError(401, "ACCESS_TOKEN_INVALID", "Expired"))
    .mockResolvedValue(transaction);
  const client = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <AuthSessionProvider api={api}>{children}</AuthSessionProvider>
    </QueryClientProvider>
  );
  const { result, unmount } = renderHook(
    () => ({ session: useAuthSession(), create: useCreateTransaction("p") }),
    { wrapper },
  );
  await waitFor(() =>
    expect(result.current.session.state.status).toBe("authenticated"),
  );
  await act(async () => {
    expect(
      await result.current.create.mutateAsync({
        command,
        key: "single-semantic-attempt",
      }),
    ).toEqual(transaction);
  });
  expect(create).toHaveBeenNthCalledWith(
    1,
    "initial-memory-access",
    "p",
    command,
    "single-semantic-attempt",
  );
  expect(create).toHaveBeenNthCalledWith(
    2,
    "renewed-memory-access",
    "p",
    command,
    "single-semantic-attempt",
  );
  expect(api.refresh).toHaveBeenCalledTimes(2);
  unmount();
  client.clear();
  create.mockRestore();
});
