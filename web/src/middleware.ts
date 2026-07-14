import { NextRequest, NextResponse } from "next/server";
import { AUTH_COOKIE } from "@/lib/auth";

export function middleware(req: NextRequest) {
  const isLoggedIn = Boolean(req.cookies.get(AUTH_COOKIE)?.value);
  const { pathname } = req.nextUrl;

  if (!isLoggedIn && pathname !== "/login") {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("next", pathname);
    return NextResponse.redirect(url);
  }

  if (isLoggedIn && pathname === "/login") {
    const url = req.nextUrl.clone();
    url.pathname = "/";
    url.search = "";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

export const config = {
  // Route handlers under /api manage their own auth (401 JSON) — only guard pages here.
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};
