import type { Metadata } from "next";
import { theme } from "@/lib/theme";
import "./globals.css";

export const metadata: Metadata = {
  title: theme.brandName,
  description: `${theme.brandName} — workflow & ticketing console`,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body
        style={
          {
            "--color-primary": theme.colorPrimary,
            "--color-accent": theme.colorAccent,
          } as React.CSSProperties
        }
      >
        {children}
      </body>
    </html>
  );
}
