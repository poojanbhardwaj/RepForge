import { validateOIDCMobileConfiguration } from "./config";

const valid = {
  audience: "https://api.example.test",
  clientId: "mobile-client-id",
  domain: "tenant.example.auth0.com",
  mode: "oidc",
};

it("accepts a strict OIDC mobile configuration", () => {
  expect(() => validateOIDCMobileConfiguration(valid)).not.toThrow();
});

it.each([
  [{ ...valid, domain: "https://tenant.example.auth0.com" }, "AUTH0_DOMAIN"],
  [{ ...valid, domain: "-tenant.example.auth0.com" }, "AUTH0_DOMAIN"],
  [{ ...valid, clientId: "short" }, "AUTH0_CLIENT_ID"],
  [{ ...valid, clientId: "__POPULATE_AUTH0_MOBILE_CLIENT_ID__" }, "AUTH0_CLIENT_ID"],
  [{ ...valid, audience: "http://api.example.test" }, "AUTH0_AUDIENCE"],
  [{ ...valid, audience: "https://api.example.test?redirect=other" }, "AUTH0_AUDIENCE"],
] as const)("rejects invalid configuration %#", (configuration, expected) => {
  expect(() => validateOIDCMobileConfiguration(configuration)).toThrow(expected);
});

it("rejects unknown auth modes and permits the isolated development mode", () => {
  expect(() => validateOIDCMobileConfiguration({ ...valid, mode: "typo" })).toThrow("AUTH_MODE");
  expect(() => validateOIDCMobileConfiguration({ ...valid, mode: undefined })).toThrow("AUTH_MODE");
  expect(() =>
    validateOIDCMobileConfiguration({ audience: "", clientId: "", domain: "", mode: "dev" }),
  ).not.toThrow();
});
