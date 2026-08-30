"use client";

import type { OnboardingState, UpdateOnboarding } from "@repforge/api-client";
import Link from "next/link";
import { useCallback, useEffect, useState, useSyncExternalStore } from "react";

import { track } from "../../src/analytics";

const equipmentOptions = [
  "bodyweight",
  "dumbbells",
  "barbell",
  "rack",
  "bench",
  "cables",
  "machines",
  "bands",
] as const;
type Equipment = (typeof equipmentOptions)[number];
type Submission = { input: UpdateOnboarding; key: string; step: number };

interface FormState {
  adultAttested: boolean;
  termsAcceptedVersion: string | null;
  privacyAcceptedVersion: string | null;
  timezone: string;
  units: "metric" | "imperial";
  primaryGoal: "strength" | "muscle" | "general_fitness";
  experienceLevel: "beginner" | "intermediate";
  weeklyAvailability: string;
  sessionDurationMinutes: string;
  equipmentAccess: Equipment[];
  dietPreference: "" | "vegetarian" | "eggetarian" | "vegan" | "omnivore";
  safetyAcknowledged: boolean;
}

const initialForm: FormState = {
  adultAttested: false,
  termsAcceptedVersion: null,
  privacyAcceptedVersion: null,
  timezone: "Asia/Kolkata",
  units: "metric",
  primaryGoal: "strength",
  experienceLevel: "beginner",
  weeklyAvailability: "3",
  sessionDurationMinutes: "45",
  equipmentAccess: ["bodyweight"],
  dietPreference: "",
  safetyAcknowledged: false,
};

type LoadState =
  | { status: "loading" }
  | { status: "ready"; onboarding: OnboardingState }
  | { status: "error"; message: string };

export function OnboardingClient({ userName }: { userName: string }) {
  const online = useOnlineStatus();
  const [loadState, setLoadState] = useState<LoadState>({ status: "loading" });

  useEffect(() => {
    let active = true;
    void requestOnboarding("GET").then(
      (onboarding) => {
        if (active) setLoadState({ status: "ready", onboarding });
      },
      (caught: unknown) => {
        if (active) setLoadState({ status: "error", message: messageFor(caught) });
      },
    );
    return () => {
      active = false;
    };
  }, []);

  const restore = useCallback(async () => {
    setLoadState({ status: "loading" });
    try {
      const onboarding = await requestOnboarding("GET");
      setLoadState({ status: "ready", onboarding });
    } catch (caught) {
      setLoadState({ status: "error", message: messageFor(caught) });
    }
  }, []);

  if (!online && loadState.status !== "ready")
    return (
      <Status
        title="You’re offline"
        message="Reconnect to restore your saved onboarding progress."
      />
    );
  if (loadState.status === "loading")
    return (
      <Status
        busy
        title="Restoring your progress"
        message="Checking the server for your latest saved step…"
      />
    );
  if (loadState.status === "error")
    return (
      <Status
        title="We couldn’t restore your progress"
        message={loadState.message}
        action={
          <button type="button" onClick={() => void restore()}>
            Try again
          </button>
        }
      />
    );
  if (loadState.onboarding.state === "complete") {
    return <CompletedOnboarding userName={userName} />;
  }

  return (
    <OnboardingForm
      initialState={loadState.onboarding}
      online={online}
      onSaved={(onboarding) => setLoadState({ status: "ready", onboarding })}
      userName={userName}
    />
  );
}

function OnboardingForm({
  initialState,
  online,
  onSaved,
  userName,
}: {
  initialState: OnboardingState;
  online: boolean;
  onSaved: (state: OnboardingState) => void;
  userName: string;
}) {
  const [saving, setSaving] = useState(false);
  const [state, setState] = useState(initialState);
  const [form, setForm] = useState(() => hydrate(initialState));
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [lastSubmission, setLastSubmission] = useState<Submission | null>(null);

  const saveSubmission = async (submission: Submission) => {
    setSaving(true);
    setError(null);
    setNotice(null);
    try {
      const saved = await requestOnboarding("PATCH", submission);
      setState(saved);
      onSaved(saved);
      setNotice(saved.state === "complete" ? "Onboarding complete." : "Progress saved.");
      if (saved.state === "complete") {
        track({ name: "onboarding_completed", source: "web", step: 6 });
      } else {
        track({ name: "onboarding_progress_saved", source: "web", step: submission.step });
      }
    } catch (caught) {
      setError(messageFor(caught));
    } finally {
      setSaving(false);
    }
  };

  const submit = (complete: boolean) => {
    const weekly = Number(form.weeklyAvailability);
    const duration = Number(form.sessionDurationMinutes);
    if (
      !form.timezone.trim() ||
      !Number.isInteger(weekly) ||
      weekly < 1 ||
      weekly > 7 ||
      !Number.isInteger(duration) ||
      duration < 15 ||
      duration > 180 ||
      form.equipmentAccess.length === 0
    ) {
      setError("Check your timezone, availability, session duration, and equipment choices.");
      return;
    }
    if (
      complete &&
      (!form.adultAttested ||
        form.termsAcceptedVersion !== state.requiredTermsVersion ||
        form.privacyAcceptedVersion !== state.requiredPrivacyVersion ||
        !form.safetyAcknowledged)
    ) {
      setError("Accept the adult, Terms, Privacy, and safety acknowledgements to finish.");
      return;
    }
    const step = complete ? 6 : form.safetyAcknowledged ? 5 : form.adultAttested ? 3 : 1;
    const input: UpdateOnboarding = {
      ...(form.adultAttested ? { adultAttested: true as const } : {}),
      ...(form.termsAcceptedVersion === state.requiredTermsVersion
        ? { termsVersion: state.requiredTermsVersion }
        : {}),
      ...(form.privacyAcceptedVersion === state.requiredPrivacyVersion
        ? { privacyVersion: state.requiredPrivacyVersion }
        : {}),
      timezone: form.timezone.trim(),
      units: form.units,
      primaryGoal: form.primaryGoal,
      experienceLevel: form.experienceLevel,
      weeklyAvailability: weekly,
      sessionDurationMinutes: duration,
      equipmentAccess: form.equipmentAccess,
      dietPreference: form.dietPreference || null,
      ...(form.safetyAcknowledged ? { safetyAcknowledged: true as const } : {}),
      currentStep: step,
    };
    const submission = { input, key: crypto.randomUUID(), step };
    setLastSubmission(submission);
    void saveSubmission(submission);
  };

  return (
    <main className="product-shell">
      <ProductNav userName={userName} />
      <form
        className="panel onboarding-form"
        onSubmit={(event) => {
          event.preventDefault();
          submit(true);
        }}
      >
        <div>
          <p className="eyebrow">Resumable setup</p>
          <h1>Set up your training</h1>
          <p>
            Your progress is saved on the server. We ask for practical preferences, not diagnoses or
            medical history.
          </p>
        </div>
        {state.version === 1 ? (
          <p className="notice" role="status">
            No progress saved yet. You can stop and resume at any time.
          </p>
        ) : null}
        {!online ? (
          <p className="warning" role="alert">
            Offline: review is available, but saving is paused.
          </p>
        ) : null}
        <Check
          label="I confirm I am 18 or older"
          checked={form.adultAttested}
          onChange={(value) => setForm({ ...form, adultAttested: value })}
        />
        <Check
          label={`I accept Terms ${state.requiredTermsVersion}`}
          checked={form.termsAcceptedVersion === state.requiredTermsVersion}
          onChange={(value) =>
            setForm({
              ...form,
              termsAcceptedVersion: value ? state.requiredTermsVersion : null,
            })
          }
        />
        <Check
          label={`I accept Privacy Notice ${state.requiredPrivacyVersion}`}
          checked={form.privacyAcceptedVersion === state.requiredPrivacyVersion}
          onChange={(value) =>
            setForm({
              ...form,
              privacyAcceptedVersion: value ? state.requiredPrivacyVersion : null,
            })
          }
        />
        <label>
          Timezone
          <input
            value={form.timezone}
            onChange={(event) => setForm({ ...form, timezone: event.target.value })}
            autoComplete="off"
          />
        </label>
        <Select
          label="Units"
          value={form.units}
          options={["metric", "imperial"]}
          onChange={(value) => setForm({ ...form, units: value as FormState["units"] })}
        />
        <Select
          label="Primary goal"
          value={form.primaryGoal}
          options={["strength", "muscle", "general_fitness"]}
          onChange={(value) => setForm({ ...form, primaryGoal: value as FormState["primaryGoal"] })}
        />
        <Select
          label="Experience"
          value={form.experienceLevel}
          options={["beginner", "intermediate"]}
          onChange={(value) =>
            setForm({ ...form, experienceLevel: value as FormState["experienceLevel"] })
          }
        />
        <label>
          Training days per week (1–7)
          <input
            type="number"
            min="1"
            max="7"
            value={form.weeklyAvailability}
            onChange={(event) => setForm({ ...form, weeklyAvailability: event.target.value })}
          />
        </label>
        <label>
          Minutes per session (15–180)
          <input
            type="number"
            min="15"
            max="180"
            value={form.sessionDurationMinutes}
            onChange={(event) => setForm({ ...form, sessionDurationMinutes: event.target.value })}
          />
        </label>
        <fieldset>
          <legend>Equipment access</legend>
          <div className="check-grid">
            {equipmentOptions.map((equipment) => (
              <Check
                key={equipment}
                label={equipment}
                checked={form.equipmentAccess.includes(equipment)}
                onChange={(checked) =>
                  setForm({
                    ...form,
                    equipmentAccess: checked
                      ? [...new Set([...form.equipmentAccess, equipment])]
                      : form.equipmentAccess.filter((item) => item !== equipment),
                  })
                }
              />
            ))}
          </div>
        </fieldset>
        <Select
          label="Optional diet preference"
          value={form.dietPreference}
          options={["", "vegetarian", "eggetarian", "vegan", "omnivore"]}
          onChange={(value) =>
            setForm({ ...form, dietPreference: value as FormState["dietPreference"] })
          }
          emptyLabel="No preference"
        />
        <Check
          label="I’ll stop if something feels unsafe and seek qualified help when needed"
          checked={form.safetyAcknowledged}
          onChange={(value) => setForm({ ...form, safetyAcknowledged: value })}
        />
        {error ? (
          <p className="error-text" role="alert">
            {error}
          </p>
        ) : null}
        {notice ? (
          <p className="success-text" role="status">
            {notice}
          </p>
        ) : null}
        <div className="actions">
          <button
            className="secondary"
            type="button"
            disabled={!online || saving}
            onClick={() => submit(false)}
          >
            {saving ? "Saving…" : "Save progress"}
          </button>
          <button type="submit" disabled={!online || saving}>
            Finish onboarding
          </button>
          {error && lastSubmission ? (
            <button
              className="text-button"
              type="button"
              disabled={!online || saving}
              onClick={() => void saveSubmission(lastSubmission)}
            >
              Retry the same save
            </button>
          ) : null}
        </div>
      </form>
    </main>
  );
}

function hydrate(state: OnboardingState): FormState {
  return {
    ...initialForm,
    adultAttested: Boolean(state.adultAttestedAt),
    termsAcceptedVersion: state.termsAcceptedAt ? (state.termsVersion ?? null) : null,
    privacyAcceptedVersion: state.privacyAcceptedAt ? (state.privacyVersion ?? null) : null,
    timezone: state.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone ?? "Asia/Kolkata",
    units: state.units ?? "metric",
    primaryGoal: state.primaryGoal ?? "strength",
    experienceLevel: state.experienceLevel ?? "beginner",
    weeklyAvailability: String(state.weeklyAvailability ?? 3),
    sessionDurationMinutes: String(state.sessionDurationMinutes ?? 45),
    equipmentAccess: state.equipmentAccess.length ? state.equipmentAccess : ["bodyweight"],
    dietPreference: state.dietPreference ?? "",
    safetyAcknowledged: Boolean(state.safetyAcknowledgedAt),
  };
}

function CompletedOnboarding({ userName }: { userName: string }) {
  return (
    <main className="product-shell">
      <ProductNav userName={userName} />
      <section className="panel" aria-labelledby="ready-heading">
        <p className="eyebrow">Setup saved</p>
        <h1 id="ready-heading">You’re ready, {userName}.</h1>
        <p>
          Your onboarding is complete. Product features remain gated until their readiness
          milestones are implemented.
        </p>
      </section>
    </main>
  );
}

function subscribeToOnlineStatus(callback: () => void): () => void {
  window.addEventListener("online", callback);
  window.addEventListener("offline", callback);
  return () => {
    window.removeEventListener("online", callback);
    window.removeEventListener("offline", callback);
  };
}

function useOnlineStatus(): boolean {
  return useSyncExternalStore(
    subscribeToOnlineStatus,
    () => navigator.onLine,
    () => true,
  );
}

async function requestOnboarding(
  method: "GET" | "PATCH",
  submission?: Submission,
): Promise<OnboardingState> {
  const response = await fetch("/api/onboarding", {
    method,
    cache: "no-store",
    ...(submission
      ? {
          headers: { "Content-Type": "application/json", "Idempotency-Key": submission.key },
          body: JSON.stringify(submission.input),
        }
      : {}),
  });
  const data: unknown = await response.json().catch(() => null);
  if (!response.ok) {
    const envelope = data as { error?: { code?: string; message?: string } } | null;
    throw new Error(
      envelope?.error?.code === "session_expired"
        ? "Your session expired. Sign in again to continue."
        : (envelope?.error?.message ?? "We couldn’t save your progress."),
    );
  }
  return data as OnboardingState;
}

function messageFor(caught: unknown): string {
  return caught instanceof Error ? caught.message : "Something went wrong.";
}
function ProductNav({ userName }: { userName: string }) {
  return (
    <nav aria-label="Account navigation">
      <Link className="brand" href="/">
        RepForge
      </Link>
      <span>{userName}</span>
      <a href="/auth/logout">Log out</a>
    </nav>
  );
}
function Status({
  action,
  busy = false,
  message,
  title,
}: {
  action?: React.ReactNode;
  busy?: boolean;
  message: string;
  title: string;
}) {
  return (
    <main className="product-shell" aria-busy={busy}>
      <section className="panel" role={busy ? "status" : undefined}>
        <h1>{title}</h1>
        <p>{message}</p>
        {action}
      </section>
    </main>
  );
}
function Check({
  checked,
  label,
  onChange,
}: {
  checked: boolean;
  label: string;
  onChange: (value: boolean) => void;
}) {
  return (
    <label className="check">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
      />
      <span>{label}</span>
    </label>
  );
}
function Select({
  emptyLabel,
  label,
  onChange,
  options,
  value,
}: {
  emptyLabel?: string;
  label: string;
  onChange: (value: string) => void;
  options: readonly string[];
  value: string;
}) {
  return (
    <label>
      {label}
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        {options.map((option) => (
          <option key={option || "empty"} value={option}>
            {option ? option.replaceAll("_", " ") : emptyLabel}
          </option>
        ))}
      </select>
    </label>
  );
}
