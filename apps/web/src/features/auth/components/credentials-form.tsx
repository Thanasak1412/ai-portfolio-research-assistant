"use client";

import Link from "next/link";
import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { type AuthApi, authApi } from "@/features/auth/api/auth-api";
import {
  AuthFormError,
  authErrorMessage,
} from "@/features/auth/components/auth-form-error";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";
import {
  credentialsSchema,
  normalizedCredentials,
  type CredentialsFormValues,
} from "@/features/auth/validation/credentials";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type CredentialsFormProps = { mode: "login" | "register"; api?: AuthApi };

export function CredentialsForm({
  mode,
  api = authApi,
}: Readonly<CredentialsFormProps>) {
  const { establishSession } = useAuthSession();
  const [submissionError, setSubmissionError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setValue,
  } = useForm<CredentialsFormValues>({
    resolver: zodResolver(credentialsSchema),
    defaultValues: { email: "", password: "" },
  });
  const isLogin = mode === "login";

  const submit = async (values: CredentialsFormValues) => {
    setSubmissionError(null);
    setSuccess(false);
    try {
      const response = isLogin
        ? await api.login(normalizedCredentials(values))
        : await api.register(normalizedCredentials(values));
      establishSession(response);
      setValue("password", "");
      setSuccess(true);
    } catch (error) {
      setValue("password", "");
      setSubmissionError(authErrorMessage(error, mode));
    }
  };

  const title = isLogin ? "Sign in" : "Create account";
  const alternate = isLogin
    ? { href: "/register", label: "Create an account" }
    : { href: "/login", label: "Sign in instead" };
  return (
    <section className="mx-auto w-full max-w-md space-y-6 rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
      <div>
        <h1 className="text-2xl font-semibold">{title}</h1>
        <p className="mt-1 text-sm text-slate-600">
          Use your email address and password.
        </p>
      </div>
      <form className="space-y-4" onSubmit={handleSubmit(submit)} noValidate>
        <div className="space-y-1">
          <label htmlFor={`${mode}-email`} className="text-sm font-medium">
            Email
          </label>
          <Input
            id={`${mode}-email`}
            type="email"
            autoComplete="username"
            aria-invalid={!!errors.email}
            aria-describedby={errors.email ? `${mode}-email-error` : undefined}
            {...register("email")}
          />
          {errors.email && (
            <p
              id={`${mode}-email-error`}
              role="alert"
              className="text-sm text-red-700"
            >
              {errors.email.message}
            </p>
          )}
        </div>
        <div className="space-y-1">
          <label htmlFor={`${mode}-password`} className="text-sm font-medium">
            Password
          </label>
          <Input
            id={`${mode}-password`}
            type="password"
            autoComplete={isLogin ? "current-password" : "new-password"}
            aria-invalid={!!errors.password}
            aria-describedby={
              errors.password ? `${mode}-password-error` : undefined
            }
            {...register("password")}
          />
          {errors.password && (
            <p
              id={`${mode}-password-error`}
              role="alert"
              className="text-sm text-red-700"
            >
              {errors.password.message}
            </p>
          )}
        </div>
        <AuthFormError message={submissionError} />
        {success && (
          <p
            role="status"
            aria-live="polite"
            className="rounded-md bg-emerald-50 p-3 text-sm text-emerald-800"
          >
            You are signed in.
          </p>
        )}
        <Button type="submit" disabled={isSubmitting} className="w-full">
          {isSubmitting ? "Submitting…" : title}
        </Button>
      </form>
      <p className="text-sm text-slate-600">
        <Link className="underline" href={alternate.href}>
          {alternate.label}
        </Link>
      </p>
    </section>
  );
}
