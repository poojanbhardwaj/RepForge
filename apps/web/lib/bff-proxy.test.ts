import { NextRequest, NextResponse } from "next/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getAccessToken: vi.fn(),
  getSession: vi.fn(),
}));

vi.mock("./auth0", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./auth0")>();
  return {
    ...actual,
    getAuth0: () => ({
      getAccessToken: mocks.getAccessToken,
      getSession: mocks.getSession,
    }),
    readWebAuthConfig: () => ({
      domain: "tenant.example.auth0.com",
      clientId: "web-client-id",
      clientSecret: "synthetic-client-secret",
      secret: "a".repeat(64),
      appBaseUrl: "http://127.0.0.1:3000",
      audience: "https://api.example.invalid",
      apiBaseUrl: "http://127.0.0.1:8080",
      secureCookies: false,
    }),
  };
});

import { proxyOnboarding } from "./bff";

describe("proxyOnboarding", () => {
  beforeEach(() => {
    mocks.getSession.mockReset().mockResolvedValue({ user: { sub: "synthetic-user" } });
    mocks.getAccessToken
      .mockReset()
      .mockImplementation((_request: NextRequest, response: NextResponse) => {
        response.cookies.set("appSession", "synthetic-refreshed-session", {
          httpOnly: true,
          sameSite: "lax",
        });
        return Promise.resolve({ token: "synthetic-access-token", expiresAt: 1_800_000_000 });
      });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects missing or unreadable server sessions before contacting the API", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    mocks.getSession.mockResolvedValueOnce(null);
    const missing = await proxyOnboarding(request("GET"), "GET");
    expect(missing.status).toBe(401);
    expect(await missing.json()).toMatchObject({ error: { code: "session_expired" } });

    mocks.getSession.mockRejectedValueOnce(new Error("synthetic corrupt session"));
    const unreadable = await proxyOnboarding(request("GET"), "GET");
    expect(unreadable.status).toBe(401);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("returns the defined status for an invalid idempotency key", async () => {
    vi.stubGlobal("fetch", vi.fn());
    const response = await proxyOnboarding(
      request("PATCH", { "idempotency-key": "short" }),
      "PATCH",
    );
    expect(response.status).toBe(400);
    expect(await response.json()).toMatchObject({ error: { code: "invalid_idempotency_key" } });
    expect(mocks.getAccessToken).not.toHaveBeenCalled();
  });

  it("forwards only the server token and exact idempotency key and persists refreshed cookies", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ state: "in_progress" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const response = await proxyOnboarding(request("PATCH"), "PATCH");

    expect(response.status).toBe(200);
    expect(response.headers.get("cache-control")).toBe("no-store");
    expect(response.headers.get("set-cookie")).toContain("HttpOnly");
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    const headers = new Headers(init.headers);
    expect(url).toBe("http://127.0.0.1:8080/v1/onboarding");
    expect(headers.get("Authorization")).toBe("Bearer synthetic-access-token");
    expect(headers.get("Idempotency-Key")).toBe("12345678-1234-4234-8234-123456789012");
    expect(init.cache).toBe("no-store");
  });

  it("retains a refreshed session cookie when the upstream API is unavailable", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("synthetic unavailable")));
    const response = await proxyOnboarding(request("GET"), "GET");
    expect(response.status).toBe(502);
    expect(response.headers.get("set-cookie")).toContain("HttpOnly");
    expect(await response.json()).toMatchObject({ error: { code: "api_unavailable" } });
  });

  it("rejects oversized requests and malformed upstream JSON", async () => {
    vi.stubGlobal("fetch", vi.fn());
    const oversized = await proxyOnboarding(
      request("PATCH", { "content-length": String(32 * 1024 + 1) }),
      "PATCH",
    );
    expect(oversized.status).toBe(413);

    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("not-json", {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const malformed = await proxyOnboarding(request("GET"), "GET");
    expect(malformed.status).toBe(502);
    expect(await malformed.json()).toMatchObject({
      error: { code: "invalid_upstream_response" },
    });
  });

  it("bounds a chunked upstream response before parsing it", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response("x".repeat(1024 * 1024 + 1), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const response = await proxyOnboarding(request("GET"), "GET");
    expect(response.status).toBe(502);
    expect(await response.json()).toMatchObject({
      error: { code: "invalid_upstream_response" },
    });
  });
});

function request(method: "GET" | "PATCH", headers: Record<string, string> = {}) {
  return new NextRequest("http://127.0.0.1:3000/api/onboarding", {
    method,
    headers: {
      ...(method === "PATCH"
        ? {
            origin: "http://127.0.0.1:3000",
            "sec-fetch-site": "same-origin",
            "content-type": "application/json",
            "idempotency-key": "12345678-1234-4234-8234-123456789012",
          }
        : {}),
      ...headers,
    },
    ...(method === "PATCH" ? { body: JSON.stringify({ currentStep: 2 }) } : {}),
  });
}
