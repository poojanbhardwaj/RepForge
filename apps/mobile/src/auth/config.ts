export const mobileAuthMode = process.env.EXPO_PUBLIC_AUTH_MODE;
export const auth0Domain = process.env.EXPO_PUBLIC_AUTH0_DOMAIN ?? "";
export const auth0ClientId = process.env.EXPO_PUBLIC_AUTH0_CLIENT_ID ?? "";
export const auth0Audience = process.env.EXPO_PUBLIC_AUTH0_AUDIENCE ?? "";
export const auth0CustomScheme = "repforge";
export const auth0Scopes = "openid profile email offline_access profile:read profile:write";

export interface MobileAuthConfiguration {
  audience: string;
  clientId: string;
  domain: string;
  mode: string | undefined;
}

export function validateOIDCMobileConfiguration(
  configuration: MobileAuthConfiguration = {
    audience: auth0Audience,
    clientId: auth0ClientId,
    domain: auth0Domain,
    mode: mobileAuthMode,
  },
): void {
  if (!["dev", "oidc"].includes(configuration.mode ?? "")) {
    throw new Error("EXPO_PUBLIC_AUTH_MODE must be dev or oidc.");
  }
  if (configuration.mode !== "oidc") return;
  const { audience, clientId, domain } = configuration;
  if (
    !/^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(
      domain,
    )
  ) {
    throw new Error("EXPO_PUBLIC_AUTH0_DOMAIN must be an Auth0 hostname.");
  }
  if (
    clientId.length < 8 ||
    clientId.length > 256 ||
    /\s/.test(clientId) ||
    clientId.startsWith("__POPULATE_")
  ) {
    throw new Error("EXPO_PUBLIC_AUTH0_CLIENT_ID is invalid in OIDC mode.");
  }
  if (!isCanonicalHTTPSAudience(audience)) {
    throw new Error(
      "EXPO_PUBLIC_AUTH0_AUDIENCE must be a canonical HTTPS identifier in OIDC mode.",
    );
  }
}

function isCanonicalHTTPSAudience(value: string): boolean {
  if (!value || value.trim() !== value) return false;
  try {
    const parsed = new URL(value);
    return (
      parsed.protocol === "https:" &&
      parsed.username === "" &&
      parsed.password === "" &&
      parsed.search === "" &&
      parsed.hash === "" &&
      value === parsed.toString().replace(/\/$/, "")
    );
  } catch {
    return false;
  }
}
