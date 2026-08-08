import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AuthApi } from "@/features/auth/api/auth-api";
import { AuthApiError } from "@/features/auth/api/auth-api";
import { CredentialsForm } from "@/features/auth/components/credentials-form";
import {
  AuthSessionProvider,
  useAuthSession,
} from "@/features/auth/model/auth-session-provider";

const validCredential = "x".repeat(12);

const response = {
  accessToken: "opaque-token",
  tokenType: "Bearer" as const,
  expiresIn: 900 as const,
  user: {
    id: "user_opaque",
    email: "person@example.com",
    status: "active" as const,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  },
};
function fakeApi(overrides: Partial<AuthApi> = {}): AuthApi {
  return {
    register: vi.fn().mockResolvedValue(response),
    login: vi.fn().mockResolvedValue(response),
    refresh: vi.fn(),
    logout: vi.fn(),
    me: vi.fn(),
    ...overrides,
  };
}
function SessionProbe() {
  const { session } = useAuthSession();
  return <output>{session?.user.email ?? "anonymous"}</output>;
}
function renderForm(mode: "login" | "register", api: AuthApi) {
  return render(
    <AuthSessionProvider>
      <CredentialsForm mode={mode} api={api} />
      <SessionProbe />
    </AuthSessionProvider>,
  );
}

describe("CredentialsForm", () => {
  it("validates login fields and establishes an in-memory session", async () => {
    const api = fakeApi();
    renderForm("login", api);
    const email = screen.getByLabelText("Email");
    const password = screen.getByLabelText("Password");
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByText("Email is required")).toBeInTheDocument();
    fireEvent.change(email, { target: { value: "invalid" } });
    fireEvent.change(password, { target: { value: "short" } });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(
      await screen.findByText("Enter a valid email address"),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("Password must be at least 12 characters"),
    ).toBeInTheDocument();
    fireEvent.change(email, { target: { value: " person@example.com " } });
    fireEvent.change(password, {
      target: { value: validCredential },
    });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    await waitFor(() =>
      expect(api.login).toHaveBeenCalledWith({
        email: "person@example.com",
        password: validCredential,
      }),
    );
    expect(await screen.findByText("You are signed in.")).toBeInTheDocument();
    expect(screen.getByText("person@example.com")).toBeInTheDocument();
    expect(password).toHaveValue("");
  });

  it("keeps login failures generic and clears the password", async () => {
    const api = fakeApi({
      login: vi
        .fn()
        .mockRejectedValue(
          new AuthApiError(401, "AUTHENTICATION_FAILED", "internal detail"),
        ),
    });
    renderForm("login", api);
    fireEvent.change(screen.getByLabelText("Email"), {
      target: { value: "person@example.com" },
    });
    const password = screen.getByLabelText("Password");
    fireEvent.change(password, {
      target: { value: validCredential },
    });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Authentication failed. Check your email and password and try again.",
    );
    expect(
      screen.queryByText(/account does not exist|incorrect password|disabled/i),
    ).not.toBeInTheDocument();
    expect(password).toHaveValue("");
  });

  it("shows safe rate-limit feedback and submits registration with no extra fields", async () => {
    const api = fakeApi({
      register: vi
        .fn()
        .mockRejectedValue(
          new AuthApiError(
            429,
            "RATE_LIMIT_EXCEEDED",
            "internal",
            undefined,
            15,
          ),
        ),
    });
    renderForm("register", api);
    fireEvent.change(screen.getByLabelText("Email"), {
      target: { value: "person@example.com" },
    });
    const password = screen.getByLabelText("Password");
    fireEvent.change(password, {
      target: { value: validCredential },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Too many attempts. Try again in 15 seconds.",
    );
    expect(api.register).toHaveBeenCalledWith({
      email: "person@example.com",
      password: validCredential,
    });
    expect(password).toHaveValue("");
  });

  it("establishes a registration session and keeps rejection generic", async () => {
    const successApi = fakeApi();
    const { unmount } = renderForm("register", successApi);
    fireEvent.change(screen.getByLabelText("Email"), {
      target: { value: "person@example.com" },
    });
    const password = screen.getByLabelText("Password");
    fireEvent.change(password, {
      target: { value: validCredential },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByText("You are signed in.")).toBeInTheDocument();
    expect(screen.getByText("person@example.com")).toBeInTheDocument();
    expect(password).toHaveValue("");
    unmount();

    const rejectionApi = fakeApi({
      register: vi
        .fn()
        .mockRejectedValue(
          new AuthApiError(409, "REGISTRATION_REJECTED", "internal detail"),
        ),
    });
    renderForm("register", rejectionApi);
    fireEvent.change(screen.getByLabelText("Email"), {
      target: { value: "person@example.com" },
    });
    fireEvent.change(screen.getByLabelText("Password"), {
      target: { value: validCredential },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Registration could not be completed. Please try again.",
    );
    expect(screen.queryByText(/already registered/i)).not.toBeInTheDocument();
  });

  it("rejects a password over the approved UTF-8 byte limit", async () => {
    const api = fakeApi();
    renderForm("register", api);
    fireEvent.change(screen.getByLabelText("Email"), {
      target: { value: "person@example.com" },
    });
    fireEvent.change(screen.getByLabelText("Password"), {
      target: { value: "😀".repeat(257) },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create account" }));
    expect(
      await screen.findByText("Password must not exceed 1,024 bytes"),
    ).toBeInTheDocument();
    expect(api.register).not.toHaveBeenCalled();
  });
});
