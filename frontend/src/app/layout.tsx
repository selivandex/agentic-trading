/** @format */

import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "@/styles/globals.css";
import { AuthProvider } from "@/features/auth/lib";
import { ToastProvider } from "@/shared/lib/toast-provider";
import { GlobalErrorBoundary } from "@/shared/ui/error-boundary/global-error-boundary";
import { ApolloProvider } from "@/shared/api";
import {
  ContextMenuProvider,
  ContextMenuRenderer,
} from "@/shared/ui/context-menu";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Prometheus Trading - AI-Powered Trading Platform",
  description: "Autonomous AI-driven hedge fund operating 24/7 at scale",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <GlobalErrorBoundary>
          <AuthProvider>
            <ApolloProvider>
              <ContextMenuProvider>
                {children}
                <ToastProvider />
                <ContextMenuRenderer />
              </ContextMenuProvider>
            </ApolloProvider>
          </AuthProvider>
        </GlobalErrorBoundary>
      </body>
    </html>
  );
}
