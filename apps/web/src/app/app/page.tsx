import Link from "next/link";

import { Button } from "@/components/ui/button";

export default function ProtectedAppPage() {
  return (
    <main className="mx-auto max-w-3xl space-y-4 px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-semibold">Your workspace</h1>
      <p className="text-sm text-slate-600">
        Manage your Portfolio records from one protected workspace.
      </p>
      <Button asChild>
        <Link href="/app/portfolios">View Portfolios</Link>
      </Button>
    </main>
  );
}
