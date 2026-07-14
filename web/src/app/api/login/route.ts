import { NextRequest, NextResponse } from "next/server";
import { AUTH_COOKIE, buildBasicAuthHeader } from "@/lib/auth";

const API_URL = process.env.CONDUIT_API_URL || "http://localhost:8080";

export async function POST(req: NextRequest) {
  const { username, password } = (await req.json()) as { username?: string; password?: string };
  if (!username || !password) {
    return NextResponse.json({ error: "Username and password are required" }, { status: 400 });
  }

  const authHeader = buildBasicAuthHeader(username, password);

  // Verify the credential against the backend before trusting it.
  const check = await fetch(`${API_URL}/graphql`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: authHeader },
    body: JSON.stringify({ query: "{ workflows { name } }" }),
    cache: "no-store",
  });

  if (check.status === 401) {
    return NextResponse.json({ error: "Invalid credentials" }, { status: 401 });
  }
  if (!check.ok) {
    return NextResponse.json({ error: "Backend unavailable" }, { status: 502 });
  }

  const res = NextResponse.json({ ok: true });
  res.cookies.set(AUTH_COOKIE, authHeader, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 24 * 7,
  });
  return res;
}
