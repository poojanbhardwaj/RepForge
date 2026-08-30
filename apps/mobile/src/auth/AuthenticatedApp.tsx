import { useNetInfo } from "@react-native-community/netinfo";
import { colors, radius, spacing, touchTarget } from "@repforge/ui-tokens";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useAuth0 } from "react-native-auth0";

import { track } from "../analytics";
import { createMobileApiClient } from "../api";
import { OnboardingScreen } from "../onboarding/OnboardingScreen";
import { ProfileScreen } from "../profile/ProfileScreen";
import { auth0Audience, auth0CustomScheme, auth0Scopes } from "./config";

type AuthAction = "login" | "signup" | "logout" | null;

function authErrorType(error: unknown): string {
  return typeof error === "object" && error !== null && "type" in error
    ? String((error as { type: unknown }).type)
    : "";
}

export function shouldClearExpiredCredentials(error: unknown): boolean {
  return ["NO_CREDENTIALS", "NO_REFRESH_TOKEN", "RENEW_FAILED", "SESSION_EXPIRED"].includes(
    authErrorType(error),
  );
}

export function authErrorMessage(error: unknown): string | null {
  const type = authErrorType(error);
  if (type === "USER_CANCELLED") return null;
  if (type === "ACCESS_DENIED") {
    return "Verify your email using the message Auth0 sent you, then try again.";
  }
  if (type === "NETWORK_ERROR") {
    return "You appear to be offline. Reconnect and try again.";
  }
  if (type === "NO_NETWORK") {
    return "Your session could not be renewed while offline. Reconnect and retry.";
  }
  if (["NO_CREDENTIALS", "NO_REFRESH_TOKEN", "RENEW_FAILED", "SESSION_EXPIRED"].includes(type)) {
    return "Your session has expired. Sign in again.";
  }
  return "We couldn’t complete authentication. Please try again.";
}

export function AuthenticatedApp() {
  const auth = useAuth0();
  const { clearCredentials, getCredentials, resumeSession } = auth;
  const network = useNetInfo();
  const queryClient = useQueryClient();
  const [action, setAction] = useState<AuthAction>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [restoring, setRestoring] = useState(true);
  const [onboardingComplete, setOnboardingComplete] = useState(false);

  useEffect(() => {
    if (auth.isLoading) return;
    let active = true;
    void resumeSession()
      .catch((error: unknown) => {
        if (active) setMessage(authErrorMessage(error));
      })
      .finally(() => {
        if (active) setRestoring(false);
      });
    return () => {
      active = false;
    };
  }, [auth.isLoading, resumeSession]);

  const getAccessToken = useCallback(async () => {
    try {
      const credentials = await getCredentials(auth0Scopes, 60);
      return credentials.accessToken;
    } catch (error) {
      if (shouldClearExpiredCredentials(error)) {
        setMessage(authErrorMessage(error));
        setOnboardingComplete(false);
        queryClient.clear();
        await clearCredentials().catch(() => {
          setMessage("Your session expired, but its local credentials could not be cleared.");
        });
      }
      throw error;
    }
  }, [clearCredentials, getCredentials, queryClient]);
  const client = useMemo(() => createMobileApiClient(getAccessToken), [getAccessToken]);

  const login = async (signup: boolean) => {
    setAction(signup ? "signup" : "login");
    setMessage(null);
    track({ name: "auth_login_started", source: "mobile" });
    try {
      await auth.authorize(
        {
          audience: auth0Audience,
          scope: auth0Scopes,
          ...(signup ? { additionalParameters: { screen_hint: "signup" } } : {}),
        },
        { customScheme: auth0CustomScheme },
      );
      track({ name: "auth_login_completed", source: "mobile" });
    } catch (error) {
      setMessage(authErrorMessage(error));
    } finally {
      setAction(null);
    }
  };

  const logout = async () => {
    setAction("logout");
    setMessage(null);
    let localSessionCleared = false;
    try {
      await auth.clearSession(undefined, { customScheme: auth0CustomScheme });
      localSessionCleared = true;
      track({ name: "auth_logout_completed", source: "mobile" });
    } catch {
      try {
        await clearCredentials();
        localSessionCleared = true;
        setMessage(
          network.isConnected === false
            ? "Signed out on this device. Reconnect before signing in again to finish the Auth0 logout."
            : "Signed out on this device, but the Auth0 browser logout did not finish.",
        );
      } catch {
        setMessage("We couldn’t securely clear your local session. Try logging out again.");
      }
    } finally {
      if (localSessionCleared) {
        queryClient.clear();
        setOnboardingComplete(false);
      }
      setAction(null);
    }
  };

  if (auth.isLoading || restoring) {
    return (
      <SafeAreaView style={styles.centered}>
        <ActivityIndicator accessibilityLabel="Restoring your secure session" size="large" />
        <Text style={styles.muted}>Restoring your secure session…</Text>
      </SafeAreaView>
    );
  }

  if (!auth.user) {
    const disabled = action !== null || network.isConnected === false;
    return (
      <SafeAreaView style={styles.centered}>
        <View style={styles.card}>
          <Text accessibilityRole="header" style={styles.title}>
            Welcome to RepForge
          </Text>
          <Text style={styles.muted}>
            Sign in securely in your browser. RepForge never asks for your Auth0 password inside the
            app.
          </Text>
          {network.isConnected === false ? (
            <Text accessibilityRole="alert" style={styles.error}>
              You’re offline. Reconnect to sign in or create an account.
            </Text>
          ) : null}
          {message ? (
            <Text accessibilityRole="alert" style={styles.error}>
              {message}
            </Text>
          ) : null}
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ disabled }}
            disabled={disabled}
            onPress={() => void login(false)}
            style={[styles.button, disabled && styles.disabled]}
          >
            <Text style={styles.buttonText}>{action === "login" ? "Opening…" : "Log in"}</Text>
          </Pressable>
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ disabled }}
            disabled={disabled}
            onPress={() => void login(true)}
            style={[styles.secondaryButton, disabled && styles.disabled]}
          >
            <Text style={styles.secondaryButtonText}>
              {action === "signup" ? "Opening…" : "Create account"}
            </Text>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  if (message) {
    return (
      <SafeAreaView style={styles.centered}>
        <View style={styles.card}>
          <Text accessibilityRole="header" style={styles.title}>
            Your session needs attention
          </Text>
          <Text accessibilityRole="alert" style={styles.error}>
            {message}
          </Text>
          <Pressable
            accessibilityRole="button"
            onPress={() => setMessage(null)}
            style={styles.secondaryButton}
          >
            <Text style={styles.secondaryButtonText}>Try again</Text>
          </Pressable>
          <Pressable
            accessibilityRole="button"
            disabled={action !== null}
            onPress={() => void logout()}
            style={[styles.button, action !== null && styles.disabled]}
          >
            <Text style={styles.buttonText}>
              {action === "logout" ? "Logging out…" : "Log out"}
            </Text>
          </Pressable>
        </View>
      </SafeAreaView>
    );
  }

  const cacheIdentity = auth.user.sub;

  if (!onboardingComplete) {
    return (
      <OnboardingScreen
        cacheIdentity={cacheIdentity}
        client={client}
        onComplete={() => setOnboardingComplete(true)}
        onLogout={() => void logout()}
      />
    );
  }

  return (
    <ProfileScreen cacheIdentity={cacheIdentity} client={client} onLogout={() => void logout()} />
  );
}

const styles = StyleSheet.create({
  centered: {
    flex: 1,
    justifyContent: "center",
    padding: spacing.lg,
    backgroundColor: colors.background,
  },
  card: {
    gap: spacing.md,
    padding: spacing.lg,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.lg,
    backgroundColor: colors.surface,
  },
  title: { color: colors.text, fontSize: 30, fontWeight: "700" },
  muted: { color: colors.muted, fontSize: 16, lineHeight: 23 },
  error: { color: colors.danger, fontSize: 15, lineHeight: 21 },
  button: {
    minHeight: touchTarget,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: radius.md,
    backgroundColor: colors.primary,
  },
  secondaryButton: {
    minHeight: touchTarget,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.primary,
  },
  buttonText: { color: colors.surface, fontSize: 17, fontWeight: "700" },
  secondaryButtonText: { color: colors.primary, fontSize: 17, fontWeight: "700" },
  disabled: { opacity: 0.5 },
});
