import type { NextRequest } from "next/server";

import { proxyOnboarding } from "../../../lib/bff";

export const dynamic = "force-dynamic";

export function GET(request: NextRequest) {
  return proxyOnboarding(request, "GET");
}

export function PATCH(request: NextRequest) {
  return proxyOnboarding(request, "PATCH");
}
