import { createRepForgeClient, type RepForgeClient } from "@repforge/api-client";

const baseUrl = process.env.EXPO_PUBLIC_API_URL ?? "http://127.0.0.1:8080";
const authMode = process.env.EXPO_PUBLIC_AUTH_MODE;
const localToken = process.env.EXPO_PUBLIC_DEV_AUTH_TOKEN;

function literalLoopbackAuthority(raw: string, parsed: URL): boolean {
  const schemeSeparator = raw.indexOf("://");
  if (schemeSeparator < 0) return false;
  const remainder = raw.slice(schemeSeparator + 3);
  const authorityEnd = remainder.search(/[/?#]/);
  const authority = authorityEnd < 0 ? remainder : remainder.slice(0, authorityEnd);
  if (!authority || authority.includes("@")) return false;
  if (authority.startsWith("[")) {
    const closingBracket = authority.indexOf("]");
    if (closingBracket < 0) return false;
    const suffix = authority.slice(closingBracket + 1);
    if (suffix && !/^:\d+$/.test(suffix)) return false;
    return parsed.hostname.toLowerCase() === "[::1]";
  }
  const colon = authority.lastIndexOf(":");
  const host = colon < 0 ? authority : authority.slice(0, colon);
  if (colon >= 0 && !/^\d+$/.test(authority.slice(colon + 1))) return false;
  const octets = host.split(".");
  return (
    octets.length === 4 &&
    octets.every(
      (octet) => /^\d{1,3}$/.test(octet) && Number(octet) <= 255 && String(Number(octet)) === octet,
    ) &&
    Number(octets[0]) === 127
  );
}

export function isLiteralLoopbackHTTPOrigin(raw: string): boolean {
  if (!raw || raw.trim() !== raw) return false;
  try {
    const parsed = new URL(raw);
    return (
      parsed.protocol === "http:" &&
      parsed.username === "" &&
      parsed.password === "" &&
      parsed.pathname === "/" &&
      parsed.search === "" &&
      parsed.hash === "" &&
      literalLoopbackAuthority(raw, parsed)
    );
  } catch {
    return false;
  }
}

export function isCanonicalHTTPSOrigin(raw: string): boolean {
  if (!raw || raw.trim() !== raw) return false;
  try {
    const parsed = new URL(raw);
    return (
      parsed.protocol === "https:" &&
      parsed.username === "" &&
      parsed.password === "" &&
      parsed.pathname === "/" &&
      parsed.search === "" &&
      parsed.hash === "" &&
      (raw === parsed.origin || raw === `${parsed.origin}/`)
    );
  } catch {
    return false;
  }
}

export function validateOIDCApiOrigin({
  apiOrigin,
  isDevelopment,
}: {
  apiOrigin: string;
  isDevelopment: boolean;
}): void {
  if (isCanonicalHTTPSOrigin(apiOrigin)) return;
  if (isDevelopment && isLiteralLoopbackHTTPOrigin(apiOrigin)) return;
  throw new Error(
    "OIDC authentication requires a canonical HTTPS API origin, except for literal loopback HTTP during development.",
  );
}

interface DevelopmentAccessTokenOptions {
  apiOrigin: string;
  authMode: string | undefined;
  token: string | undefined;
  isDevelopment: boolean;
}

export function resolveDevelopmentAccessToken({
  apiOrigin,
  authMode: configuredAuthMode,
  token,
  isDevelopment,
}: DevelopmentAccessTokenOptions): string | undefined {
  if (configuredAuthMode !== undefined && !["dev", "oidc"].includes(configuredAuthMode)) {
    throw new Error("EXPO_PUBLIC_AUTH_MODE must be dev or oidc.");
  }
  if (configuredAuthMode === "oidc") {
    if (token) {
      throw new Error("EXPO_PUBLIC_DEV_AUTH_TOKEN must be removed in OIDC mode.");
    }
    return undefined;
  }
  if (configuredAuthMode !== "dev") {
    if (token) throw new Error("EXPO_PUBLIC_DEV_AUTH_TOKEN requires development auth mode.");
    return undefined;
  }
  if (!token) throw new Error("Development authentication requires a local synthetic token.");
  if (!isLiteralLoopbackHTTPOrigin(apiOrigin)) {
    throw new Error(
      "Development authentication requires an HTTP origin with a literal loopback IP.",
    );
  }
  if (!isDevelopment) {
    throw new Error(
      "Development authentication cannot be included in a non-development mobile build.",
    );
  }
  return token;
}

if (authMode === "oidc") {
  validateOIDCApiOrigin({ apiOrigin: baseUrl, isDevelopment: __DEV__ });
}

const developmentAccessToken = resolveDevelopmentAccessToken({
  apiOrigin: baseUrl,
  authMode,
  token: localToken,
  isDevelopment: __DEV__,
});

export function createMobileApiClient(
  getOIDCAccessToken?: () => Promise<string | undefined>,
): RepForgeClient {
  return createRepForgeClient({
    baseUrl,
    getAccessToken: () =>
      authMode === "oidc" ? (getOIDCAccessToken?.() ?? undefined) : developmentAccessToken,
  });
}

export const apiClient = createMobileApiClient();
