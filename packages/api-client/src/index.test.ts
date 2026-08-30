import { afterEach, describe, expect, it, vi } from "vitest";

import { ApiError, createRepForgeClient } from "./index";

afterEach(() => vi.restoreAllMocks());

describe("RepForge API client", () => {
  it("adds the bearer token and returns a profile", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          id: "01993c86-8fc9-7a5a-9b8d-3302c7e917ec",
          profile: {
            displayName: "Local Athlete",
            version: 1,
            createdAt: "2026-08-28T12:00:00Z",
            updatedAt: "2026-08-28T12:00:00Z",
          },
          consents: [],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    const client = createRepForgeClient({
      baseUrl: "http://localhost:8080",
      getAccessToken: () => "synthetic-token",
    });

    const result = await client.getMe();

    expect(result.profile.displayName).toBe("Local Athlete");
    const request = fetchMock.mock.calls[0]?.[0] as Request;
    expect(request.method).toBe("GET");
    expect(request.url).toBe("http://localhost:8080/v1/me");
    expect(request.headers.get("Authorization")).toBe("Bearer synthetic-token");
  });

  it("normalizes an API error envelope", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: "version_conflict",
            message: "Refresh and retry.",
            traceId: "01993c86-8fc9-7a5a-9b8d-3302c7e917ec",
          },
        }),
        { status: 409, headers: { "Content-Type": "application/json" } },
      ),
    );
    const client = createRepForgeClient({
      baseUrl: "http://localhost:8080",
      getAccessToken: () => undefined,
    });

    await expect(
      client.updateMe({ displayName: "Athlete", expectedVersion: 1 }),
    ).rejects.toMatchObject({
      status: 409,
      code: "version_conflict",
    } satisfies Partial<ApiError>);
    const request = fetchMock.mock.calls[0]?.[0] as Request;
    expect(request.method).toBe("PATCH");
    expect(request.url).toBe("http://localhost:8080/v1/me");
  });

  it("awaits an OIDC token and forwards onboarding idempotency and nullable preferences", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ state: "in_progress" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const client = createRepForgeClient({
      baseUrl: "https://api.example.test",
      getAccessToken: async () => "synthetic-oidc-token",
    });

    await client.updateOnboarding(
      { currentStep: 2, dietPreference: null },
      "12345678-1234-4234-8234-123456789012",
    );

    const request = fetchMock.mock.calls[0]?.[0] as Request;
    expect(request.url).toBe("https://api.example.test/v1/onboarding");
    expect(request.headers.get("Authorization")).toBe("Bearer synthetic-oidc-token");
    expect(request.headers.get("Idempotency-Key")).toBe("12345678-1234-4234-8234-123456789012");
    expect(await request.json()).toMatchObject({ currentStep: 2, dietPreference: null });
  });
});
