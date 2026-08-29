import { createRepForgeClient } from "@repforge/api-client";

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
  const developmentCredentialsActive = configuredAuthMode === "dev" || Boolean(token);
  if (!developmentCredentialsActive) return undefined;
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
  return configuredAuthMode === "dev" ? token : undefined;
}

const developmentAccessToken = resolveDevelopmentAccessToken({
  apiOrigin: baseUrl,
  authMode,
  token: localToken,
  isDevelopment: __DEV__,
});

export const apiClient = createRepForgeClient({
  baseUrl,
  getAccessToken: () => developmentAccessToken,
});
