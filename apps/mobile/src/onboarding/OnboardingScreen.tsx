import { useNetInfo } from "@react-native-community/netinfo";
import {
  ApiError,
  type OnboardingState,
  type RepForgeClient,
  type UpdateOnboarding,
} from "@repforge/api-client";
import { colors, radius, spacing, touchTarget } from "@repforge/ui-tokens";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { randomUUID } from "expo-crypto";
import { useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";

import { track } from "../analytics";

function onboardingKey(cacheIdentity: string) {
  return ["onboarding", "me", cacheIdentity] as const;
}
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

type Submission = { input: UpdateOnboarding; key: string; completing: boolean; step: number };

function idempotencyKey(): string {
  return randomUUID();
}

export function OnboardingScreen({
  cacheIdentity = "local",
  client,
  onComplete,
  onLogout,
}: {
  cacheIdentity?: string;
  client: RepForgeClient;
  onComplete: () => void;
  onLogout: () => void;
}) {
  const network = useNetInfo();
  const queryKey = onboardingKey(cacheIdentity);
  const onboarding = useQuery({
    queryKey,
    queryFn: ({ signal }) => client.getOnboarding(signal),
    enabled: network.isConnected !== false,
    retry: 1,
  });

  if (network.isConnected === false && !onboarding.data) {
    return (
      <SafeAreaView style={styles.centered}>
        <Text accessibilityRole="header" style={styles.title}>
          You’re offline
        </Text>
        <Text style={styles.muted}>Reconnect to restore your saved onboarding progress.</Text>
        <LogoutButton onPress={onLogout} />
      </SafeAreaView>
    );
  }

  if (onboarding.isPending) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator accessibilityLabel="Restoring onboarding progress" size="large" />
        <Text style={styles.muted}>Restoring your progress…</Text>
      </SafeAreaView>
    );
  }

  if (onboarding.isError || !onboarding.data) {
    return (
      <SafeAreaView style={styles.centered}>
        <Text accessibilityRole="alert" style={styles.error}>
          We couldn’t restore your onboarding progress.
        </Text>
        <Pressable
          accessibilityRole="button"
          onPress={() => void onboarding.refetch()}
          style={styles.button}
        >
          <Text style={styles.buttonText}>Try again</Text>
        </Pressable>
        <LogoutButton onPress={onLogout} />
      </SafeAreaView>
    );
  }

  if (onboarding.data.state === "complete") {
    return (
      <SafeAreaView style={styles.centered}>
        <Text accessibilityRole="header" style={styles.title}>
          You’re ready
        </Text>
        <Text accessibilityLiveRegion="polite" style={styles.success}>
          Your onboarding is complete and saved.
        </Text>
        <Pressable accessibilityRole="button" onPress={onComplete} style={styles.button}>
          <Text style={styles.buttonText}>Continue to RepForge</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  return (
    <OnboardingForm
      client={client}
      offline={network.isConnected === false}
      onLogout={onLogout}
      queryKey={queryKey}
      serverState={onboarding.data}
    />
  );
}

function OnboardingForm({
  client,
  offline,
  onLogout,
  queryKey,
  serverState,
}: {
  client: RepForgeClient;
  offline: boolean;
  onLogout: () => void;
  queryKey: ReturnType<typeof onboardingKey>;
  serverState: OnboardingState;
}) {
  const queryClient = useQueryClient();
  const [adultAttested, setAdultAttested] = useState(() => Boolean(serverState.adultAttestedAt));
  const [termsAcceptedVersion, setTermsAcceptedVersion] = useState<string | null>(() =>
    serverState.termsAcceptedAt && serverState.termsVersion === serverState.requiredTermsVersion
      ? serverState.requiredTermsVersion
      : null,
  );
  const [privacyAcceptedVersion, setPrivacyAcceptedVersion] = useState<string | null>(() =>
    serverState.privacyAcceptedAt &&
    serverState.privacyVersion === serverState.requiredPrivacyVersion
      ? serverState.requiredPrivacyVersion
      : null,
  );
  const [timezone, setTimezone] = useState(
    () =>
      serverState.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone || "Asia/Kolkata",
  );
  const [units, setUnits] = useState<"metric" | "imperial">(() => serverState.units ?? "metric");
  const [goal, setGoal] = useState<"strength" | "muscle" | "general_fitness">(
    () => serverState.primaryGoal ?? "strength",
  );
  const [experience, setExperience] = useState<"beginner" | "intermediate">(
    () => serverState.experienceLevel ?? "beginner",
  );
  const [weeklyAvailability, setWeeklyAvailability] = useState(() =>
    String(serverState.weeklyAvailability ?? 3),
  );
  const [sessionDuration, setSessionDuration] = useState(() =>
    String(serverState.sessionDurationMinutes ?? 45),
  );
  const [equipment, setEquipment] = useState<Equipment[]>(() =>
    serverState.equipmentAccess.length > 0 ? [...serverState.equipmentAccess] : ["bodyweight"],
  );
  const [diet, setDiet] = useState<"" | "vegetarian" | "eggetarian" | "vegan" | "omnivore">(
    () => serverState.dietPreference ?? "",
  );
  const [safetyAcknowledged, setSafetyAcknowledged] = useState(() =>
    Boolean(serverState.safetyAcknowledgedAt),
  );
  const [localError, setLocalError] = useState<string | null>(null);
  const [lastSubmission, setLastSubmission] = useState<Submission | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const termsAccepted = termsAcceptedVersion === serverState.requiredTermsVersion;
  const privacyAccepted = privacyAcceptedVersion === serverState.requiredPrivacyVersion;

  const save = useMutation({
    mutationFn: (submission: Submission) =>
      client.updateOnboarding(submission.input, submission.key),
    onSuccess: (state, submission) => {
      queryClient.setQueryData(queryKey, state);
      setSuccess(state.state === "complete" ? "Onboarding complete." : "Progress saved.");
      setLocalError(null);
      if (state.state === "complete") {
        track({ name: "onboarding_completed", source: "mobile", step: 6 });
      } else {
        track({ name: "onboarding_progress_saved", source: "mobile", step: submission.step });
      }
    },
  });

  const submit = (completing: boolean) => {
    const weekly = Number(weeklyAvailability);
    const duration = Number(sessionDuration);
    if (
      !timezone.trim() ||
      !Number.isInteger(weekly) ||
      weekly < 1 ||
      weekly > 7 ||
      !Number.isInteger(duration) ||
      duration < 15 ||
      duration > 180 ||
      equipment.length === 0
    ) {
      setLocalError("Check your timezone, availability, session duration, and equipment choices.");
      return;
    }
    if (
      completing &&
      (!adultAttested || !termsAccepted || !privacyAccepted || !safetyAcknowledged)
    ) {
      setLocalError("Accept the adult, Terms, Privacy, and safety acknowledgements to finish.");
      return;
    }
    const step = completing ? 6 : safetyAcknowledged ? 5 : adultAttested ? 3 : 1;
    const input: UpdateOnboarding = {
      ...(adultAttested ? { adultAttested: true as const } : {}),
      ...(termsAccepted ? { termsVersion: serverState.requiredTermsVersion } : {}),
      ...(privacyAccepted ? { privacyVersion: serverState.requiredPrivacyVersion } : {}),
      timezone: timezone.trim(),
      units,
      primaryGoal: goal,
      experienceLevel: experience,
      weeklyAvailability: weekly,
      sessionDurationMinutes: duration,
      equipmentAccess: equipment,
      dietPreference: diet || null,
      ...(safetyAcknowledged ? { safetyAcknowledged: true as const } : {}),
      currentStep: step,
    };
    const submission = { input, key: idempotencyKey(), completing, step };
    setLastSubmission(submission);
    setSuccess(null);
    setLocalError(null);
    save.mutate(submission);
  };

  return (
    <SafeAreaView style={styles.page}>
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Text accessibilityRole="header" style={styles.title}>
          Set up your training
        </Text>
        <Text style={styles.muted}>
          Your progress is saved on the server. We ask only for practical training preferences—not
          diagnoses or medical history.
        </Text>
        {serverState.version === 1 ? (
          <Text accessibilityLiveRegion="polite" style={styles.empty}>
            No progress saved yet. You can stop and resume at any time.
          </Text>
        ) : null}
        {offline ? (
          <Text accessibilityRole="alert" style={styles.warning}>
            Offline: review is available, but saving is paused until you reconnect.
          </Text>
        ) : null}

        <CheckRow
          checked={adultAttested}
          label="I confirm I am 18 or older"
          onChange={setAdultAttested}
        />
        <CheckRow
          checked={termsAccepted}
          label={`I accept Terms ${serverState.requiredTermsVersion}`}
          onChange={(checked) =>
            setTermsAcceptedVersion(checked ? serverState.requiredTermsVersion : null)
          }
        />
        <CheckRow
          checked={privacyAccepted}
          label={`I accept Privacy Notice ${serverState.requiredPrivacyVersion}`}
          onChange={(checked) =>
            setPrivacyAcceptedVersion(checked ? serverState.requiredPrivacyVersion : null)
          }
        />

        <LabeledInput label="Timezone" onChangeText={setTimezone} value={timezone} />
        <ChoiceGroup
          label="Units"
          options={["metric", "imperial"]}
          value={units}
          onChange={setUnits}
        />
        <ChoiceGroup
          label="Primary goal"
          options={["strength", "muscle", "general_fitness"]}
          value={goal}
          onChange={setGoal}
        />
        <ChoiceGroup
          label="Experience"
          options={["beginner", "intermediate"]}
          value={experience}
          onChange={setExperience}
        />
        <LabeledInput
          keyboardType="number-pad"
          label="Training days per week (1–7)"
          onChangeText={setWeeklyAvailability}
          value={weeklyAvailability}
        />
        <LabeledInput
          keyboardType="number-pad"
          label="Minutes per session (15–180)"
          onChangeText={setSessionDuration}
          value={sessionDuration}
        />

        <Text accessibilityRole="header" style={styles.sectionTitle}>
          Equipment access
        </Text>
        <View accessibilityRole="list" style={styles.choices}>
          {equipmentOptions.map((item) => (
            <CheckRow
              checked={equipment.includes(item)}
              key={item}
              label={item.replace("_", " ")}
              onChange={(checked) =>
                setEquipment((current) =>
                  checked
                    ? [...new Set([...current, item])]
                    : current.filter((value) => value !== item),
                )
              }
            />
          ))}
        </View>
        <ChoiceGroup
          label="Optional diet preference"
          options={["", "vegetarian", "eggetarian", "vegan", "omnivore"]}
          value={diet}
          onChange={setDiet}
          labels={{ "": "No preference" }}
        />
        <CheckRow
          checked={safetyAcknowledged}
          label="I’ll stop if something feels unsafe and seek qualified help when needed"
          onChange={setSafetyAcknowledged}
        />

        {localError ? (
          <Text accessibilityRole="alert" style={styles.error}>
            {localError}
          </Text>
        ) : null}
        {save.isError ? (
          <View style={styles.errorPanel}>
            <Text accessibilityRole="alert" style={styles.error}>
              {save.error instanceof ApiError
                ? save.error.message
                : "We couldn’t save your progress."}
            </Text>
            {lastSubmission ? (
              <Pressable
                accessibilityRole="button"
                disabled={offline}
                onPress={() => save.mutate(lastSubmission)}
                style={styles.secondaryButton}
              >
                <Text style={styles.secondaryButtonText}>Retry the same save</Text>
              </Pressable>
            ) : null}
          </View>
        ) : null}
        {success ? (
          <Text accessibilityLiveRegion="polite" style={styles.success}>
            {success}
          </Text>
        ) : null}
        <Pressable
          accessibilityRole="button"
          disabled={offline || save.isPending}
          onPress={() => submit(false)}
          style={[styles.secondaryButton, (offline || save.isPending) && styles.disabled]}
        >
          <Text style={styles.secondaryButtonText}>
            {save.isPending ? "Saving…" : "Save progress"}
          </Text>
        </Pressable>
        <Pressable
          accessibilityRole="button"
          disabled={offline || save.isPending}
          onPress={() => submit(true)}
          style={[styles.button, (offline || save.isPending) && styles.disabled]}
        >
          <Text style={styles.buttonText}>Finish onboarding</Text>
        </Pressable>
        <LogoutButton onPress={onLogout} />
      </ScrollView>
    </SafeAreaView>
  );
}

function CheckRow({
  checked,
  label,
  onChange,
}: {
  checked: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <Pressable
      accessibilityRole="checkbox"
      accessibilityState={{ checked }}
      onPress={() => onChange(!checked)}
      style={styles.checkRow}
    >
      <View style={[styles.checkbox, checked && styles.checkboxChecked]} />
      <Text style={styles.checkLabel}>{label}</Text>
    </Pressable>
  );
}

function ChoiceGroup<T extends string>({
  label,
  labels = {},
  onChange,
  options,
  value,
}: {
  label: string;
  labels?: Record<string, string>;
  onChange: (value: T) => void;
  options: readonly T[];
  value: T;
}) {
  return (
    <View accessibilityRole="radiogroup" style={styles.field}>
      <Text style={styles.label}>{label}</Text>
      <View style={styles.choices}>
        {options.map((option) => (
          <Pressable
            accessibilityRole="radio"
            accessibilityState={{ checked: option === value }}
            key={option || "none"}
            onPress={() => onChange(option)}
            style={[styles.choice, option === value && styles.choiceSelected]}
          >
            <Text style={styles.choiceText}>{labels[option] ?? option.replace("_", " ")}</Text>
          </Pressable>
        ))}
      </View>
    </View>
  );
}

function LabeledInput({
  keyboardType,
  label,
  onChangeText,
  value,
}: {
  keyboardType?: "number-pad";
  label: string;
  onChangeText: (value: string) => void;
  value: string;
}) {
  return (
    <View style={styles.field}>
      <Text style={styles.label}>{label}</Text>
      <TextInput
        accessibilityLabel={label}
        autoCapitalize="none"
        keyboardType={keyboardType}
        onChangeText={onChangeText}
        style={styles.input}
        value={value}
      />
    </View>
  );
}

function LogoutButton({ onPress }: { onPress: () => void }) {
  return (
    <Pressable accessibilityRole="button" onPress={onPress} style={styles.logout}>
      <Text style={styles.logoutText}>Log out</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  page: { flex: 1, backgroundColor: colors.background },
  content: { gap: spacing.md, padding: spacing.lg, paddingBottom: spacing.lg * 3 },
  centered: {
    flex: 1,
    justifyContent: "center",
    gap: spacing.md,
    padding: spacing.lg,
    backgroundColor: colors.background,
  },
  title: { color: colors.text, fontSize: 30, fontWeight: "700" },
  sectionTitle: { color: colors.text, fontSize: 20, fontWeight: "700" },
  muted: { color: colors.muted, fontSize: 16, lineHeight: 23 },
  empty: { color: colors.primary, fontSize: 15, lineHeight: 21 },
  warning: {
    backgroundColor: colors.warningSurface,
    color: colors.text,
    padding: spacing.md,
    borderRadius: radius.sm,
  },
  error: { color: colors.danger, fontSize: 15, lineHeight: 21 },
  errorPanel: { gap: spacing.sm },
  success: { color: colors.primary, fontSize: 16, fontWeight: "600" },
  field: { gap: spacing.sm },
  label: { color: colors.text, fontSize: 16, fontWeight: "600" },
  input: {
    minHeight: touchTarget,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    color: colors.text,
    paddingHorizontal: spacing.md,
    fontSize: 17,
  },
  choices: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  choice: {
    minHeight: touchTarget,
    justifyContent: "center",
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
  },
  choiceSelected: {
    borderColor: colors.primary,
    borderWidth: 2,
    backgroundColor: colors.warningSurface,
  },
  choiceText: { color: colors.text, fontSize: 15, textTransform: "capitalize" },
  checkRow: { minHeight: touchTarget, flexDirection: "row", alignItems: "center", gap: spacing.md },
  checkbox: { width: 24, height: 24, borderRadius: 4, borderWidth: 2, borderColor: colors.border },
  checkboxChecked: { backgroundColor: colors.primary, borderColor: colors.primary },
  checkLabel: { flex: 1, color: colors.text, fontSize: 16, lineHeight: 22 },
  button: {
    minHeight: touchTarget,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: radius.md,
    backgroundColor: colors.primary,
    paddingHorizontal: spacing.md,
  },
  secondaryButton: {
    minHeight: touchTarget,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.primary,
    paddingHorizontal: spacing.md,
  },
  buttonText: { color: colors.surface, fontSize: 17, fontWeight: "700" },
  secondaryButtonText: { color: colors.primary, fontSize: 16, fontWeight: "700" },
  disabled: { opacity: 0.5 },
  logout: { minHeight: touchTarget, alignItems: "center", justifyContent: "center" },
  logoutText: { color: colors.danger, fontSize: 16, fontWeight: "600" },
});
