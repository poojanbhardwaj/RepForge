import { useNetInfo } from "@react-native-community/netinfo";
import { useQueryClient } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react-native";
import { useAuth0 } from "react-native-auth0";

import {
  AuthenticatedApp,
  authErrorMessage,
  shouldClearExpiredCredentials,
} from "./AuthenticatedApp";

const mockApiState: {
  cacheIdentity?: string;
  getAccessToken?: () => Promise<string | undefined>;
} = {};

jest.mock("react-native-auth0", () => ({ useAuth0: jest.fn() }));
jest.mock("@react-native-community/netinfo", () => ({ useNetInfo: jest.fn() }));
jest.mock("@tanstack/react-query", () => ({ useQueryClient: jest.fn() }));
jest.mock("../api", () => ({
  createMobileApiClient: (getAccessToken: () => Promise<string | undefined>) => {
    mockApiState.getAccessToken = getAccessToken;
    return {};
  },
}));
jest.mock("../onboarding/OnboardingScreen", () => ({
  OnboardingScreen: ({ cacheIdentity }: { cacheIdentity: string }) => {
    mockApiState.cacheIdentity = cacheIdentity;
    return null;
  },
}));
jest.mock("../profile/ProfileScreen", () => ({
  ProfileScreen: () => null,
}));

const useAuth0Mock = useAuth0 as jest.MockedFunction<typeof useAuth0>;
const useNetInfoMock = useNetInfo as jest.MockedFunction<typeof useNetInfo>;
const useQueryClientMock = useQueryClient as jest.MockedFunction<typeof useQueryClient>;

beforeEach(() => {
  jest.clearAllMocks();
  delete mockApiState.cacheIdentity;
  delete mockApiState.getAccessToken;
  useNetInfoMock.mockReturnValue({ isConnected: true } as ReturnType<typeof useNetInfo>);
});

it("handles cancellation without presenting an error", () => {
  expect(authErrorMessage({ type: "USER_CANCELLED" })).toBeNull();
});

it("turns the Auth0 post-login denial into an email verification action", () => {
  expect(authErrorMessage({ type: "ACCESS_DENIED", message: "synthetic denial" })).toMatch(
    /Verify your email/,
  );
});

it("distinguishes offline renewal and expired sessions without exposing details", () => {
  expect(authErrorMessage({ type: "NO_NETWORK", message: "sensitive upstream detail" })).toMatch(
    /offline/,
  );
  expect(authErrorMessage({ type: "RENEW_FAILED", message: "synthetic refresh token" })).toBe(
    "Your session has expired. Sign in again.",
  );
});

it("clears only terminal credential failures and retains retryable offline state", () => {
  expect(shouldClearExpiredCredentials({ type: "SESSION_EXPIRED" })).toBe(true);
  expect(shouldClearExpiredCredentials({ type: "NO_REFRESH_TOKEN" })).toBe(true);
  expect(shouldClearExpiredCredentials({ type: "NO_NETWORK" })).toBe(false);
  expect(shouldClearExpiredCredentials({ type: "API_ERROR" })).toBe(false);
});

it("waits for Auth0 initialization before process-death session restoration", async () => {
  const auth = authFixture();
  auth.isLoading = true;
  useAuth0Mock.mockReturnValue(auth as never);
  useQueryClientMock.mockReturnValue({ clear: jest.fn() } as never);
  const view = await render(<AuthenticatedApp />);
  expect(auth.resumeSession).not.toHaveBeenCalled();

  auth.isLoading = false;
  await view.rerender(<AuthenticatedApp />);
  expect(await screen.findByText("Welcome to RepForge")).toBeTruthy();
  expect(auth.resumeSession).toHaveBeenCalledTimes(1);
});

it("partitions onboarding cache identity and clears terminal credentials", async () => {
  const auth = authFixture();
  auth.user = { sub: "auth0|synthetic-a" };
  auth.getCredentials.mockRejectedValue({ type: "SESSION_EXPIRED" });
  useAuth0Mock.mockReturnValue(auth as never);
  const clearCache = jest.fn();
  useQueryClientMock.mockReturnValue({ clear: clearCache } as never);
  await render(<AuthenticatedApp />);
  await waitFor(() => expect(mockApiState.cacheIdentity).toBe("auth0|synthetic-a"));

  await act(async () => {
    await expect(mockApiState.getAccessToken?.()).rejects.toEqual({ type: "SESSION_EXPIRED" });
  });
  expect(auth.clearCredentials).toHaveBeenCalledTimes(1);
  expect(clearCache).toHaveBeenCalledTimes(1);
  expect(await screen.findByText("Your session needs attention")).toBeTruthy();
});

function authFixture() {
  return {
    isLoading: false,
    user: null as { sub: string } | null,
    resumeSession: jest.fn().mockResolvedValue(undefined),
    getCredentials: jest.fn().mockResolvedValue({ accessToken: "synthetic-access-token" }),
    clearCredentials: jest.fn().mockResolvedValue(undefined),
    authorize: jest.fn().mockResolvedValue(undefined),
    clearSession: jest.fn().mockResolvedValue(undefined),
  };
}
