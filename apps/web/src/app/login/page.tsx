import { LoginForm } from "@/features/auth/components/login-form";
import { PublicAuthRoute } from "@/features/auth/components/public-auth-route";
export default function LoginPage() {
  return (
    <PublicAuthRoute>
      <main className="px-6 py-16">
        <LoginForm />
      </main>
    </PublicAuthRoute>
  );
}
