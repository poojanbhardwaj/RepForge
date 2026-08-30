import { configureAnalyticsSink, track, type AnalyticsEvent } from "./analytics";

it("emits only the allowlisted privacy-safe onboarding shape", () => {
  const events: AnalyticsEvent[] = [];
  configureAnalyticsSink((event) => events.push(event));
  track({ name: "onboarding_progress_saved", source: "mobile", step: 3 });
  expect(events).toEqual([{ name: "onboarding_progress_saved", source: "mobile", step: 3 }]);
  expect(JSON.stringify(events)).not.toMatch(/token|email|health/i);
});
