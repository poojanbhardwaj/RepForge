import { zodResolver } from "@hookform/resolvers/zod";
import { useNetInfo } from "@react-native-community/netinfo";
import { ApiError, type RepForgeClient } from "@repforge/api-client";
import { colors, radius, spacing, touchTarget } from "@repforge/ui-tokens";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { SafeAreaView } from "react-native-safe-area-context";
import { Controller, useForm } from "react-hook-form";
import { useEffect, useRef } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Text, TextInput, View } from "react-native";

import { apiClient } from "../api";
import { profileSchema, type ProfileForm } from "./schema";

function profileKey(cacheIdentity: string) {
  return ["profile", "me", cacheIdentity] as const;
}

export function ProfileScreen({
  cacheIdentity = "local",
  client = apiClient,
  onLogout,
}: {
  cacheIdentity?: string;
  client?: Pick<RepForgeClient, "getMe" | "updateMe">;
  onLogout?: () => void;
}) {
  const network = useNetInfo();
  const queryClient = useQueryClient();
  const queryKey = profileKey(cacheIdentity);
  const profile = useQuery({
    queryKey,
    queryFn: ({ signal }) => client.getMe(signal),
    enabled: network.isConnected !== false,
    retry: 1,
  });
  const form = useForm<ProfileForm>({
    resolver: zodResolver(profileSchema),
    defaultValues: { displayName: "" },
  });
  const baselineVersion = useRef<number | null>(null);
  const update = useMutation({
    mutationFn: (values: ProfileForm) =>
      client.updateMe({
        displayName: values.displayName.trim(),
        expectedVersion: baselineVersion.current ?? 0,
      }),
    onSuccess: (value) => {
      baselineVersion.current = value.profile.version;
      queryClient.setQueryData(queryKey, value);
      form.reset({ displayName: value.profile.displayName });
    },
    onError: (error) => {
      if (error instanceof ApiError && error.code === "version_conflict") {
        void queryClient.invalidateQueries({ queryKey });
      }
    },
  });

  useEffect(() => {
    if (!profile.data || form.formState.isDirty) return;
    baselineVersion.current = profile.data.profile.version;
    form.reset({ displayName: profile.data.profile.displayName });
  }, [form, form.formState.isDirty, profile.data]);

  if (network.isConnected === false && !profile.data) {
    return (
      <SafeAreaView style={styles.centered}>
        <Text accessibilityRole="header" style={styles.title}>
          You’re offline
        </Text>
        <Text style={styles.muted}>
          Connect to the internet to load your profile. Offline workout logging arrives in a later
          milestone.
        </Text>
      </SafeAreaView>
    );
  }

  if (profile.isPending) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator
          accessibilityLabel="Loading your profile"
          color={colors.primary}
          size="large"
        />
        <Text style={styles.muted}>Loading your profile…</Text>
      </SafeAreaView>
    );
  }

  if (profile.isError || !profile.data) {
    return (
      <SafeAreaView style={styles.centered}>
        <Text accessibilityRole="alert" style={styles.error}>
          We couldn’t load your profile.
        </Text>
        <Pressable
          accessibilityRole="button"
          onPress={() => void profile.refetch()}
          style={styles.button}
        >
          <Text style={styles.buttonText}>Try again</Text>
        </Pressable>
      </SafeAreaView>
    );
  }

  const offline = network.isConnected === false;
  return (
    <SafeAreaView style={styles.page}>
      <View style={styles.card}>
        <Text accessibilityRole="header" style={styles.title}>
          Your profile
        </Text>
        <Text style={styles.muted}>This bootstrap profile uses synthetic local data.</Text>
        {offline ? (
          <Text accessibilityRole="alert" style={styles.offline}>
            Offline: changes are disabled until you reconnect.
          </Text>
        ) : null}
        <Controller
          control={form.control}
          name="displayName"
          render={({ field: { onBlur, onChange, value }, fieldState: { error } }) => (
            <View style={styles.field}>
              <Text nativeID="display-name-label" style={styles.label}>
                Display name
              </Text>
              <TextInput
                accessibilityLabel="Display name"
                accessibilityLabelledBy="display-name-label"
                accessibilityHint="The name shown in your RepForge profile"
                autoCapitalize="words"
                onBlur={onBlur}
                onChangeText={onChange}
                style={[styles.input, error ? styles.inputError : null]}
                value={value}
              />
              {error ? (
                <Text accessibilityRole="alert" style={styles.error}>
                  {error.message}
                </Text>
              ) : null}
            </View>
          )}
        />
        {update.isError ? (
          <Text accessibilityRole="alert" style={styles.error}>
            {update.error instanceof ApiError
              ? update.error.message
              : "We couldn’t save your profile."}
          </Text>
        ) : null}
        {update.isSuccess ? (
          <Text accessibilityLiveRegion="polite" style={styles.success}>
            Profile saved.
          </Text>
        ) : null}
        <Pressable
          accessibilityRole="button"
          accessibilityState={{ disabled: offline || update.isPending }}
          disabled={offline || update.isPending}
          onPress={form.handleSubmit((values) => update.mutate(values))}
          style={({ pressed }) => [
            styles.button,
            pressed && styles.buttonPressed,
            (offline || update.isPending) && styles.buttonDisabled,
          ]}
        >
          <Text style={styles.buttonText}>{update.isPending ? "Saving…" : "Save profile"}</Text>
        </Pressable>
        {onLogout ? (
          <Pressable accessibilityRole="button" onPress={onLogout} style={styles.logout}>
            <Text style={styles.logoutText}>Log out</Text>
          </Pressable>
        ) : null}
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  page: { flex: 1, backgroundColor: colors.background, padding: spacing.md },
  centered: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.md,
    padding: spacing.lg,
    backgroundColor: colors.background,
  },
  card: {
    backgroundColor: colors.surface,
    borderRadius: radius.lg,
    borderColor: colors.border,
    borderWidth: 1,
    padding: spacing.lg,
    gap: spacing.md,
  },
  title: { color: colors.text, fontSize: 28, fontWeight: "700" },
  muted: { color: colors.muted, fontSize: 16, lineHeight: 23 },
  offline: {
    backgroundColor: colors.warningSurface,
    color: colors.text,
    padding: spacing.md,
    borderRadius: radius.sm,
    fontSize: 16,
  },
  field: { gap: spacing.sm },
  label: { color: colors.text, fontSize: 16, fontWeight: "600" },
  input: {
    minHeight: touchTarget,
    borderColor: colors.border,
    borderWidth: 1,
    borderRadius: radius.md,
    paddingHorizontal: spacing.md,
    color: colors.text,
    fontSize: 18,
  },
  inputError: { borderColor: colors.danger, borderWidth: 2 },
  error: { color: colors.danger, fontSize: 15, lineHeight: 21 },
  success: { color: colors.primary, fontSize: 15, fontWeight: "600" },
  button: {
    minHeight: touchTarget,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: radius.md,
    backgroundColor: colors.primary,
    paddingHorizontal: spacing.md,
  },
  buttonPressed: { backgroundColor: colors.primaryPressed },
  buttonDisabled: { opacity: 0.5 },
  buttonText: { color: colors.surface, fontSize: 17, fontWeight: "700" },
  logout: { minHeight: touchTarget, alignItems: "center", justifyContent: "center" },
  logoutText: { color: colors.danger, fontSize: 16, fontWeight: "600" },
});
