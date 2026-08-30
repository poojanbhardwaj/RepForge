import { redirect } from "next/navigation";

import { getAuth0 } from "../../lib/auth0";
import { OnboardingClient } from "./onboarding-client";

export const dynamic = "force-dynamic";

export default async function ProductPage() {
  const session = await getAuth0().getSession();
  if (!session) redirect("/auth/login?returnTo=/app");

  return <OnboardingClient userName={displayName(session.user)} />;
}

function displayName(user: Record<string, unknown>): string {
  if (typeof user.name === "string" && user.name.trim()) return user.name.trim();
  if (typeof user.nickname === "string" && user.nickname.trim()) return user.nickname.trim();
  return "Athlete";
}
