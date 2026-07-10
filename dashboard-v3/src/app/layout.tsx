import type { Metadata } from "next";
import { ThemeProvider } from "next-themes";
import { QueryProvider } from "@/lib/query-provider";
import { AccountProvider } from "@/lib/auth/account-context";
import "./globals.css";

export const metadata: Metadata = {
  title: "Castellan",
  description: "Usage-based API monetization gateway",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className="antialiased">
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <QueryProvider>
            <AccountProvider>{children}</AccountProvider>
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
