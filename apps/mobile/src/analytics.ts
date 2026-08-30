export type AnalyticsEvent =
  | { name: "auth_login_started"; source: "mobile" }
  | { name: "auth_login_completed"; source: "mobile" }
  | { name: "auth_logout_completed"; source: "mobile" }
  | { name: "onboarding_progress_saved"; source: "mobile"; step: number }
  | { name: "onboarding_completed"; source: "mobile"; step: 6 };

type AnalyticsSink = (event: AnalyticsEvent) => void;
let sink: AnalyticsSink = () => undefined;

export function configureAnalyticsSink(next: AnalyticsSink): void {
  sink = next;
}

export function track(event: AnalyticsEvent): void {
  sink(event);
}
