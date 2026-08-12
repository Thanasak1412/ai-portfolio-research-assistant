"use client";

import { Button } from "@/components/ui/button";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";

export function BootstrapRecovery() {
  const { retryBootstrap } = useAuthSession();
  return (
    <main className="mx-auto max-w-3xl space-y-4 px-6 py-16">
      <p role="alert" className="text-sm text-red-800">
        Authentication is temporarily unavailable. Your session status could not
        be confirmed.
      </p>
      <Button type="button" onClick={retryBootstrap}>
        Retry
      </Button>
    </main>
  );
}
