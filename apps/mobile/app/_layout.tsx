import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Stack } from "expo-router";
import { useState } from "react";

export default function RootLayout() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: { queries: { retry: 1, staleTime: 30_000 }, mutations: { retry: 0 } },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <Stack screenOptions={{ headerTitle: "RepForge", headerBackTitle: "Back" }} />
    </QueryClientProvider>
  );
}
