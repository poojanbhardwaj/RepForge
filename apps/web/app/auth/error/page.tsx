const messages: Record<string, string> = {
  access_denied:
    "Sign in was denied. If you just created your account, verify your email and then try again.",
  authentication_failed:
    "We couldn’t complete sign in. Try again without changing the callback URL.",
};

export default async function AuthenticationErrorPage({
  searchParams,
}: {
  searchParams: Promise<{ code?: string }>;
}) {
  const code = (await searchParams).code ?? "authentication_failed";
  const message = messages[code] ?? messages.authentication_failed;
  return (
    <main className="product-shell">
      <section className="panel" role="alert">
        <h1>Sign in needs attention</h1>
        <p>{message}</p>
        <a className="button-link" href="/auth/login?returnTo=/app">
          Try sign in again
        </a>
      </section>
    </main>
  );
}
