"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { AuthenticationLoading } from "@/features/auth/components/authentication-loading";
import { BootstrapRecovery } from "@/features/auth/components/bootstrap-recovery";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";

export function PublicAuthRoute({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const { state } = useAuthSession();
  const router = useRouter();

  useEffect(() => {
    if (state.status === "authenticated") router.replace("/app");
  }, [router, state.status]);

  if (state.status === "bootstrapping" || state.status === "authenticated") {
    return <AuthenticationLoading />;
  }
  if (state.status === "bootstrap-error") return <BootstrapRecovery />;
  return children;
}
