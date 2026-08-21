"use client";

import Link from "next/link";

import { LogoutButton } from "@/features/auth/components/logout-button";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";

export function ProtectedApplicationShell({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const { state } = useAuthSession();
  if (state.status !== "authenticated") return null;
  return (
    <div>
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-3xl flex-wrap items-center justify-between gap-3 px-4 py-4 sm:px-6">
          <div>
            <p className="font-semibold">Portfolio Research Assistant</p>
            <p className="text-xs text-slate-600">
              Signed in as {state.session.user.email}
            </p>
          </div>
          <nav aria-label="Application">
            <div className="flex items-center gap-3">
              <Link className="text-sm underline" href="/app">
                Home
              </Link>
              <Link className="text-sm underline" href="/app/portfolios">
                Portfolios
              </Link>
              <LogoutButton />
            </div>
          </nav>
        </div>
      </header>
      {children}
    </div>
  );
}
