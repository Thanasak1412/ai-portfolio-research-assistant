import { ProtectedApplicationShell } from "@/features/auth/components/protected-application-shell";
import { RequireAuthenticated } from "@/features/auth/components/require-authenticated";

export default function ProtectedAppPage() {
  return (
    <RequireAuthenticated>
      <ProtectedApplicationShell />
    </RequireAuthenticated>
  );
}
