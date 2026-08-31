import { useNetInfo } from "@react-native-community/netinfo";
import { ApiError, type OnboardingState, type RepForgeClient } from "@repforge/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import type { PropsWithChildren } from "react";

import { OnboardingScreen } from "./OnboardingScreen";

jest.mock("@react-native-community/netinfo", () => ({ useNetInfo: jest.fn() }));

const timestamp = "2026-08-30T12:00:00Z";
const savedState: OnboardingState = {
  userId: "01993c86-8fc9-7a5a-9b8d-3302c7e917ec",
  state: "in_progress",
  currentStep: 5,
  version: 4,
  adultAttestedAt: timestamp,
  termsVersion: "terms-1",
  termsAcceptedAt: timestamp,
  privacyVersion: "privacy-1",
  privacyAcceptedAt: timestamp,
  timezone: "Asia/Kolkata",
  units: "metric",
  primaryGoal: "strength",
  experienceLevel: "beginner",
  weeklyAvailability: 4,
  sessionDurationMinutes: 60,
  equipmentAccess: ["dumbbells"],
  dietPreference: "vegetarian",
  safetyAcknowledgedAt: timestamp,
  completedAt: null,
  createdAt: timestamp,
  updatedAt: timestamp,
  requiredTermsVersion: "terms-1",
  requiredPrivacyVersion: "privacy-1",
};

function createWrapper(queryClient = createQueryClient()) {
  return function Wrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Number.POSITIVE_INFINITY },
      mutations: { retry: false, gcTime: Number.POSITIVE_INFINITY },
    },
  });
}

function createClient(overrides: Partial<RepForgeClient> = {}): RepForgeClient {
  return {
    getMe: jest.fn(),
    updateMe: jest.fn(),
    getOnboarding: jest.fn().mockResolvedValue(savedState),
    updateOnboarding: jest.fn().mockResolvedValue(savedState),
    ...overrides,
  };
}

beforeEach(() => {
  jest.mocked(useNetInfo).mockReturnValue({ isConnected: true } as ReturnType<typeof useNetInfo>);
});

it("restores the server snapshot without effect-driven hydration", async () => {
  let resolveState!: (state: OnboardingState) => void;
  const client = createClient({
    getOnboarding: jest.fn(
      () =>
        new Promise((resolve) => {
          resolveState = resolve;
        }),
    ),
  });

  await render(<OnboardingScreen client={client} onComplete={jest.fn()} onLogout={jest.fn()} />, {
    wrapper: createWrapper(),
  });
  expect(screen.getByLabelText("Restoring onboarding progress")).toBeOnTheScreen();

  await act(async () => {
    resolveState(savedState);
  });
  expect(await screen.findByLabelText("Timezone")).toHaveDisplayValue("Asia/Kolkata");
  expect(screen.getByLabelText("Training days per week (1–7)")).toHaveDisplayValue("4");
  expect(screen.getByRole("checkbox", { name: "I accept Terms terms-1" })).toBeChecked();
  expect(screen.getByRole("checkbox", { name: "dumbbells" })).toBeChecked();
  expect(screen.getByRole("checkbox", { name: "bodyweight" })).not.toBeChecked();
});

it.each([
  ["null", null],
  ["empty", []],
])("defaults %s equipment access to bodyweight", async (_label, equipmentAccess) => {
  const client = createClient({
    getOnboarding: jest.fn().mockResolvedValue({ ...savedState, equipmentAccess }),
  });

  await render(<OnboardingScreen client={client} onComplete={jest.fn()} onLogout={jest.fn()} />, {
    wrapper: createWrapper(),
  });

  expect(await screen.findByRole("checkbox", { name: "bodyweight" })).toBeChecked();
  expect(screen.getByRole("checkbox", { name: "dumbbells" })).not.toBeChecked();
});

it("preserves edits while a new consent version re-gates acceptance", async () => {
  const queryClient = createQueryClient();
  const client = createClient();
  await render(<OnboardingScreen client={client} onComplete={jest.fn()} onLogout={jest.fn()} />, {
    wrapper: createWrapper(queryClient),
  });
  const timezone = await screen.findByLabelText("Timezone");
  await fireEvent.changeText(timezone, "Europe/London");

  await act(async () => {
    queryClient.setQueryData(["onboarding", "me", "local"], {
      ...savedState,
      state: "in_progress",
      requiredTermsVersion: "terms-2",
    });
  });

  expect(await screen.findByRole("checkbox", { name: "I accept Terms terms-2" })).not.toBeChecked();
  expect(screen.getByLabelText("Timezone")).toHaveDisplayValue("Europe/London");
  expect(screen.getByRole("checkbox", { name: "I accept Privacy Notice privacy-1" })).toBeChecked();
});

it("retries a failed mutation with the same idempotency key and payload", async () => {
  const updateOnboarding = jest
    .fn()
    .mockRejectedValueOnce(new ApiError(502, "api_unavailable", "Try again."))
    .mockResolvedValueOnce({ ...savedState, version: 5 });
  const client = createClient({ updateOnboarding });
  await render(<OnboardingScreen client={client} onComplete={jest.fn()} onLogout={jest.fn()} />, {
    wrapper: createWrapper(),
  });
  await screen.findByLabelText("Timezone");

  await fireEvent.press(screen.getByRole("button", { name: "Save progress" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("Try again.");
  await waitFor(() => expect(updateOnboarding).toHaveBeenCalledTimes(1));
  const firstCall = updateOnboarding.mock.calls[0];

  await fireEvent.press(screen.getByRole("button", { name: "Retry the same save" }));
  await waitFor(() => expect(updateOnboarding).toHaveBeenCalledTimes(2));
  expect(updateOnboarding.mock.calls[1]).toEqual(firstCall);
  expect(await screen.findByText("Progress saved.")).toBeOnTheScreen();
});

it("can clear a previously saved optional diet preference", async () => {
  const updateOnboarding = jest.fn().mockResolvedValue({
    ...savedState,
    version: 5,
    dietPreference: null,
  });
  const client = createClient({ updateOnboarding });
  await render(<OnboardingScreen client={client} onComplete={jest.fn()} onLogout={jest.fn()} />, {
    wrapper: createWrapper(),
  });
  await screen.findByLabelText("Timezone");

  await fireEvent.press(screen.getByRole("radio", { name: "No preference" }));
  await fireEvent.press(screen.getByRole("button", { name: "Save progress" }));
  await waitFor(() => expect(updateOnboarding).toHaveBeenCalledTimes(1));
  expect(updateOnboarding.mock.calls[0]?.[0]).toMatchObject({ dietPreference: null });
});

it("routes only after the server confirms completion", async () => {
  const completedState: OnboardingState = {
    ...savedState,
    state: "complete",
    currentStep: 6,
    version: 5,
    completedAt: timestamp,
  };
  const client = createClient({ updateOnboarding: jest.fn().mockResolvedValue(completedState) });
  const onComplete = jest.fn();
  await render(<OnboardingScreen client={client} onComplete={onComplete} onLogout={jest.fn()} />, {
    wrapper: createWrapper(),
  });
  await screen.findByLabelText("Timezone");

  await fireEvent.press(screen.getByRole("button", { name: "Finish onboarding" }));
  expect(await screen.findByRole("button", { name: "Continue to RepForge" })).toBeOnTheScreen();
  expect(onComplete).not.toHaveBeenCalled();
  await fireEvent.press(screen.getByRole("button", { name: "Continue to RepForge" }));
  expect(onComplete).toHaveBeenCalledTimes(1);
});

it("shows offline restoration and allows retrying an initial load failure", async () => {
  jest.mocked(useNetInfo).mockReturnValue({ isConnected: false } as ReturnType<typeof useNetInfo>);
  const offlineClient = createClient({ getOnboarding: jest.fn() });
  const onLogout = jest.fn();
  const view = await render(
    <OnboardingScreen client={offlineClient} onComplete={jest.fn()} onLogout={onLogout} />,
    { wrapper: createWrapper() },
  );
  expect(screen.getByRole("header", { name: "You’re offline" })).toBeOnTheScreen();
  await fireEvent.press(screen.getByRole("button", { name: "Log out" }));
  expect(onLogout).toHaveBeenCalledTimes(1);
  await view.unmount();

  jest.mocked(useNetInfo).mockReturnValue({ isConnected: true } as ReturnType<typeof useNetInfo>);
  const getOnboarding = jest
    .fn()
    .mockRejectedValueOnce(new TypeError("synthetic failure"))
    .mockRejectedValueOnce(new TypeError("synthetic failure"))
    .mockResolvedValueOnce(savedState);
  await render(
    <OnboardingScreen
      client={createClient({ getOnboarding })}
      onComplete={jest.fn()}
      onLogout={jest.fn()}
    />,
    { wrapper: createWrapper() },
  );
  expect(
    await screen.findByText(
      "We couldn’t restore your onboarding progress.",
      {},
      { timeout: 4_000 },
    ),
  ).toBeOnTheScreen();
  await fireEvent.press(screen.getByRole("button", { name: "Try again" }));
  expect(await screen.findByLabelText("Timezone")).toHaveDisplayValue("Asia/Kolkata");
});
