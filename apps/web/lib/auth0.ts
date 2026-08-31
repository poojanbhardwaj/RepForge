import { Auth0Client } from "@auth0/nextjs-auth0/server";
import { NextResponse } from "next/server";

const requiredEnvironment = [
  "AUTH0_DOMAIN",
  "AUTH0_CLIENT_ID",
  "AUTH0_CLIENT_SECRET",
  "AUTH0_SECRET",
  "APP_BASE_URL",
  "AUTH0_AUDIENCE",
  "API_BASE_URL",
] as const;

export const WEB_AUTHORIZATION_SCOPE =
  "openid profile email offline_access profile:read profile:write";

export interface WebAuthConfig {
  domain: string;
  clientId: string;
  clientSecret: string;
  secret: string;
  appBaseUrl: string;
  audience: string;
  apiBaseUrl: string;
  secureCookies: boolean;
}

export class WebAuthConfigurationError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "WebAuthConfigurationError";
  }
}

export function readWebAuthConfig(environment: NodeJS.ProcessEnv = process.env): WebAuthConfig {
  const missing = requiredEnvironment.filter((name) => !environment[name]?.trim());
  if (missing.length > 0) {
    throw new WebAuthConfigurationError(`Missing required environment: ${missing.join(", ")}`);
  }

  const domain = environment.AUTH0_DOMAIN!;
  const clientId = environment.AUTH0_CLIENT_ID!;
  const clientSecret = environment.AUTH0_CLIENT_SECRET!;
  const secret = environment.AUTH0_SECRET!;
  const appBaseUrl = canonicalOrigin(
    environment.APP_BASE_URL!,
    "APP_BASE_URL",
    environment.NODE_ENV,
  );
  const audience = canonicalHttpsUrl(environment.AUTH0_AUDIENCE!, "AUTH0_AUDIENCE");
  const apiBaseUrl = canonicalApiBaseUrl(environment.API_BASE_URL!, environment.NODE_ENV);

  if (
    !/^(?=.{1,253}$)(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(
      domain,
    )
  ) {
    throw new WebAuthConfigurationError("AUTH0_DOMAIN must be a bare DNS hostname");
  }
  if (
    clientId.length < 8 ||
    clientId.length > 256 ||
    /\s/.test(clientId) ||
    clientId.startsWith("__POPULATE_")
  ) {
    throw new WebAuthConfigurationError("AUTH0_CLIENT_ID is invalid");
  }
  if (
    clientSecret.length < 16 ||
    /\s/.test(clientSecret) ||
    clientSecret.startsWith("__POPULATE_")
  ) {
    throw new WebAuthConfigurationError("AUTH0_CLIENT_SECRET is invalid");
  }
  if (!/^[a-fA-F0-9]{64}$/.test(secret)) {
    throw new WebAuthConfigurationError("AUTH0_SECRET must be a 32-byte hex value");
  }

  return {
    domain,
    clientId,
    clientSecret,
    secret,
    appBaseUrl,
    audience,
    apiBaseUrl,
    secureCookies: new URL(appBaseUrl).protocol === "https:",
  };
}

function canonicalOrigin(value: string, name: string, nodeEnvironment?: string): string {
  let parsed: URL;
  try {
    if (value.trim() !== value) throw new Error();
    parsed = new URL(value);
  } catch {
    throw new WebAuthConfigurationError(`${name} must be an absolute URL`);
  }
  if (
    (parsed.protocol !== "https:" && parsed.protocol !== "http:") ||
    parsed.username ||
    parsed.password ||
    parsed.pathname !== "/" ||
    parsed.search ||
    parsed.hash ||
    value !== parsed.origin
  ) {
    throw new WebAuthConfigurationError(`${name} must be a canonical HTTP(S) origin`);
  }
  if (nodeEnvironment === "production" && parsed.protocol !== "https:") {
    throw new WebAuthConfigurationError(`${name} must use HTTPS in production`);
  }
  return parsed.origin;
}

function canonicalHttpsUrl(value: string, name: string): string {
  let parsed: URL;
  try {
    if (value.trim() !== value) throw new Error();
    parsed = new URL(value);
  } catch {
    throw new WebAuthConfigurationError(`${name} must be an absolute URL`);
  }
  if (
    parsed.protocol !== "https:" ||
    parsed.username ||
    parsed.password ||
    parsed.search ||
    parsed.hash ||
    value !== parsed.toString().replace(/\/$/, "")
  ) {
    throw new WebAuthConfigurationError(`${name} must be a canonical HTTPS URL`);
  }
  return parsed.toString().replace(/\/$/, "");
}

function canonicalApiBaseUrl(value: string, nodeEnvironment: string | undefined): string {
  const origin = canonicalOrigin(value, "API_BASE_URL", nodeEnvironment);
  const parsed = new URL(origin);
  if (nodeEnvironment === "production" && parsed.protocol !== "https:") {
    throw new WebAuthConfigurationError("API_BASE_URL must use HTTPS in production");
  }
  return origin;
}

let auth0Client: Auth0Client | undefined;

export function getAuth0(): Auth0Client {
  if (auth0Client) return auth0Client;
  const config = readWebAuthConfig();
  auth0Client = new Auth0Client({
    domain: config.domain,
    clientId: config.clientId,
    clientSecret: config.clientSecret,
    secret: config.secret,
    appBaseUrl: config.appBaseUrl,
    signInReturnToPath: "/app",
    authorizationParameters: {
      audience: config.audience,
      scope: WEB_AUTHORIZATION_SCOPE,
    },
    session: {
      rolling: true,
      absoluteDuration: 60 * 60 * 24 * 3,
      inactivityDuration: 60 * 60 * 8,
      cookie: {
        sameSite: "lax",
        secure: config.secureCookies,
        path: "/",
        transient: false,
      },
    },
    transactionCookie: {
      sameSite: "lax",
      secure: config.secureCookies,
      path: "/",
      maxAge: 60 * 10,
    },
    logoutStrategy: "oidc",
    includeIdTokenHintInOIDCLogoutUrl: true,
    enableAccessTokenEndpoint: false,
    tokenRefreshBuffer: 60,
    httpTimeout: 5_000,
    onCallback(error, context, session) {
      if (error || !session) {
        const code = isAccessDenied(error) ? "access_denied" : "authentication_failed";
        return Promise.resolve(
          NextResponse.redirect(new URL(`/auth/error?code=${code}`, context.appBaseUrl)),
        );
      }
      return Promise.resolve(NextResponse.redirect(new URL("/app", context.appBaseUrl)));
    },
  });
  return auth0Client;
}

function isAccessDenied(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  const cause = "cause" in error ? error.cause : undefined;
  return Boolean(
    cause && typeof cause === "object" && "code" in cause && cause.code === "access_denied",
  );
}
