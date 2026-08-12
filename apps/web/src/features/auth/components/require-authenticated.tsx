"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { AuthenticationLoading } from "@/features/auth/components/authentication-loading";
import { BootstrapRecovery } from "@/features/auth/components/bootstrap-recovery";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";

export function RequireAuthenticated({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const { state } = useAuthSession();
  const router = useRouter();

  useEffect(() => {
    if (state.status === "unauthenticated") router.replace("/login");
  }, [router, state.status]);

  if (state.status === "bootstrapping") return <AuthenticationLoading />;
  if (state.status === "bootstrap-error") return <BootstrapRecovery />;
  if (state.status !== "authenticated") return null;
  return children;
}
