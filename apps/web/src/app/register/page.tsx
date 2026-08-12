import { RegisterForm } from "@/features/auth/components/register-form";
import { PublicAuthRoute } from "@/features/auth/components/public-auth-route";
export default function RegisterPage() {
  return (
    <PublicAuthRoute>
      <main className="px-6 py-16">
        <RegisterForm />
      </main>
    </PublicAuthRoute>
  );
}
