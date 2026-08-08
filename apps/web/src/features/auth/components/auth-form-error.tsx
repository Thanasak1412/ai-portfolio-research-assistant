import { AuthApiError } from "@/features/auth/api/auth-api";

export function authErrorMessage(
  error: unknown,
  action: "login" | "register",
): string {
  if (error instanceof AuthApiError) {
    if (error.code === "RATE_LIMIT_EXCEEDED") {
      return error.retryAfterSeconds
        ? `Too many attempts. Try again in ${error.retryAfterSeconds} seconds.`
        : "Too many attempts. Try again shortly.";
    }
    if (action === "login" && error.code === "AUTHENTICATION_FAILED")
      return "Authentication failed. Check your email and password and try again.";
    if (action === "register" && error.code === "REGISTRATION_REJECTED")
      return "Registration could not be completed. Please try again.";
  }
  return "Authentication is temporarily unavailable. Please try again.";
}

export function AuthFormError({
  message,
}: Readonly<{ message: string | null }>) {
  if (!message) return null;
  return (
    <p
      role="alert"
      aria-live="polite"
      className="rounded-md bg-red-50 p-3 text-sm text-red-800"
    >
      {message}
    </p>
  );
}
