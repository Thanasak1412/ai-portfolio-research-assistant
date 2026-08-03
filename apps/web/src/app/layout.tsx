import type { Metadata } from "next";

import { clientEnvironment } from "@/platform/config/client-environment";
import { ApplicationProviders } from "@/platform/providers/application-providers";

import "./globals.css";

export const metadata: Metadata = {
  title: "Portfolio Research Assistant",
  description: "Engineering bootstrap for the portfolio research platform.",
};

void clientEnvironment;

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>
        <ApplicationProviders>{children}</ApplicationProviders>
      </body>
    </html>
  );
}
