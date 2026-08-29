# RepForge: Production Fitness App — Market Brief and Codex Master Prompt

Prepared: 27 August 2026  
Working name: **RepForge** (rename before launch)  
Initial market: India; architecture must support global expansion  
Primary audience: adults (18+) who are beginner-to-intermediate gym users

## How to use this file

1. Create an empty Git repository and put this file in its root.
2. Open that repository in Codex using Plan mode and give Codex the entire section titled **Master bootstrap prompt** once.
3. Let Codex finish the bootstrap phase and review its plan, commands, tests, and diff.
4. Use the numbered follow-up prompts one at a time. Do not ask one agent turn to build the entire production app.
5. Keep this file in the repository as `docs/PRODUCT_BRIEF.md` after bootstrap so later work can refer to it.

The structure follows current official Codex guidance: give the agent a clear goal, context, constraints, and definition of done; plan complex work first; place durable repository rules in `AGENTS.md`; and require tests and a final review. See [OpenAI's Codex best practices](https://learn.chatgpt.com/guides/best-practices).

## Important scope statement

No useful analysis can literally inspect every fitness app in every country. This is a representative 2026 scan of category leaders across social endurance, nutrition, strength logging, adaptive programming, free content, instructor-led classes, and India-focused coaching. The strategic gaps below are inferences from publicly documented product scope, not claims that a competitor is technically incapable of a feature.

## Market scan

| Product            | Category                           | Documented strengths                                                                                                          | Product gap we can exploit (inference)                                                                                      |
| ------------------ | ---------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Strava             | Endurance + social network         | Excellent activity feed, clubs, challenges, segments, routes, GPS analysis, and device ecosystem                              | Not designed as an end-to-end strength progression and nutrition adherence product                                          |
| MyFitnessPal       | Nutrition logging                  | Large food-logging workflow, calorie/macro tracking, recipes, meal plans, barcode/meal/voice logging in paid tiers            | Training prescription and set-by-set strength progression are not the core experience; detailed logging can create friction |
| HealthifyMe        | India-focused nutrition + coaching | Indian food database, photo meal logging, AI guidance, diet/workout suggestions, and human coaching                           | Deep, transparent strength-program progression and fast gym-session logging are not the main wedge                          |
| Fitbod             | Adaptive strength                  | Personalized strength sessions based on goals, equipment, progressive overload, and recovery                                  | Premium positioning; nutrition, India localization, and friend accountability are not its central proposition               |
| Hevy               | Strength logger + social           | Fast workout logging, routines, analytics, social feed, routine sharing, and a strong free tier                               | Rich adaptive prescription and nutrition are not the primary value proposition                                              |
| Strong             | Strength logger                    | Focused logging UX, supersets, custom exercises, RPE, timers, charts, warm-up/plate calculators, export, and wearable support | User largely supplies the program; limited end-to-end coaching or nutrition loop                                            |
| JEFIT              | Planner + analytics + community    | Large planning/logging surface, adaptive plans, fatigue-aware recommendations, analytics, and community                       | Breadth risks a crowded experience; an opinionated beginner journey can be simpler and more trustworthy                     |
| Freeletics         | Adaptive digital coach             | Personalized bodyweight, HIIT, strength, weights, cardio, and running; adapts to goals and constraints                        | Less specialized for barbell/gym logging, Indian nutrition, and transparent progression decisions                           |
| Nike Training Club | Free expert content                | 200+ free workouts/programs across strength, yoga, conditioning, recovery, and mindfulness with polished trainer-led content  | Content library rather than a deeply adaptive log-feedback-progress system; hard to compete on free video volume            |
| Peloton            | Instructor-led content + community | High-quality instructors and a wide class catalog across strength, cycling, running, yoga, meditation, and more               | Expensive content operation; less suited to a lean startup whose core advantage is adaptive gym programming                 |
| StrengthLog        | Strength education + logging       | Clean, no-ad strength log, programs, and lifter-focused content                                                               | Smaller social/local-nutrition opportunity; coaching loop can be more personalized                                          |

### Source snapshot

- [Strava product features](https://www.strava.com/) document tracking, community, routes, segments, clubs, and challenges.
- [MyFitnessPal free features](https://support.myfitnesspal.com/hc/en-us/articles/15457546881805-What-is-included-in-the-free-version) and [Premium+ features](https://support.myfitnesspal.com/hc/en-us/articles/34347930588557-Premium) document nutrition logging and premium capture methods.
- [HealthifyMe's app page](https://www.healthifyme.com/app/) documents lifestyle trackers, Indian foods, human coaches, and its AI coach; its [current India page](https://www.healthifyme.com/in/) documents photo logging and personalized guidance.
- [Fitbod](https://fitbod.me/) documents goal-, level-, equipment-, recovery-, and progressive-overload-based workouts.
- [Hevy features](https://www.hevyapp.com/features/) describe logging, progress, and socializing as its three pillars.
- [Strong](https://www.strong.app/) documents its focused logger, RPE, calculators, charts, export, integrations, and custom routines.
- [JEFIT](https://www.jefit.com/) documents planning, metrics, community, and adaptive plans.
- [Freeletics](https://www.freeletics.com/) documents tailored multi-modality training.
- [Nike Training Club](https://www.nike.com/gb/ntc-app) documents 200+ free workouts and programs.
- [Peloton classes](https://www.onepeloton.com/classes) documents its instructor-led class range.
- [StrengthLog](https://www.strengthlog.com/) documents its free, no-ad strength log and programs.

Features and prices change. Re-check primary sources and store policies immediately before launch.

## Recommended product strategy

### Positioning

**RepForge tells an Indian gym user exactly what to do today, lets them log it with almost no friction, explains why the next session changed, and connects training progress to realistic local nutrition habits.**

Do not begin as an all-sports super-app. Win one repeated loop:

1. Receive a safe, achievable workout.
2. Log sets quickly, including offline.
3. Report reps-in-reserve, pain flags, energy, and session difficulty.
4. Get a transparent next-session adjustment.
5. See a small, meaningful progress signal.
6. Follow simple calorie/protein targets with Indian foods and household units.

### Differentiators

- **Transparent adaptation:** show the rule behind load, rep, exercise, and deload changes. Use deterministic training rules first; an LLM must never be the source of truth for load progression.
- **Flexible rather than rigid:** the user can swap equipment, shorten a session, add bodyweight exercises, fast on selected days, and keep their program structure.
- **India-ready:** kilograms, centimeters, vegetarian/eggetarian/vegan preferences, Indian foods and household measures, INR pricing, UPI-ready web checkout, and low-bandwidth/offline behavior.
- **Fast gym UX:** one-handed set logging, smart previous values, rest timer, plate calculator, warm-up sets, RPE/RIR, and no modal maze.
- **Trust:** no medical promises, no sale or advertising use of health data, export/delete controls, visible calculation explanations, and human-readable safety boundaries.

### What not to build in the first release

- A social feed, public leaderboards, live classes, trainer marketplace, wearable integrations, meal-photo AI, body-form computer vision, supplements store, medical-condition coaching, minors' accounts, or custom ML training.
- Microservices, Kubernetes, event sourcing, GraphQL, a data lake, or multi-region active-active infrastructure.
- A proprietary exercise-video studio. Seed licensed or internally owned media only; placeholders are acceptable in development.

## Business model

### Free tier

- Onboarding and one starter program
- Reliable workout logging, offline queue, timers, and workout history
- Up to three saved routines
- Basic progress charts and personal records
- Manual nutrition target and food logging
- Data export and account deletion

### Pro tier

- Adaptive programming and automatic next-session recommendations
- Unlimited programs/routines and exercise substitutions
- Advanced analytics, fatigue trends, volume landmarks, and plateau detection
- Flexible schedule/regeneration tools
- Rich nutrition insights and meal templates
- Optional AI explanation/chat features with strict safety guardrails

### Suggested price hypotheses, not facts

- Founding beta: ₹299/month or ₹1,999/year
- Standard candidate: ₹399/month or ₹2,499/year
- Seven-day trial only after the core value is visible; clearly show renewal and cancellation terms
- Test annual-first versus monthly-first presentation. Never hard-code price text outside billing-product configuration.

### Later revenue, only after retention is proven

- Coach marketplace with a transparent platform fee
- Gym/college group licenses
- Paid expert programs with revenue sharing
- Corporate wellness plans that do not expose individual health data to employers

Do not use targeted advertising based on health or fitness data. Apple explicitly restricts advertising, marketing, and data-mining uses of health/fitness data in its [App Review Guidelines](https://developer.apple.com/app-store/review/guidelines/).

### Metrics and commercial gates

Instrument the funnel but do not invent success numbers in code. Start with hypotheses and revise from cohort data:

- Acquisition: landing-page to install/signup rate, cost per activated user
- Activation: onboarding completed and first workout logged within 72 hours
- Value: two workouts completed in the first seven days
- Retention: W1, W4/D30, and eight-week program retention by acquisition cohort
- Engagement: planned/completed ratio, logs per workout, program changes, notification opt-out
- Monetization: trial start, trial-to-paid, monthly/annual mix, refund rate, churn, grace-period recovery
- Quality: crash-free sessions, API error rate, sync-conflict rate, workout-save p95 latency
- Safety: pain/injury flags, unsafe recommendation reports, nutrition-warning triggers

Only scale paid acquisition once retention and contribution margin make it rational. Revenue is an outcome of repeated user value, not an MVP feature.

---

# Master bootstrap prompt

You are the founding staff engineer, product-minded architect, security reviewer, and delivery owner for a production-grade fitness product. Work inside the current Git repository. The product brief is in `FITNESS_APP_CODEX_MASTER_PROMPT.md`; treat it as product context, but create concise durable repository documentation from it.

## Goal

Build **RepForge**, a commercially viable, India-first adaptive strength-training and nutrition adherence app for adults. The product must be reliable enough to progress from local development to a closed beta and then production without a rewrite. The first release centers on fast workout logging, transparent deterministic progression, flexible schedules, basic nutrition, subscriptions, and safe user-data handling.

Do not attempt every feature in one agent turn. First inspect the repository, produce an execution plan, create the durable architecture and engineering rules, and implement only the bootstrap vertical slice specified below. Future prompts will advance one milestone at a time.

## Required working behavior

1. Start by inspecting the repository and toolchain. Preserve any existing user work.
2. If a decision is genuinely blocking and has a large downstream cost, ask one concise question. Otherwise state the assumption in an ADR and proceed.
3. Use current stable supported versions available in the environment. Verify versions from official release documentation, pin them, and record them in `docs/DEPENDENCIES.md`. Do not guess "latest" version numbers.
4. Before coding, create/update `PLANS.md` with milestones, dependencies, risks, acceptance criteria, and current status.
5. Keep `AGENTS.md` short and operational: layout, exact run/test/lint commands, conventions, safety rules, and definition of done. Put detailed product material in `docs/`.
6. Implement in small reviewable slices. At the end of each slice run formatting, static analysis, unit/integration tests, and builds that are relevant to changed code.
7. Review the final diff for security, privacy, broken behavior, migration hazards, accessibility, and accidental secrets. Report commands actually run and distinguish passing, failing, and not-run checks.
8. Never claim production readiness merely because a skeleton compiles. Maintain `docs/PRODUCTION_READINESS.md` with explicit incomplete gates.
9. Do not commit, push, deploy, purchase services, create paid resources, or use real production credentials unless explicitly requested.

## Fixed technology direction

Use a monorepo with:

- **Backend:** Go, standard `net/http` compatible router, explicit dependency injection, SQL-first persistence, PostgreSQL, Redis, and an S3-compatible object-store interface.
- **Mobile:** React Native with Expo and TypeScript. Use Expo Router, TanStack Query, React Hook Form, Zod, and an accessible component/theme layer. Workout logging must be designed for offline use.
- **Web:** Next.js + TypeScript for marketing pages and an authenticated admin/operations console. Do not duplicate the consumer mobile app on web during MVP.
- **Contract:** REST JSON described by OpenAPI 3.1. Generate or validate typed clients from the contract; do not maintain drifting hand-written request types.
- **Local platform:** Docker Compose for PostgreSQL, Redis, object storage, and an OIDC-compatible local identity provider or a well-defined test identity adapter.
- **Production reference:** AWS in `ap-south-1` using Terraform, with a container service, managed PostgreSQL, managed Redis, object storage/CDN, secret manager, email provider, WAF/rate limiting, backups, and separate staging/production state. Keep cloud modules optional during early local work.
- **CI:** GitHub Actions for format, lint, tests, OpenAPI checks, migration checks, dependency/security scanning, and builds. Release/deploy jobs require protected environments and must not run during bootstrap.
- **Observability:** structured logs with request/trace IDs, OpenTelemetry traces/metrics, health/readiness endpoints, error tracking adapter, and product analytics behind consent/configuration.

Prefer a **modular monolith plus worker process**, not microservices. Modules own their domain rules and persistence interfaces. They may communicate in-process and through a transactional outbox for asynchronous side effects. Design extraction seams but do not extract services prematurely.

## Proposed repository layout

Adapt only when the environment gives a concrete reason:

```text
/
  AGENTS.md
  PLANS.md
  Makefile
  compose.yaml
  .env.example
  apps/
    mobile/
    web/
  backend/
    cmd/api/
    cmd/worker/
    internal/
      auth/
      users/
      training/
      workouts/
      progression/
      nutrition/
      billing/
      notifications/
      analytics/
      platform/
    migrations/
    openapi/
  packages/
    api-client/
    ui-tokens/
    config/
  infra/
    terraform/
  docs/
    adr/
    runbooks/
```

Do not add empty decorative directories. Add a directory when its first useful file exists.

## Domain boundaries and minimum data model

Use UUIDv7 or another justified sortable opaque identifier; never expose sequential database IDs. Store timestamps in UTC and present them in the user's timezone. Use integer minor currency units and explicit ISO currency codes. Use decimal-safe or integer representations for body and lifting measurements; do not use binary floating point for money.

Minimum entities, refined through migrations:

- identity reference, user, profile, consent/version, goal, preference, equipment access
- exercise, exercise alias, muscle/equipment metadata, substitution relationship
- program, program version, training day, prescription, progression policy
- workout, workout exercise, set log, session feedback, pain flag, personal record
- body metric, daily readiness check-in
- nutrition target, food item/source/verification status, serving unit, meal/food log
- billing customer, provider subscription, entitlement, webhook event
- device/push token, notification preference/job
- outbox event, idempotency record, audit event

Separate personally identifying profile data from high-volume workout events where practical. Every user-owned table must have an explicit ownership check in its query path. Add retention/deletion/export design before collecting optional sensitive data.

## API conventions

- Version routes under `/v1` and use consistent error envelopes with a stable machine code, safe message, details, and trace ID.
- Validate at the boundary and keep domain validation in the domain layer.
- Cursor pagination for growing collections; explicit sort order.
- Idempotency keys on workout completion, offline mutation replay, billing actions, and other retryable creates.
- Optimistic concurrency/version fields for offline-editable resources.
- Strong request body limits, timeouts, cancellation propagation, rate limits, and safe CORS configuration.
- Never log tokens, passwords, authorization headers, health/nutrition payloads, or full request bodies.
- Produce examples in OpenAPI and contract tests that prove handlers match the specification.

## Authentication and authorization

- Use OAuth 2.1/OIDC Authorization Code + PKCE for mobile and web. Backend verifies issuer, audience, signature, expiry, and key rotation.
- Hide the identity vendor behind an adapter. Local development must work without real vendor secrets.
- Roles: user, support-readonly, content-editor, admin. Default deny. Sensitive support actions require a reason and produce an audit event.
- Never build an insecure "temporary" production password system. Test bypasses must be impossible when the environment is not local/test.
- Short-lived access tokens; follow the selected provider's supported refresh/session model. Do not store bearer tokens in insecure mobile storage.

## Workout and progression rules

The workout state machine must be explicit, e.g. scheduled → in_progress → completed/abandoned, with safe resume and offline replay. A set log supports warm-up/working/drop/failure types, reps, load, duration or distance where applicable, RPE/RIR, completion time, and notes.

Implement deterministic, versioned progression policies with unit tests and human-readable reason codes. Initial policy should support:

- double progression inside a configured rep range
- smallest available load increment and kg/lb conversion without drift
- progression only when prescribed working sets meet the top of the rep range at the target RIR for the configured number of exposures
- maintain or regress after missed targets using conservative rules
- deload suggestion from configurable signals such as repeated performance decline plus elevated session difficulty; never diagnose overtraining
- exercise substitution that preserves movement pattern and equipment constraints without silently comparing incompatible history
- manual override that records the recommendation, user choice, and reason

Never increase a load because an LLM said so. AI may explain a structured recommendation produced by the rules engine, but cannot modify it. Pain flags stop automatic progression for the affected movement and show a non-diagnostic safety message encouraging appropriate professional help.

## Nutrition boundaries

- Start with energy/protein targets, manual food creation, meal templates, verified/unverified food-source labels, Indian household measures, and flexible daily targets.
- Every calculated target shows its inputs, formula/version, uncertainty, and an edit option.
- Do not prescribe crash diets, drugs, supplements, eating-disorder behavior, or plans for pregnancy, minors, serious disease, or medical conditions. Route these scenarios to a clear safety boundary.
- Food photo recognition and generative meal planning are later experiments, not MVP dependencies.
- Never represent crowd-entered nutrition data as verified.

## Billing and entitlements

- Model entitlement separately from payment-provider subscription state.
- Use provider adapters and signed, replay-safe webhooks. Store provider event IDs, verify signatures against the raw request body, process idempotently, and retain an auditable state transition.
- Mobile digital subscriptions must follow current Apple App Store and Google Play billing rules. A provider such as RevenueCat may normalize mobile entitlements. Web billing may use Razorpay for India and a global provider later, but do not use web checkout inside mobile in violation of store rules.
- Implement grace periods, cancellation-at-period-end, refunds/revocations, restore purchases, webhook reordering, and customer-support reconciliation before production.
- Never unlock Pro from a client-supplied boolean or an unverified redirect.
- Razorpay supports subscription plan/cycle APIs documented in its [subscription API](https://razorpay.com/docs/api/payments/subscriptions/create-subscription/), but integration must remain behind an adapter.

## Privacy, safety, and compliance requirements

Treat fitness, nutrition, body metrics, and behavior data as sensitive even where a law uses a narrower definition.

- Data minimization, purpose limitation, explicit versioned consent, least privilege, encryption in transit and at rest, deletion, export, retention schedules, and incident response.
- Prepare for India's [Digital Personal Data Protection Act, 2023](https://www.meity.gov.in/static/uploads/2024/06/2bf1f0e9f04e6fb4f8fef35e82c42aa5.pdf) and the phased [Digital Personal Data Protection Rules, 2025](https://www.meity.gov.in/static/uploads/2025/11/53450e6e5dc0bfa85ebd78686cadad39.pdf). Record that qualified legal review is required before launch; code comments are not legal advice.
- Google Play requires health-app declarations and privacy disclosures; track release evidence using the current [Health Content and Services policy](https://support.google.com/googleplay/android-developer/answer/16679511?hl=en).
- Apple limits advertising/marketing use of health and fitness data; review the current [App Review Guidelines](https://developer.apple.com/app-store/review/guidelines/) before submission.
- No medical diagnosis, treatment claims, emergency function, or guarantee of results. Show appropriate disclaimer and emergency guidance without using disclaimers to excuse unsafe product logic.
- Make account export/deletion self-service. Deletion jobs must be resumable, auditable, and tested, including object storage and analytics identifiers.
- Do not use production user data in local development, demos, tests, screenshots, or AI prompts.

Create a threat model covering broken object authorization, token theft, webhook forgery, offline replay, sensitive log leakage, account enumeration, credential stuffing, admin abuse, unsafe recommendations, supply-chain risk, backup exposure, and deletion failure.

## Security architecture and non-negotiable release gates

Security is a continuous engineering responsibility and a release blocker. No prompt, framework, cloud provider, or audit can guarantee that a breach will never occur. The goal is to minimize stored sensitive data, prevent unauthorized access through independent layers, detect abuse quickly, limit blast radius, and recover safely.

Use these external baselines and record requirement-to-evidence mappings in `docs/security/`:

- Meet all applicable [OWASP ASVS 5.0](https://owasp.org/www-project-application-security-verification-standard/) Level 2 requirements, plus risk-selected Level 3 requirements for authentication, authorization, cryptography, administration, and sensitive health/fitness data.
- Assess Android and iOS releases against [OWASP MASVS](https://mas.owasp.org/MASVS/) and execute relevant tests from the [OWASP Mobile Application Security Testing Guide](https://mas.owasp.org/MASTG/).
- Integrate the final [NIST Secure Software Development Framework](https://csrc.nist.gov/pubs/sp/800/218/final) practices into planning, implementation, review, release, and vulnerability response.
- Apply [NIST zero-trust principles](https://csrc.nist.gov/pubs/sp/800/207/final): network location alone never grants trust; authenticate and authorize every protected action and service identity.

Required controls:

- **Data inventory and minimization:** maintain a machine-readable inventory of collected fields, purpose, legal basis/consent, sensitivity, storage location, recipients, retention, deletion path, and owner. Do not collect a field until this record exists.
- **Encryption:** TLS 1.2+ in transit with modern configuration; managed encryption at rest for databases, replicas, backups, logs, queues, caches, and object storage. Use envelope/application-level encryption for the most sensitive fields where the threat model justifies it. Keys live in managed KMS/HSM-backed services, are separated by environment and purpose, have least-privilege policies, rotation procedures, access alerts, and tested recovery. Never invent custom cryptography.
- **Secrets:** no credentials in source, mobile bundles, container images, build logs, analytics, crash reports, shell history, or Terraform state. Use workload identity and short-lived credentials where possible; otherwise use a managed secret store with rotation. Secret scanning runs locally and in CI, including commit history and generated artifacts.
- **Authorization:** every user-owned object is authorized server-side on every read and write. Use deny-by-default policy, negative authorization tests, separate service identities, least-privilege database roles, short-lived admin sessions, mandatory MFA for staff, auditable privileged access, and no shared administrator accounts.
- **Infrastructure isolation:** production databases, caches, queues, and internal administration endpoints are not publicly reachable. Separate production from staging accounts/projects and credentials. Use private networking, restricted egress where practical, WAF/API throttling, DDoS protections, hardened base images, read-only containers where possible, and infrastructure-as-code review.
- **API and web:** protect against broken object authorization, injection, request smuggling, SSRF, unsafe file uploads, mass assignment, XSS, CSRF where applicable, open redirects, cache leakage, account enumeration, credential stuffing, and webhook forgery. Use parameterized queries, allowlists, strict parsers, body/time limits, safe response headers, restrictive CORS, and a tested Content Security Policy for web.
- **Mobile:** bearer/session material only in iOS Keychain or Android Keystore-backed secure storage; never plaintext AsyncStorage. Minimize offline sensitive data, encrypt the local database with OS-protected keys when supported and threat-model appropriate, redact app-switcher previews, exclude sensitive files from backups, prevent sensitive notification text by default, and test logs, deep links, clipboard, exported components, screenshots, and rooted/jailbroken-device behavior. The server never trusts client integrity or client-calculated entitlements.
- **Logs and analytics:** default-deny telemetry schema. No access tokens, secrets, raw profile/body/nutrition/workout payloads, free-text notes, full IP addresses beyond justified security retention, or payment details. Apply structured allowlisted fields, redaction tests, restricted access, short retention, integrity protection, and alerts for abnormal access/export patterns.
- **Payments:** never store card data. Use hosted/provider SDK flows, verify signed callbacks/webhooks, restrict provider keys, and keep payment identifiers separate from fitness data. Document PCI scope with the provider and a qualified reviewer.
- **Supply chain:** lock dependencies, verify checksums/signatures where supported, generate an SBOM for releases, scan source/dependencies/images/IaC, pin third-party CI actions by immutable commit, restrict package install scripts, protect branches/environments, require review for workflow and dependency changes, and maintain a vulnerability patch SLA.
- **Backups and deletion:** encrypt backups with separate access controls, prevent public snapshots, monitor exports, test point-in-time restore, and ensure deletion/retention jobs cover replicas, caches, objects, search/analytics stores, and eventual backup expiry. Restore drills must not copy production data into lower environments.
- **Third parties:** maintain a vendor/subprocessor inventory, data-flow diagram, contracts/DPA status, permissions, data location/retention, breach contact, and removal procedure. Send only the minimum data required. No production health data may be sent to an LLM unless explicitly designed, consented, legally reviewed, redacted/minimized, contractually protected, and approved through a separate security review.
- **Detection and response:** centralized security alerts for suspicious authentication, privilege changes, bulk reads/exports, KMS/secret access, WAF blocks, webhook failures, and configuration drift. Maintain tested incident, credential/key compromise, data breach, ransomware, and third-party compromise runbooks with decision owners and evidence-preservation steps.
- **Security testing:** threat model every milestone; use SAST, dependency/container/IaC/secret scanning, DAST against staging, authorization and abuse-case tests, fuzzing for parsers, and manual review of authentication, billing, file handling, admin, export, and deletion paths. Never run intrusive tests against production without an approved test plan.

Hard release gates:

1. No unresolved critical or high-severity security vulnerability. A medium-risk exception requires written impact, compensating controls, owner, and expiry date.
2. ASVS/MASVS evidence matrix is complete for applicable controls; “not applicable” entries include a reason.
3. An independent qualified penetration test covers API, web, mobile builds, cloud configuration, authorization, and business-logic abuse before public launch; findings are remediated and critical/high fixes are independently retested.
4. Secrets, keys, staff MFA, access reviews, backup restore, deletion/export, alert delivery, incident escalation, dependency provenance, and rollback have current test evidence.
5. A documented breach-response drill and production access review have been completed. Security and legal owners approve the residual-risk register.
6. These gates are re-evaluated for every material authentication, billing, data-sharing, admin, AI, or infrastructure change and at least before each public release.

## Reliability and performance targets

Record targets as service objectives, then measure them rather than asserting them:

- No acknowledged completed workout may be lost.
- Offline workout mutations are durable locally, ordered, idempotent, and visibly reconciled.
- API availability target: 99.9% after production launch, excluding announced maintenance.
- Interactive read p95 target under 300 ms and mutation p95 under 500 ms at the API boundary under the documented beta load profile, excluding third-party latency.
- Graceful shutdown, database pool limits, bounded worker concurrency, retries with jitter, dead-letter handling, and backpressure.
- Automated PostgreSQL backups with restore drills; define RPO/RTO hypotheses and validate them before launch.
- `/healthz` proves process health; `/readyz` checks only dependencies required to serve traffic and uses tight timeouts.

## Quality requirements

- Go: formatting, vet/static analysis, race tests for concurrency-sensitive packages, unit tests, repository integration tests with real PostgreSQL/Redis containers, migration up/down or forward/rollback strategy, fuzz/property tests for critical parsers and progression invariants.
- TypeScript: strict mode, lint/format, unit/component tests, API contract validation, accessibility checks, and E2E coverage for critical web flows.
- Mobile: test airplane mode, process death during a workout, repeated sync, conflicting edits, background/foreground transitions, timezone/daylight-saving behavior, metric/imperial switching, low-memory behavior, and accessible touch targets/screen-reader labels.
- Billing: fixture-driven signature tests, duplicates, reordering, delayed events, refund/revocation, grace period, cancellation, and restore.
- Security: ASVS/MASVS control mapping, secret scanning, SAST, dependency/container/IaC scanning, SBOM generation, DAST against staging, authorization matrix tests, rate-limit tests, fuzzing, mobile static/dynamic tests, and an abuse-case checklist.
- Use seeded synthetic fixtures only. Tests must be deterministic and independent of live third-party APIs.

## UX and accessibility principles

- Calm, athletic, trustworthy visual direction; avoid copied competitor branding and generic neon-bodybuilder styling.
- In-workout set entry is the most important screen: large controls, minimal taps, previous performance visible, undo, haptic/sound settings, and sunlight/dark-mode readability.
- WCAG 2.2 AA targets for web; equivalent mobile accessibility practices, dynamic type, screen readers, reduced motion, non-color-only states, and 44×44 minimum touch targets.
- Every loading, empty, offline, sync-conflict, partial-failure, and destructive-confirmation state must be designed.
- No manipulative streak loss, fake scarcity, hidden renewal, obstructive cancellation, or shame-based copy.

## Bootstrap vertical slice to implement now

In this first run, produce a working foundation, not the full application:

1. Create concise `AGENTS.md`, `PLANS.md`, `README.md`, and docs for PRD, architecture, ADRs, threat model, data classification, data-flow/vendor inventory, ASVS/MASVS evidence mapping, security test plan, incident response, production readiness, dependencies, API conventions, and local runbook.
2. Scaffold the Go API and worker, Expo mobile app, Next.js web app, shared packages, Docker Compose, `.env.example`, and CI. Do not put real secrets in files.
3. Implement API `/healthz`, `/readyz`, version/build information, structured request logging with redaction, consistent error responses, graceful shutdown, and configuration validation.
4. Create the first PostgreSQL migration for a minimal user profile plus consent record, a repository implementation, and one authenticated `/v1/me` read/update vertical slice using the local/test identity adapter. Keep the adapter impossible to enable in staging/production.
5. Define the initial OpenAPI 3.1 contract and generate/validate a TypeScript client used by one mobile profile screen. The screen must cover loading, error, success, edit validation, offline/unavailable messaging, and accessibility basics.
6. Add representative backend unit/integration tests and frontend unit/component tests. CI must run them without real cloud credentials.
7. Provide a `make help` or equally discoverable command surface for setup, dev, generate, format, lint, test, integration test, build, and clean. Any clean command must target only known project-generated paths.
8. Run the relevant checks. End with: outcome, changed files, architecture decisions, tests/checks with exact results, known limitations, security notes, and the exact recommended next prompt.

## Bootstrap definition of done

- A new contributor can follow `README.md` from a clean checkout to run the infrastructure, API, mobile/web development commands, and tests.
- The API, database migration, `/v1/me` flow, generated contract client, and profile UI form one real tested vertical slice.
- Local/test authentication is clearly isolated and fails closed outside local/test.
- No secrets, fake production claims, unbounded TODO dump, or silently failing checks.
- Documentation and code agree on commands, ports, environment variables, and directory names.
- `PLANS.md` clearly marks what is done and what remains.

Start by inspecting the repository and presenting a concrete plan. Then execute the bootstrap slice unless a truly blocking decision requires my answer.

---

# Follow-up prompts

Use these in order after reviewing the prior milestone. Each prompt assumes the repository's `AGENTS.md`, `PLANS.md`, ADRs, OpenAPI contract, and production-readiness checklist are authoritative.

## Prompt 1 — Authentication and onboarding

Read `AGENTS.md`, `PLANS.md`, the product brief, ADRs, threat model, and current diff/history. Plan and implement the authentication/onboarding milestone only. Replace the bootstrap test identity on real app paths with the chosen OIDC Authorization Code + PKCE integration while retaining isolated deterministic test fixtures. Implement adult-age gate, consent versioning, timezone/units, goal, training experience, schedule, equipment, diet preference, and safety-boundary questions. Add resume-safe onboarding, ownership/RBAC tests, OpenAPI/client updates, mobile screens and states, analytics events that contain no sensitive values, migrations, docs, and runbooks. Do not implement workout planning yet. Verify issuer/audience/signature/expiry handling and prove the local bypass cannot activate in staging/production. Run all relevant checks, review the diff, update `PLANS.md` and production-readiness evidence, and end with the exact next prompt.

## Prompt 2 — Exercise catalog, programs, and workout logging

Read the repository guidance and completed milestone evidence. Plan and implement the exercise catalog, versioned starter programs, scheduling, and offline-first workout logging vertical slice. Include exercise/equipment metadata, content licensing/source fields, substitutions, warm-up/working/drop/failure set types, previous values, timers, notes, RPE/RIR, undo, abandon/resume/complete state transitions, idempotent offline replay, optimistic concurrency, and conflict UI. Seed only synthetic or clearly reusable content. Add migrations, indexes justified by query plans, OpenAPI/client generation, mobile UX states, admin-safe seed tooling, unit/integration/component/E2E tests, and airplane-mode/process-death scenarios. Do not implement automatic load progression yet. Run checks, review privacy/security/accessibility, update plans/readiness, and recommend the next prompt.

## Prompt 3 — Deterministic progression engine

Implement the versioned deterministic progression engine described in the product brief. First write an ADR and executable acceptance examples for double progression, load increments, RIR targets, conservative maintain/regress rules, deload suggestions, substitutions, unit conversion, manual overrides, and pain flags. Make decisions return structured reason codes and user-readable parameterized explanations. Use property/fuzz tests for invariants: recommendations stay within safe configured bounds, conversions do not drift across round trips, the same history/config produces the same result, and LLM output cannot alter prescriptions. Integrate recommendations into the next-workout API and mobile review/edit flow. Backfill/version policy references safely. Add observability without sensitive payloads, run performance tests on a realistic synthetic history, update docs and readiness, and stop after this milestone.

## Prompt 4 — Progress, readiness, and retention loop

Implement progress dashboards, personal records, body metrics, daily readiness check-ins, transparent weekly summaries, flexible rescheduling, and plateau indicators. Use statistically honest labels and avoid implying causation or medical diagnosis. Add notification preferences and a provider adapter, but use a local fake and do not send real notifications. Instrument activation and retention events with a documented analytics schema that excludes raw health/nutrition values. Add timezone-safe scheduling, quiet hours, opt-out, deletion propagation, APIs, screens, accessibility states, tests, and operational metrics. Validate charts with empty/sparse/outlier data and metric/imperial switching. Update plans/readiness and recommend the next prompt.

## Prompt 5 — Nutrition MVP

Implement the bounded nutrition MVP: versioned energy/protein calculations, user-editable targets, manual foods, source and verification status, serving/unit conversion, Indian household measures, meal templates, and daily logs. Every recommendation must expose inputs, formula/version, uncertainty, and safety boundaries. Add explicit handling that declines medical-condition, pregnancy, minor, eating-disorder, drug, supplement, and crash-diet guidance. Never label user-entered data verified. Add data provenance, APIs/client, offline-safe logging, mobile screens, admin review flow for catalog data, migrations/indexes, deterministic unit/property tests, authorization/deletion/export tests, and accessibility. Do not add photo recognition or generative meal planning. Run all checks and review for unsafe output before completion.

## Prompt 6 — Subscriptions and entitlements

Implement subscriptions behind provider adapters. Before coding, verify current Apple, Google Play, Razorpay, and any selected entitlement-provider documentation and record dated links/decisions. Keep server-side entitlement as the authorization source. Build signed raw-body webhook verification, replay protection, idempotency, out-of-order transitions, grace period, cancellation at period end, refund/revocation, restore purchases, reconciliation jobs, and support-visible audit history. Use sandbox/test providers only and never require real payment credentials in CI. Add Pro feature gates on API and clients, paywall accessibility, transparent renewal/cancellation copy, analytics, fixtures for failure cases, and a billing runbook. Perform a threat-model review and update readiness; do not enable production payments.

## Prompt 7 — Admin, support, privacy operations, and observability

Implement the minimal Next.js operations console and backend controls for content editing, user lookup with field minimization, entitlement reconciliation, audit review, deletion/export job status, feature flags, and safety-report triage. Enforce role/attribute checks server-side; support-readonly cannot mutate. Require reasons and audit events for sensitive views/actions. Complete self-service export/deletion, retention jobs, outbox/dead-letter operations, dashboards/alerts as code where practical, tracing/metrics/log redaction, backup/restore runbooks, and incident response. Test broken object authorization and admin abuse paths. Do not expose raw sensitive payloads in analytics, logs, traces, or support UI.

## Prompt 8 — Dedicated security hardening and penetration-test preparation

Treat this milestone as an adversarial security assessment, not a feature sprint. Read the threat model, data inventory/flows, vendor inventory, ASVS/MASVS mappings, NIST SSDF practices, production architecture, and every changed trust boundary. Update the threat model using concrete abuse cases for account takeover, broken object authorization, admin abuse, bulk export/scraping, mobile token/database theft, offline replay, webhook forgery, payment/entitlement manipulation, SSRF/file upload, log/analytics leakage, KMS/secret compromise, dependency/CI compromise, backup exposure, deletion failure, and unsafe AI data disclosure. Complete applicable OWASP ASVS Level 2 plus risk-selected Level 3 evidence and OWASP MASVS/MASTG tests. Run SAST, secrets/history scan, dependency/container/IaC scans, SBOM/provenance checks, staging DAST, authorization matrix tests, fuzz/property tests, mobile static/dynamic analysis, and manual business-logic review. Test key/credential rotation, staff MFA and access revocation, alerts, incident escalation, backup restore, export/deletion, environment isolation, and rollback. Fix all critical/high findings and document medium exceptions with owner, compensating control, and expiry. Produce a sanitized package and rules of engagement for an independent qualified penetration tester; do not fabricate an external assessment or attack production. End with a security go/no-go report and the exact next prompt.

## Prompt 9 — Beta hardening and release candidate

Treat this as a release-candidate audit, not a feature sprint. Require Prompt 8 security approval evidence and independent penetration-test results; do not waive or invent them. Read every production-readiness gate and gather evidence. Run the complete format/lint/type/unit/integration/contract/E2E/security/dependency/build suite; load-test the documented beta profile; test offline sync, backup restore, migration rollback/forward recovery, account export/deletion, billing replay/reconciliation, key rotation, rate limiting, degraded dependencies, and mobile release builds. Review store health declarations, privacy disclosures, permissions, subscription copy, content licenses, and legal-review blockers against current primary sources. Fix issues within scope, document unresolved blockers with owner/severity, create staging deployment instructions and a rollback plan, and produce a go/no-go report. Do not deploy or activate real billing without explicit approval and credentials.

## Reusable continuation prompt

Use this for a narrowly defined feature after the main milestones:

> Read `AGENTS.md`, `PLANS.md`, relevant ADRs, the OpenAPI contract, and production-readiness gates. Inspect current code and preserve existing work. For this request, state assumptions and acceptance criteria, create a small plan, implement the smallest complete vertical slice, update contract/client/migrations/docs together, add tests for success/failure/authorization/offline or retry behavior as applicable, run relevant checks, review the diff for security/privacy/accessibility/regressions, update `PLANS.md`, and report exact results and remaining risks. Request: **[INSERT ONE COHERENT OUTCOME]**.

## Prompts to avoid

Avoid: “Build the entire app, make it production ready, and deploy it.” That encourages shallow breadth and unverifiable claims.

Prefer: “Implement milestone 2 exactly as specified, run the listed checks, update readiness evidence, and stop.”

## Founder decisions still required before public beta

These do not block local bootstrap, but they must be resolved with evidence:

- Final brand/name, trademark/domain check, and visual identity
- Exact first persona: college gym beginners, general beginners, or intermediate lifters
- OIDC provider and email/SMS costs
- Licensed exercise media/content source
- Verified Indian food-data source and update rights
- Mobile purchase/entitlement provider and web payment provider contracts
- Entity, tax/GST, terms, privacy notice, grievance/contact process, and qualified Indian legal review
- Human fitness/nutrition expert review of starter programs and formulas
- Analytics/error/notification vendors and data-processing terms
- Independent security testing provider, penetration-test scope, retest budget, vulnerability disclosure contact, and incident-response owner
- Beta cohort, support capacity, pricing experiment, and explicit go/no-go metrics
