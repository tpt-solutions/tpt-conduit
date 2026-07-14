"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { theme } from "@/lib/theme";

export function Header() {
  const router = useRouter();
  const [loggingOut, setLoggingOut] = useState(false);

  async function handleLogout() {
    setLoggingOut(true);
    await fetch("/api/logout", { method: "POST" });
    router.push("/login");
    router.refresh();
  }

  return (
    <header className="top-nav">
      <div className="brand">
        {theme.logoUrl ? <img src={theme.logoUrl} alt={theme.brandName} /> : null}
        <span>{theme.brandName}</span>
      </div>
      <nav>
        <Link href="/tickets">Tickets</Link>
        <Link href="/runs">Runs</Link>
      </nav>
      <button className="btn" onClick={handleLogout} disabled={loggingOut}>
        {loggingOut ? "Signing out…" : "Sign out"}
      </button>
    </header>
  );
}
