export type WebAnalyticsEvent =
  | { name: "auth_login_selected"; source: "web" }
  | { name: "auth_signup_selected"; source: "web" }
  | { name: "onboarding_progress_saved"; source: "web"; step: number }
  | { name: "onboarding_completed"; source: "web"; step: 6 };

// This allowlisted sink is intentionally local-only until analytics consent and a vendor are approved.
export function track(event: WebAnalyticsEvent): void {
  void event;
}
