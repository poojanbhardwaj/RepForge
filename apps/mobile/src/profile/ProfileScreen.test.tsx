import { useNetInfo } from "@react-native-community/netinfo";
import { ApiError, type Me, type RepForgeClient } from "@repforge/api-client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import type { PropsWithChildren } from "react";

import { ProfileScreen } from "./ProfileScreen";

jest.mock("@react-native-community/netinfo", () => ({ useNetInfo: jest.fn() }));

const me: Me = {
  id: "01993c86-8fc9-7a5a-9b8d-3302c7e917ec",
  profile: {
    displayName: "Local Athlete",
    version: 1,
    createdAt: "2026-08-28T12:00:00Z",
    updatedAt: "2026-08-28T12:00:00Z",
  },
  consents: [],
};

function wrapper({ children }: PropsWithChildren) {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Number.POSITIVE_INFINITY },
      mutations: { retry: false, gcTime: Number.POSITIVE_INFINITY },
    },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  jest.mocked(useNetInfo).mockReturnValue({ isConnected: true } as ReturnType<typeof useNetInfo>);
});

it("shows loading and then the accessible profile form", async () => {
  let resolveProfile!: (value: Me) => void;
  const client: Pick<RepForgeClient, "getMe" | "updateMe"> = {
    getMe: jest.fn(
      () =>
        new Promise((resolve) => {
          resolveProfile = resolve;
        }),
    ),
    updateMe: jest.fn(),
  };
  await render(<ProfileScreen client={client} />, { wrapper });
  expect(screen.getByLabelText("Loading your profile")).toBeOnTheScreen();
  resolveProfile(me);
  const input = await screen.findByLabelText("Display name");
  expect(input).toHaveDisplayValue("Local Athlete");
  expect(input.props.maxLength).toBeUndefined();
  expect(screen.getByRole("button", { name: "Save profile" })).toBeEnabled();
});

it("validates and saves a profile update", async () => {
  const client: Pick<RepForgeClient, "getMe" | "updateMe"> = {
    getMe: jest.fn().mockResolvedValue(me),
    updateMe: jest
      .fn()
      .mockResolvedValue({ ...me, profile: { ...me.profile, displayName: "Pooja", version: 2 } }),
  };
  await render(<ProfileScreen client={client} />, { wrapper });
  const input = await screen.findByLabelText("Display name");
  await fireEvent.changeText(input, "");
  await fireEvent.press(screen.getByRole("button", { name: "Save profile" }));
  expect(await screen.findByText("Enter a display name.")).toBeOnTheScreen();
  await fireEvent.changeText(input, "Pooja");
  await fireEvent.press(screen.getByRole("button", { name: "Save profile" }));
  await waitFor(() =>
    expect(client.updateMe).toHaveBeenCalledWith({ displayName: "Pooja", expectedVersion: 1 }),
  );
  expect(await screen.findByText("Profile saved.")).toBeOnTheScreen();
});

it("shows an offline state and does not offer an unsafe write", async () => {
  jest.mocked(useNetInfo).mockReturnValue({ isConnected: false } as ReturnType<typeof useNetInfo>);
  const client: Pick<RepForgeClient, "getMe" | "updateMe"> = {
    getMe: jest.fn().mockRejectedValue(new TypeError("Network unavailable")),
    updateMe: jest.fn(),
  };
  await render(<ProfileScreen client={client} />, { wrapper });
  expect(await screen.findByRole("header", { name: "You’re offline" })).toBeOnTheScreen();
  expect(screen.queryByRole("button", { name: "Save profile" })).not.toBeOnTheScreen();
});

it("shows a safe conflict message and refreshes stale profile data", async () => {
  const getMe = jest
    .fn()
    .mockResolvedValueOnce(me)
    .mockResolvedValueOnce({ ...me, profile: { ...me.profile, version: 2 } });
  const client: Pick<RepForgeClient, "getMe" | "updateMe"> = {
    getMe,
    updateMe: jest
      .fn()
      .mockRejectedValue(
        new ApiError(409, "version_conflict", "The profile changed. Refresh and try again."),
      ),
  };
  await render(<ProfileScreen client={client} />, { wrapper });
  await screen.findByLabelText("Display name");
  await fireEvent.press(screen.getByRole("button", { name: "Save profile" }));
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "The profile changed. Refresh and try again.",
  );
  await waitFor(() => expect(getMe).toHaveBeenCalledTimes(2));
});

it("preserves a dirty draft and its conflict basis across background refetches", async () => {
  const serverRefresh = {
    ...me,
    profile: { ...me.profile, displayName: "Server edit", version: 2 },
  };
  const conflictRefresh = {
    ...me,
    profile: { ...me.profile, displayName: "Later server edit", version: 3 },
  };
  const getMe = jest
    .fn()
    .mockResolvedValueOnce(me)
    .mockResolvedValueOnce(serverRefresh)
    .mockResolvedValueOnce(conflictRefresh);
  const client: Pick<RepForgeClient, "getMe" | "updateMe"> = {
    getMe,
    updateMe: jest
      .fn()
      .mockRejectedValue(
        new ApiError(409, "version_conflict", "The profile changed. Refresh and try again."),
      ),
  };
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: Number.POSITIVE_INFINITY },
      mutations: { retry: false, gcTime: Number.POSITIVE_INFINITY },
    },
  });
  function draftWrapper({ children }: PropsWithChildren) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }

  await render(<ProfileScreen client={client} />, { wrapper: draftWrapper });
  const input = await screen.findByLabelText("Display name");
  await fireEvent.changeText(input, "Unsaved draft");
  await act(async () => {
    await queryClient.refetchQueries({ queryKey: ["profile", "me", "local"] });
  });
  await waitFor(() => expect(getMe).toHaveBeenCalledTimes(2));
  expect(input).toHaveDisplayValue("Unsaved draft");

  await fireEvent.press(screen.getByRole("button", { name: "Save profile" }));
  await waitFor(() =>
    expect(client.updateMe).toHaveBeenCalledWith({
      displayName: "Unsaved draft",
      expectedVersion: 1,
    }),
  );
  expect(await screen.findByRole("alert")).toHaveTextContent(
    "The profile changed. Refresh and try again.",
  );
  await waitFor(() => expect(getMe).toHaveBeenCalledTimes(3));
  expect(input).toHaveDisplayValue("Unsaved draft");
});
