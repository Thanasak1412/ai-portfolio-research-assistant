"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { useAuthSession } from "@/features/auth/model/auth-session-provider";

export function LogoutButton() {
  const { logout } = useAuthSession();
  const router = useRouter();
  const [pending, setPending] = useState(false);

  const signOut = async () => {
    if (pending) return;
    setPending(true);
    await logout();
    router.replace("/login");
  };

  return (
    <Button type="button" disabled={pending} onClick={signOut}>
      {pending ? "Signing out…" : "Sign out"}
    </Button>
  );
}
