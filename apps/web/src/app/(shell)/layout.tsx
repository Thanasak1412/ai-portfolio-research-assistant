export default function ApplicationShell({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <div className="min-h-screen">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto max-w-5xl px-6 py-4 text-sm font-semibold">
          Portfolio Research Assistant
        </div>
      </header>
      {children}
    </div>
  );
}
