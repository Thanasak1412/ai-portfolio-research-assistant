"use client";

import { LogoutButton } from "@/features/auth/components/logout-button";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";

export function ProtectedApplicationShell() {
  const { state } = useAuthSession();
  if (state.status !== "authenticated") return null;
  return (
    <main className="mx-auto max-w-3xl space-y-6 px-6 py-16">
      <div>
        <h1 className="text-2xl font-semibold">Portfolio Research Assistant</h1>
        <p className="mt-2 text-sm text-slate-600">
          Signed in as {state.session.user.email}
        </p>
      </div>
      <p className="text-sm text-slate-600">
        Your authenticated workspace is ready. Portfolio functionality is not
        part of this milestone.
      </p>
      <LogoutButton />
    </main>
  );
}
