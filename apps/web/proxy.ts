import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

import { getAuth0, WebAuthConfigurationError } from "./lib/auth0";

export async function proxy(request: NextRequest) {
  try {
    return await getAuth0().middleware(request);
  } catch (error) {
    const protectedPath =
      request.nextUrl.pathname.startsWith("/auth/") ||
      request.nextUrl.pathname.startsWith("/app") ||
      request.nextUrl.pathname.startsWith("/api/");
    if (error instanceof WebAuthConfigurationError && !protectedPath) {
      return NextResponse.next();
    }
    return NextResponse.json(
      { error: { code: "authentication_unavailable", message: "Authentication is unavailable." } },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)"],
};
