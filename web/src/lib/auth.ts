import { cookies } from "next/headers";

// The backend has no token-exchange endpoint (see api/auth.go) — it accepts a
// static Basic-auth credential or API key on every request. We authenticate
// the user once, then store the resulting `Authorization` header value in an
// httpOnly cookie so client-side JS never touches the credential directly.
export const AUTH_COOKIE = "conduit_auth";

export function buildBasicAuthHeader(username: string, password: string): string {
  const encoded = Buffer.from(`${username}:${password}`, "utf-8").toString("base64");
  return `Basic ${encoded}`;
}

/** Server-only: read the stored Authorization header value, if the user is logged in. */
export async function getAuthHeader(): Promise<string | null> {
  const value = (await cookies()).get(AUTH_COOKIE)?.value;
  return value || null;
}
