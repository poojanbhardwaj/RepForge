import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { validateMutationRequest } from "./bff";

const originalEnvironment = { ...process.env };

describe("validateMutationRequest", () => {
  beforeEach(() => {
    Object.assign(process.env, {
      NODE_ENV: "test",
      AUTH0_DOMAIN: "tenant.example.auth0.com",
      AUTH0_CLIENT_ID: "web-client-id",
      AUTH0_CLIENT_SECRET: "test-client-secret-value",
      AUTH0_SECRET: "a".repeat(64),
      APP_BASE_URL: "http://127.0.0.1:3000",
      AUTH0_AUDIENCE: "https://api.example.invalid",
      API_BASE_URL: "http://127.0.0.1:8080",
    });
  });

  afterEach(() => {
    process.env = { ...originalEnvironment };
  });

  it("accepts an exact-origin JSON mutation with a safe idempotency key", () => {
    const headers = new Headers({
      origin: "http://127.0.0.1:3000",
      "sec-fetch-site": "same-origin",
      "content-type": "application/json; charset=utf-8",
      "idempotency-key": "12345678-1234-4234-8234-123456789012",
    });
    expect(validateMutationRequest({ headers })).toBeNull();
  });

  it("rejects missing or reflected origins", () => {
    const base = {
      "content-type": "application/json",
      "idempotency-key": "12345678-1234-4234-8234-123456789012",
    };
    expect(validateMutationRequest({ headers: new Headers(base) })).toBe("origin_not_allowed");
    expect(
      validateMutationRequest({
        headers: new Headers({ ...base, origin: "http://localhost:3000" }),
      }),
    ).toBe("origin_not_allowed");
  });

  it("rejects cross-site requests, unsafe keys, and non-JSON bodies", () => {
    const base = {
      origin: "http://127.0.0.1:3000",
      "sec-fetch-site": "cross-site",
      "content-type": "application/json",
      "idempotency-key": "12345678-1234-4234-8234-123456789012",
    };
    expect(validateMutationRequest({ headers: new Headers(base) })).toBe("cross_site_request");
    expect(
      validateMutationRequest({
        headers: new Headers({
          ...base,
          "sec-fetch-site": "same-origin",
          "idempotency-key": "short",
        }),
      }),
    ).toBe("invalid_idempotency_key");
    expect(
      validateMutationRequest({
        headers: new Headers({
          ...base,
          "sec-fetch-site": "same-origin",
          "content-type": "text/plain",
        }),
      }),
    ).toBe("unsupported_media_type");
  });
});
