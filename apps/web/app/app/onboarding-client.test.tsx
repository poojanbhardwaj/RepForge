import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { OnboardingClient } from "./onboarding-client";

const incompleteState = {
  userId: "01993c86-8fc9-7a5a-9b8d-3302c7e917ec",
  state: "in_progress",
  currentStep: 3,
  version: 2,
  adultAttestedAt: "2026-08-30T00:00:00Z",
  termsVersion: "terms-1",
  termsAcceptedAt: "2026-08-30T00:00:00Z",
  privacyVersion: "privacy-1",
  privacyAcceptedAt: "2026-08-30T00:00:00Z",
  timezone: "Asia/Kolkata",
  units: "metric",
  primaryGoal: "strength",
  experienceLevel: "beginner",
  weeklyAvailability: 4,
  sessionDurationMinutes: 60,
  equipmentAccess: ["dumbbells"],
  dietPreference: null,
  safetyAcknowledgedAt: null,
  completedAt: null,
  createdAt: "2026-08-30T00:00:00Z",
  updatedAt: "2026-08-30T00:00:00Z",
  requiredTermsVersion: "terms-1",
  requiredPrivacyVersion: "privacy-1",
} as const;

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("OnboardingClient", () => {
  it("shows loading and hydrates resumable server progress", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse(incompleteState)));
    render(<OnboardingClient userName="Pooja" />);
    expect(screen.getByRole("status")).toHaveTextContent("Restoring your progress");
    expect(
      await screen.findByRole("heading", { name: "Set up your training" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "I confirm I am 18 or older" })).toBeChecked();
    expect(screen.getByLabelText("Training days per week (1–7)")).toHaveValue(4);
    expect(screen.getByRole("checkbox", { name: "dumbbells" })).toBeChecked();
  });

  it("shows a safe error and retries restoration", async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new Error("Network unavailable"))
      .mockResolvedValueOnce(jsonResponse(incompleteState));
    vi.stubGlobal("fetch", fetchMock);
    render(<OnboardingClient userName="Athlete" />);
    expect(
      await screen.findByRole("heading", { name: "We couldn’t restore your progress" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(
      await screen.findByRole("heading", { name: "Set up your training" }),
    ).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("re-gates consent when the server requires a newer legal version", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        jsonResponse({
          ...incompleteState,
          requiredTermsVersion: "terms-2",
          requiredPrivacyVersion: "privacy-2",
        }),
      ),
    );
    render(<OnboardingClient userName="Pooja" />);
    await screen.findByRole("heading", { name: "Set up your training" });

    expect(screen.getByRole("checkbox", { name: "I accept Terms terms-2" })).not.toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: "I accept Privacy Notice privacy-2" }),
    ).not.toBeChecked();
  });

  it("gates product UI until the server reports completion", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        jsonResponse({
          ...incompleteState,
          state: "complete",
          completedAt: "2026-08-30T00:05:00Z",
        }),
      ),
    );
    render(<OnboardingClient userName="Pooja" />);
    expect(
      await screen.findByRole("heading", { name: "You’re ready, Pooja." }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Finish onboarding" })).not.toBeInTheDocument();
  });

  it("reuses the same idempotency key when retrying a failed save", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(incompleteState))
      .mockResolvedValueOnce(
        jsonResponse({ error: { code: "api_unavailable", message: "Try later." } }, 502),
      )
      .mockResolvedValueOnce(jsonResponse(incompleteState));
    vi.stubGlobal("fetch", fetchMock);
    render(<OnboardingClient userName="Pooja" />);
    await screen.findByRole("heading", { name: "Set up your training" });
    fireEvent.click(screen.getByRole("button", { name: "Save progress" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Try later");
    fireEvent.click(screen.getByRole("button", { name: "Retry the same save" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
    const firstSave = fetchMock.mock.calls[1]?.[1] as RequestInit;
    const retry = fetchMock.mock.calls[2]?.[1] as RequestInit;
    expect(new Headers(firstSave.headers).get("Idempotency-Key")).toBe(
      new Headers(retry.headers).get("Idempotency-Key"),
    );
  });

  it("sends an explicit null when the optional diet preference is cleared", async () => {
    const withDiet = { ...incompleteState, dietPreference: "vegetarian" as const };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(withDiet))
      .mockResolvedValueOnce(jsonResponse({ ...withDiet, dietPreference: null }));
    vi.stubGlobal("fetch", fetchMock);
    render(<OnboardingClient userName="Pooja" />);
    await screen.findByRole("heading", { name: "Set up your training" });

    fireEvent.change(screen.getByLabelText("Optional diet preference"), {
      target: { value: "" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save progress" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    const request = fetchMock.mock.calls[1]?.[1] as RequestInit;
    expect(JSON.parse(String(request.body))).toMatchObject({ dietPreference: null });
  });
});

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}
