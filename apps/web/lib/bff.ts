import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

import { getAuth0, readWebAuthConfig, WebAuthConfigurationError } from "./auth0";

const maximumBodyBytes = 32 * 1024;
const maximumResponseBytes = 1024 * 1024;
const idempotencyKeyPattern = /^[A-Za-z0-9._:-]{16,128}$/;

export function validateMutationRequest(request: Pick<NextRequest, "headers">): string | null {
  const config = readWebAuthConfig();
  if (request.headers.get("origin") !== config.appBaseUrl) {
    return "origin_not_allowed";
  }
  const fetchSite = request.headers.get("sec-fetch-site");
  if (fetchSite && fetchSite !== "same-origin") {
    return "cross_site_request";
  }
  const contentType = request.headers.get("content-type")?.split(";", 1)[0]?.trim();
  if (contentType !== "application/json") {
    return "unsupported_media_type";
  }
  const key = request.headers.get("idempotency-key") ?? "";
  if (!idempotencyKeyPattern.test(key)) {
    return "invalid_idempotency_key";
  }
  return null;
}

export async function proxyOnboarding(request: NextRequest, method: "GET" | "PATCH") {
  let auth0: ReturnType<typeof getAuth0>;
  try {
    auth0 = getAuth0();
  } catch (error) {
    if (error instanceof WebAuthConfigurationError) {
      return bffError(503, "authentication_unavailable", "Authentication is unavailable.");
    }
    throw error;
  }
  let session;
  try {
    session = await auth0.getSession(request);
  } catch {
    return bffError(401, "session_invalid", "Log out and sign in again to continue.");
  }
  if (!session) return bffError(401, "session_expired", "Sign in again to continue.");

  let body: string | undefined;
  if (method === "PATCH") {
    let invalid: string | null;
    try {
      invalid = validateMutationRequest(request);
    } catch (error) {
      if (error instanceof WebAuthConfigurationError) {
        return bffError(503, "authentication_unavailable", "Authentication is unavailable.");
      }
      throw error;
    }
    if (invalid)
      return bffError(
        invalid === "unsupported_media_type"
          ? 415
          : invalid === "invalid_idempotency_key"
            ? 400
            : 403,
        invalid,
        "The request was rejected.",
      );
    const declaredLength = request.headers.get("content-length");
    if (
      declaredLength &&
      (!/^\d+$/.test(declaredLength) || Number(declaredLength) > maximumBodyBytes)
    ) {
      return bffError(413, "request_too_large", "The request is too large.");
    }
  }

  const cookieResponse = new NextResponse(null);
  let token: string;
  try {
    ({ token } = await auth0.getAccessToken(request, cookieResponse));
  } catch (error) {
    return withCookies(accessTokenFailure(error), cookieResponse);
  }

  if (method === "PATCH") {
    body = await request.text();
    if (new TextEncoder().encode(body).byteLength > maximumBodyBytes) {
      return withCookies(
        bffError(413, "request_too_large", "The request is too large."),
        cookieResponse,
      );
    }
    try {
      const parsed: unknown = JSON.parse(body);
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error();
      body = JSON.stringify(parsed);
    } catch {
      return withCookies(
        bffError(400, "invalid_json", "The request body must be a JSON object."),
        cookieResponse,
      );
    }
  }

  let config;
  try {
    config = readWebAuthConfig();
  } catch (error) {
    if (error instanceof WebAuthConfigurationError) {
      return withCookies(
        bffError(503, "authentication_unavailable", "Authentication is unavailable."),
        cookieResponse,
      );
    }
    throw error;
  }
  let upstream: Response;
  try {
    upstream = await fetch(`${config.apiBaseUrl}/v1/onboarding`, {
      method,
      cache: "no-store",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
        ...(method === "PATCH"
          ? {
              "Content-Type": "application/json",
              "Idempotency-Key": request.headers.get("idempotency-key")!,
              Origin: config.appBaseUrl,
            }
          : {}),
      },
      ...(body ? { body } : {}),
      signal: AbortSignal.timeout(10_000),
    });
  } catch {
    return withCookies(
      bffError(502, "api_unavailable", "The service is temporarily unavailable."),
      cookieResponse,
    );
  }

  const declaredResponseLength = upstream.headers.get("content-length");
  if (
    declaredResponseLength &&
    (!/^\d+$/.test(declaredResponseLength) || Number(declaredResponseLength) > maximumResponseBytes)
  ) {
    return withCookies(
      bffError(502, "invalid_upstream_response", "The service returned an invalid response."),
      cookieResponse,
    );
  }
  const responseMediaType = upstream.headers
    .get("content-type")
    ?.split(";", 1)[0]
    ?.trim()
    .toLowerCase();
  const responseBody = await readBoundedResponse(upstream, maximumResponseBytes);
  if (responseBody === null || responseMediaType !== "application/json") {
    return withCookies(
      bffError(502, "invalid_upstream_response", "The service returned an invalid response."),
      cookieResponse,
    );
  }
  try {
    const parsed: unknown = JSON.parse(responseBody);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error();
  } catch {
    return withCookies(
      bffError(502, "invalid_upstream_response", "The service returned an invalid response."),
      cookieResponse,
    );
  }
  const outgoing = new NextResponse(responseBody, {
    status: upstream.status,
    headers: { "Content-Type": "application/json", "Cache-Control": "no-store" },
  });
  return withCookies(outgoing, cookieResponse);
}

async function readBoundedResponse(
  response: Response,
  maximumBytes: number,
): Promise<string | null> {
  if (!response.body) return "";
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let received = 0;
  let result = "";
  try {
    while (true) {
      const chunk = await reader.read();
      if (chunk.done) return result + decoder.decode();
      received += chunk.value.byteLength;
      if (received > maximumBytes) {
        await reader.cancel();
        return null;
      }
      result += decoder.decode(chunk.value, { stream: true });
    }
  } finally {
    reader.releaseLock();
  }
}

function withCookies(response: NextResponse, cookieSource: NextResponse): NextResponse {
  for (const cookie of cookieSource.cookies.getAll()) response.cookies.set(cookie);
  return response;
}

function bffError(status: number, code: string, message: string) {
  return NextResponse.json(
    { error: { code, message } },
    { status, headers: { "Cache-Control": "no-store" } },
  );
}

function accessTokenFailure(error: unknown): NextResponse {
  const code =
    error && typeof error === "object" && "code" in error && typeof error.code === "string"
      ? error.code
      : null;
  if (code === "missing_session" || code === "session_expired") {
    return bffError(401, "session_expired", "Sign in again to continue.");
  }
  if (code === "missing_refresh_token") {
    return bffError(
      401,
      "reauthentication_required",
      "Log out and sign in again to renew API access.",
    );
  }
  return bffError(503, "authentication_unavailable", "Authentication is unavailable.");
}
