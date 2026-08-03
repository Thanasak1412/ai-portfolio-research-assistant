import Link from "next/link";

export default function NotFound() {
  return (
    <main className="mx-auto max-w-xl space-y-4 p-8">
      <h1 className="text-3xl font-semibold">Page not found</h1>
      <p className="text-slate-600">
        The requested route is not part of the current bootstrap.
      </p>
      <Link className="underline" href="/">
        Return to the application foundation
      </Link>
    </main>
  );
}
