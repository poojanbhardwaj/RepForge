export default function HomePage() {
  return (
    <main>
      <nav aria-label="Primary navigation">
        <a className="brand" href="#top" aria-label="RepForge home">
          RepForge
        </a>
        <div className="nav-actions">
          <a href="#principles">Principles</a>
          <a href="/auth/login?returnTo=/app">Log in</a>
          <a className="button-link compact" href="/auth/login?screen_hint=signup&returnTo=/app">
            Create account
          </a>
        </div>
      </nav>
      <section className="hero" id="top">
        <p className="eyebrow">Built for a calmer gym session</p>
        <h1>Know what to do today—and why tomorrow changes.</h1>
        <p className="lede">
          RepForge is an India-first strength-training product in development. It is designed for
          fast logging, transparent progression, flexible schedules, and realistic nutrition habits.
        </p>
        <p className="notice" role="status">
          Authentication and onboarding are under local verification. No subscriptions are enabled.
        </p>
      </section>
      <section aria-labelledby="principles-heading" className="principles" id="principles">
        <h2 id="principles-heading">The product promise</h2>
        <div className="grid">
          <article>
            <h3>Fast</h3>
            <p>Minimal-tap workout logging designed around the gym floor.</p>
          </article>
          <article>
            <h3>Explainable</h3>
            <p>
              Deterministic recommendations with clear reason codes—not an AI guessing your load.
            </p>
          </article>
          <article>
            <h3>Respectful</h3>
            <p>
              No shame, manipulative streaks, health-data advertising, or fake medical certainty.
            </p>
          </article>
        </div>
      </section>
    </main>
  );
}
