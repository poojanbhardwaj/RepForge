import {
  isCanonicalHTTPSOrigin,
  isLiteralLoopbackHTTPOrigin,
  resolveDevelopmentAccessToken,
  validateOIDCApiOrigin,
} from "./api";

describe("mobile development credential origin", () => {
  it.each([
    "http://127.0.0.1:8080",
    "http://127.20.30.40",
    "http://[::1]:8080",
    "http://[0:0:0:0:0:0:0:1]:8080/",
  ])("accepts literal loopback HTTP origin %s", (apiOrigin) => {
    expect(isLiteralLoopbackHTTPOrigin(apiOrigin)).toBe(true);
    expect(
      resolveDevelopmentAccessToken({
        apiOrigin,
        authMode: "dev",
        token: "not-a-secret",
        isDevelopment: true,
      }),
    ).toBe("not-a-secret");
  });

  it.each([
    "https://127.0.0.1:8080",
    "http://192.0.2.10:8080",
    "http://0.0.0.0:8080",
    "http://localhost:8080",
    "http://user@127.0.0.1:8080",
    "http://2130706433:8080",
    "http://127.0.0.1:8080/v1",
    "not-an-origin",
  ])("rejects %s before exposing the development token", (apiOrigin) => {
    expect(isLiteralLoopbackHTTPOrigin(apiOrigin)).toBe(false);
    expect(() =>
      resolveDevelopmentAccessToken({
        apiOrigin,
        authMode: "dev",
        token: "not-a-secret",
        isDevelopment: true,
      }),
    ).toThrow(/literal loopback IP/);
  });

  it("rejects development credentials in a non-development build", () => {
    expect(() =>
      resolveDevelopmentAccessToken({
        apiOrigin: "http://127.0.0.1:8080",
        authMode: "dev",
        token: "not-a-secret",
        isDevelopment: false,
      }),
    ).toThrow(/non-development mobile build/);
  });

  it("does not require a local origin when development credentials are inactive", () => {
    expect(
      resolveDevelopmentAccessToken({
        apiOrigin: "https://api.example.test",
        authMode: undefined,
        token: undefined,
        isDevelopment: false,
      }),
    ).toBeUndefined();
  });

  it("rejects missing, misplaced, or mixed development credentials", () => {
    expect(() =>
      resolveDevelopmentAccessToken({
        apiOrigin: "http://127.0.0.1:8080",
        authMode: "dev",
        token: undefined,
        isDevelopment: true,
      }),
    ).toThrow(/synthetic token/);
    expect(() =>
      resolveDevelopmentAccessToken({
        apiOrigin: "https://api.example.test",
        authMode: "oidc",
        token: "not-a-secret",
        isDevelopment: true,
      }),
    ).toThrow(/removed in OIDC/);
    expect(() =>
      resolveDevelopmentAccessToken({
        apiOrigin: "http://127.0.0.1:8080",
        authMode: "typo",
        token: undefined,
        isDevelopment: true,
      }),
    ).toThrow(/AUTH_MODE/);
  });
});

describe("mobile OIDC API origin", () => {
  it.each(["https://api.example.test", "https://api.example.test:8443/"])(
    "accepts canonical HTTPS origin %s",
    (apiOrigin) => {
      expect(isCanonicalHTTPSOrigin(apiOrigin)).toBe(true);
      expect(() => validateOIDCApiOrigin({ apiOrigin, isDevelopment: false })).not.toThrow();
    },
  );

  it("allows literal loopback HTTP only for development builds", () => {
    expect(() =>
      validateOIDCApiOrigin({ apiOrigin: "http://127.0.0.1:8080", isDevelopment: true }),
    ).not.toThrow();
    expect(() =>
      validateOIDCApiOrigin({ apiOrigin: "http://127.0.0.1:8080", isDevelopment: false }),
    ).toThrow(/HTTPS API origin/);
  });

  it.each([
    "http://192.0.2.10:8080",
    "https://user@api.example.test",
    "https://api.example.test/v1",
    "https://api.example.test?target=other",
    " https://api.example.test",
  ])("rejects unsafe OIDC API origin %s", (apiOrigin) => {
    expect(() => validateOIDCApiOrigin({ apiOrigin, isDevelopment: true })).toThrow(
      /HTTPS API origin/,
    );
  });
});
