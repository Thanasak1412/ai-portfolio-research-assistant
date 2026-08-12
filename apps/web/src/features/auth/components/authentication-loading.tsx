export function AuthenticationLoading() {
  return (
    <main
      aria-busy="true"
      aria-live="polite"
      className="mx-auto max-w-3xl px-6 py-16 text-sm text-slate-600"
    >
      Restoring your secure session…
    </main>
  );
}
