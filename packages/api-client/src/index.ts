import createClient from "openapi-fetch";

import type { components, paths } from "./generated/schema";

export type Me = components["schemas"]["Me"];
export type UpdateMe = components["schemas"]["UpdateMe"];

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly traceId?: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export interface RepForgeClient {
  getMe(signal?: AbortSignal): Promise<Me>;
  updateMe(input: UpdateMe, signal?: AbortSignal): Promise<Me>;
}

interface ClientOptions {
  baseUrl: string;
  getAccessToken: () => string | undefined;
}

export function createRepForgeClient(options: ClientOptions): RepForgeClient {
  const client = createClient<paths>({ baseUrl: options.baseUrl });
  client.use({
    onRequest({ request }) {
      const token = options.getAccessToken();
      if (token) request.headers.set("Authorization", `Bearer ${token}`);
      return request;
    },
  });

  const normalizeError = (status: number, error: unknown): ApiError => {
    const envelope = error as components["schemas"]["ErrorEnvelope"] | undefined;
    return new ApiError(
      status,
      envelope?.error.code ?? "unexpected_response",
      envelope?.error.message ?? "The service returned an unexpected response.",
      envelope?.error.traceId,
    );
  };

  return {
    async getMe(signal) {
      const { data, error, response } = await client.GET("/v1/me", signal ? { signal } : {});
      if (!data) throw normalizeError(response.status, error);
      return data;
    },
    async updateMe(input, signal) {
      const { data, error, response } = await client.PATCH("/v1/me", {
        body: input,
        ...(signal ? { signal } : {}),
      });
      if (!data) throw normalizeError(response.status, error);
      return data;
    },
  };
}
