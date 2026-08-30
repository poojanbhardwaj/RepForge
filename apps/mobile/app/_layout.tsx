import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Stack } from "expo-router";
import { useState } from "react";
import { Auth0Provider } from "react-native-auth0";

import {
  auth0ClientId,
  auth0Domain,
  mobileAuthMode,
  validateOIDCMobileConfiguration,
} from "../src/auth/config";

export default function RootLayout() {
  validateOIDCMobileConfiguration();
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: { queries: { retry: 1, staleTime: 30_000 }, mutations: { retry: 0 } },
      }),
  );

  const application = (
    <QueryClientProvider client={queryClient}>
      <Stack screenOptions={{ headerTitle: "RepForge", headerBackTitle: "Back" }} />
    </QueryClientProvider>
  );

  if (mobileAuthMode !== "oidc") return application;
  return (
    <Auth0Provider clientId={auth0ClientId} domain={auth0Domain} maxRetries={1} useDPoP={false}>
      {application}
    </Auth0Provider>
  );
}
