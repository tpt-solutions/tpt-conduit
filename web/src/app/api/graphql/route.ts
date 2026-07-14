import { NextRequest, NextResponse } from "next/server";
import { getAuthHeader } from "@/lib/auth";

const API_URL = process.env.CONDUIT_API_URL || "http://localhost:8080";

// Same-origin proxy so client components can call GraphQL without ever
// seeing the stored credential — it's attached here from the httpOnly cookie.
export async function POST(req: NextRequest) {
  const authHeader = getAuthHeader();
  if (!authHeader) {
    return NextResponse.json({ errors: [{ message: "Not authenticated" }] }, { status: 401 });
  }

  const body = await req.text();
  const res = await fetch(`${API_URL}/graphql`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: authHeader },
    body,
    cache: "no-store",
  });

  const text = await res.text();
  return new NextResponse(text, {
    status: res.status === 401 ? 401 : 200,
    headers: { "Content-Type": "application/json" },
  });
}
