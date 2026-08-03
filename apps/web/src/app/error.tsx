"use client";

import { useEffect } from "react";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error("Unhandled application error", { digest: error.digest });
  }, [error]);

  return (
    <main className="mx-auto max-w-xl space-y-4 p-8">
      <h1 className="text-3xl font-semibold">Something went wrong</h1>
      <p className="text-slate-600">
        The request could not be completed. No sensitive details were exposed.
      </p>
      <button
        className="rounded bg-slate-900 px-4 py-2 text-white"
        onClick={reset}
        type="button"
      >
        Try again
      </button>
    </main>
  );
}
