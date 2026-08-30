"use client";

export default function ProductError({ reset }: { reset: () => void }) {
  return (
    <main className="product-shell">
      <section className="panel" role="alert">
        <h1>We couldn’t open RepForge</h1>
        <p>Your account data was not changed. Retry, or sign in again if your session expired.</p>
        <div className="actions">
          <button type="button" onClick={reset}>
            Try again
          </button>
          <a className="button-link secondary" href="/auth/login?returnTo=/app">
            Sign in again
          </a>
        </div>
      </section>
    </main>
  );
}
