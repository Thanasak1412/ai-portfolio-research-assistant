import { ProtectedApplicationShell } from "@/features/auth/components/protected-application-shell";
import { RequireAuthenticated } from "@/features/auth/components/require-authenticated";

export default function ProtectedAppLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <RequireAuthenticated>
      <ProtectedApplicationShell>{children}</ProtectedApplicationShell>
    </RequireAuthenticated>
  );
}
