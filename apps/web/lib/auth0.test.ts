import { describe, expect, it } from "vitest";

import { readWebAuthConfig, WebAuthConfigurationError } from "./auth0";

const validEnvironment: NodeJS.ProcessEnv = {
  NODE_ENV: "test",
  AUTH0_DOMAIN: "tenant.example.auth0.com",
  AUTH0_CLIENT_ID: "web-client-id",
  AUTH0_CLIENT_SECRET: "test-client-secret-value",
  AUTH0_SECRET: "a".repeat(64),
  APP_BASE_URL: "http://127.0.0.1:3000",
  AUTH0_AUDIENCE: "https://api.example.invalid",
  API_BASE_URL: "http://127.0.0.1:8080",
};

describe("readWebAuthConfig", () => {
  it("accepts canonical local configuration without exposing secrets to public variables", () => {
    const config = readWebAuthConfig(validEnvironment);
    expect(config.appBaseUrl).toBe("http://127.0.0.1:3000");
    expect(config.apiBaseUrl).toBe("http://127.0.0.1:8080");
    expect(config.secureCookies).toBe(false);
    expect(Object.keys(validEnvironment)).not.toContain("NEXT_PUBLIC_AUTH0_CLIENT_SECRET");
  });

  it("fails closed when a required variable is missing", () => {
    expect(() => readWebAuthConfig({ ...validEnvironment, AUTH0_SECRET: "" })).toThrow(
      WebAuthConfigurationError,
    );
  });

  it("rejects non-hex cookie secrets and URL credentials", () => {
    expect(() => readWebAuthConfig({ ...validEnvironment, AUTH0_SECRET: "x".repeat(64) })).toThrow(
      "AUTH0_SECRET",
    );
    expect(() =>
      readWebAuthConfig({ ...validEnvironment, API_BASE_URL: "http://user:pass@127.0.0.1:8080" }),
    ).toThrow("canonical");
    expect(() =>
      readWebAuthConfig({ ...validEnvironment, APP_BASE_URL: "http://127.0.0.1:3000/" }),
    ).toThrow("canonical");
  });

  it("rejects malformed domains and whitespace-bearing client credentials", () => {
    expect(() =>
      readWebAuthConfig({ ...validEnvironment, AUTH0_DOMAIN: "-tenant.example.auth0.com" }),
    ).toThrow("AUTH0_DOMAIN");
    expect(() =>
      readWebAuthConfig({ ...validEnvironment, AUTH0_CLIENT_ID: "web client id" }),
    ).toThrow("AUTH0_CLIENT_ID");
    expect(() =>
      readWebAuthConfig({ ...validEnvironment, AUTH0_CLIENT_SECRET: "synthetic secret value" }),
    ).toThrow("AUTH0_CLIENT_SECRET");
    expect(() =>
      readWebAuthConfig({ ...validEnvironment, AUTH0_CLIENT_ID: " web-client-id" }),
    ).toThrow("AUTH0_CLIENT_ID");
    expect(() =>
      readWebAuthConfig({
        ...validEnvironment,
        AUTH0_CLIENT_SECRET: "__POPULATE_AUTH0_WEB_CLIENT_SECRET__",
      }),
    ).toThrow("AUTH0_CLIENT_SECRET");
  });

  it("requires HTTPS origins in production", () => {
    expect(() => readWebAuthConfig({ ...validEnvironment, NODE_ENV: "production" })).toThrow(
      "HTTPS",
    );
  });
});
