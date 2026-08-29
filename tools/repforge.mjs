import { randomBytes } from "node:crypto";
import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, posix, relative, resolve } from "node:path";
import { spawn, spawnSync } from "node:child_process";
import process from "node:process";
import { fileURLToPath, pathToFileURL } from "node:url";

export function workspaceRootFromModuleURL(moduleURL) {
  return resolve(dirname(fileURLToPath(moduleURL)), "..");
}

export const WORKSPACE_ROOT = workspaceRootFromModuleURL(import.meta.url);
const root = WORKSPACE_ROOT;
const backend = join(root, "backend");
const envPath = join(root, ".env.local");
const pnpm = process.platform === "win32" ? "pnpm.cmd" : "pnpm";
const pnpmScript = process.env.npm_execpath;
const go = process.platform === "win32" ? "go.exe" : "go";
const goCache = join(root, ".cache", "go-build");
const expoHome = join(root, ".cache", "expo");
const pureGoEnv = { ...process.env, CGO_ENABLED: "0", GOCACHE: goCache };
const bootstrapVersion = "0.1.0-bootstrap";
export const TRIVY_IMAGE =
  "aquasec/trivy:0.73.0@sha256:7cced7cae583819fc7806d4cbc0dbbc7cad18b99f7d3e235192e6da8c091045c";
export const GITLEAKS_IMAGE =
  "zricethezav/gitleaks:v8.28.0@sha256:cdbb7c955abce02001a9f6c9f602fb195b7fadc1e812065883f695d1eeaba854";
export const IMAGE_REFERENCES = Object.freeze({
  postgres:
    "postgres:18.6-alpine@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2",
  redis:
    "redis:8.2.9-alpine@sha256:30abb90e62f14b737010746def3ba99cc79fe19dcdb3d37b41f21fc62e7da19d",
  garage:
    "dxflrs/garage:v2.3.0@sha256:866bd13ed2038ba7e7190e840482bc27234c4afaf77be8cfa439ae088c1e4690",
});
export const PNPM_AUDIT_ARGS = ["audit", "--audit-level", "high"];

function workspaceToolEnv(base = process.env) {
  return {
    ...base,
    EXPO_HOME: expoHome,
    EXPO_NO_TELEMETRY: "1",
    NEXT_TELEMETRY_DISABLED: "1",
  };
}

function localGoEnv() {
  return { ...localEnv(), CGO_ENABLED: "0", GOCACHE: goCache };
}

function validateLocalPlatform() {
  const env = localGoEnv();
  validateDevelopmentCredentialOrigin(env);
  execute(go, ["run", "./cmd/configcheck"], { cwd: backend, env });
}

function dockerCommand() {
  if (process.env.DOCKER_PATH && existsSync(process.env.DOCKER_PATH))
    return process.env.DOCKER_PATH;
  const local = process.env.LOCALAPPDATA;
  if (process.platform === "win32" && local) {
    const candidate = join(local, "Programs", "DockerDesktop", "resources", "bin", "docker.exe");
    if (existsSync(candidate)) return candidate;
  }
  return process.platform === "win32" ? "docker.exe" : "docker";
}

function execute(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd ?? root,
    env: options.env ?? process.env,
    encoding: "utf8",
    stdio: options.capture ? "pipe" : "inherit",
    shell: options.shell ?? false,
  });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    if (options.capture) {
      process.stderr.write(result.stdout ?? "");
      process.stderr.write(result.stderr ?? "");
    }
    throw new Error(`${command} ${args.join(" ")} exited with ${result.status}`);
  }
  return (result.stdout ?? "").trim();
}

function executePnpm(args, options = {}) {
  const pnpmOptions = { ...options, env: options.env ?? workspaceToolEnv() };
  if (pnpmScript && existsSync(pnpmScript)) {
    return execute(process.execPath, [pnpmScript, ...args], pnpmOptions);
  }
  return execute(pnpm, args, { ...pnpmOptions, shell: process.platform === "win32" });
}

function tryVersion(label, command, args) {
  try {
    const value = execute(command, args, { capture: true });
    console.log(`${label}: ${value}`);
    return true;
  } catch (error) {
    console.error(`${label}: unavailable (${error.message})`);
    return false;
  }
}

function parseEnv(text) {
  const values = {};
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const separator = line.indexOf("=");
    if (separator < 1) continue;
    values[line.slice(0, separator)] = line.slice(separator + 1);
  }
  return values;
}

function literalLoopbackAuthority(raw, parsed) {
  const schemeSeparator = raw.indexOf("://");
  if (schemeSeparator < 0) return false;
  const remainder = raw.slice(schemeSeparator + 3);
  const authorityEnd = remainder.search(/[/?#]/);
  const authority = authorityEnd < 0 ? remainder : remainder.slice(0, authorityEnd);
  if (!authority || authority.includes("@")) return false;
  if (authority.startsWith("[")) {
    const closingBracket = authority.indexOf("]");
    if (closingBracket < 0) return false;
    const suffix = authority.slice(closingBracket + 1);
    if (suffix && !/^:\d+$/.test(suffix)) return false;
    return parsed.hostname.toLowerCase() === "[::1]";
  }
  const colon = authority.lastIndexOf(":");
  const host = colon < 0 ? authority : authority.slice(0, colon);
  if (colon >= 0 && !/^\d+$/.test(authority.slice(colon + 1))) return false;
  const octets = host.split(".");
  return (
    octets.length === 4 &&
    octets.every(
      (octet) => /^\d{1,3}$/.test(octet) && Number(octet) <= 255 && String(Number(octet)) === octet,
    ) &&
    Number(octets[0]) === 127
  );
}

export function isLiteralLoopbackHTTPOrigin(raw) {
  if (typeof raw !== "string" || !raw || raw.trim() !== raw) return false;
  try {
    const parsed = new URL(raw);
    return (
      parsed.protocol === "http:" &&
      parsed.username === "" &&
      parsed.password === "" &&
      parsed.pathname === "/" &&
      parsed.search === "" &&
      parsed.hash === "" &&
      literalLoopbackAuthority(raw, parsed)
    );
  } catch {
    return false;
  }
}

export function validateDevelopmentCredentialOrigin(environment) {
  const authMode = environment.EXPO_PUBLIC_AUTH_MODE;
  const token = environment.EXPO_PUBLIC_DEV_AUTH_TOKEN;
  if (authMode !== "dev" && !token) return;
  const apiOrigin = environment.EXPO_PUBLIC_API_URL ?? "http://127.0.0.1:8080";
  if (!isLiteralLoopbackHTTPOrigin(apiOrigin)) {
    throw new Error(
      "EXPO_PUBLIC_API_URL must be an HTTP origin with a literal loopback IP while mobile development credentials are active",
    );
  }
}

function localEnv() {
  if (!existsSync(envPath)) {
    throw new Error(".env.local is missing; run `pnpm run setup` first");
  }
  return { ...process.env, ...parseEnv(readFileSync(envPath, "utf8")) };
}

function createLocalEnv() {
  if (existsSync(envPath)) return;
  const postgresPassword = randomBytes(24).toString("base64url");
  const redisPassword = randomBytes(24).toString("base64url");
  const devToken = `dev_${randomBytes(24).toString("base64url")}`;
  const garageAccessKey = `GK${randomBytes(16).toString("hex").toUpperCase()}`;
  const garageSecretKey = randomBytes(32).toString("hex");
  const value = readFileSync(join(root, ".env.example"), "utf8")
    .replaceAll("__GENERATED_POSTGRES_PASSWORD__", postgresPassword)
    .replaceAll("__GENERATED_REDIS_PASSWORD__", redisPassword)
    .replaceAll("__GENERATED_DEV_AUTH_TOKEN__", devToken)
    .replaceAll("__GENERATED_GARAGE_ACCESS_KEY__", garageAccessKey)
    .replaceAll("__GENERATED_GARAGE_SECRET_KEY__", garageSecretKey);
  writeFileSync(envPath, value, { encoding: "utf8", mode: 0o600, flag: "wx" });
  console.log("Created .env.local with synthetic local-only credentials.");
}

function compose(args, env = localEnv()) {
  return execute(
    dockerCommand(),
    ["compose", "--env-file", envPath, "-f", join(root, "compose.yaml"), ...args],
    { env },
  );
}

export function assertImmutableImageReferences(references) {
  if (references.length === 0) throw new Error("no Compose image references were found");
  for (const reference of references) {
    if (!/^[^\s@]+@sha256:[a-f0-9]{64}$/.test(reference)) {
      throw new Error(`image reference is not immutable: ${reference}`);
    }
  }
}

export function composeImageReferences(text) {
  return [...text.matchAll(/^\s+image:\s+([^\s#]+)\s*$/gm)].map((match) => match[1]);
}

function configuredImageReferences() {
  const references = composeImageReferences(readFileSync(join(root, "compose.yaml"), "utf8"));
  assertImmutableImageReferences(references);
  return references;
}

function verifyRegistryManifest(reference) {
  const digest = reference.slice(reference.indexOf("@") + 1);
  const metadata = execute(dockerCommand(), ["buildx", "imagetools", "inspect", reference], {
    capture: true,
  });
  if (!metadata.includes(`Digest:    ${digest}`) && !metadata.includes(`Digest: ${digest}`)) {
    throw new Error(`registry manifest did not resolve to ${digest}: ${reference}`);
  }
  console.log(`Verified registry manifest: ${reference}`);
}

const exactImageScanPolicies = Object.freeze({
  [IMAGE_REFERENCES.postgres]: Object.freeze({
    waiverFile: "local/trivy/postgres-ignore.yaml",
    allowMissingInventory: false,
    localOnly: false,
  }),
  [IMAGE_REFERENCES.redis]: Object.freeze({
    waiverFile: "local/trivy/redis-ignore.yaml",
    allowMissingInventory: false,
    localOnly: false,
  }),
  [IMAGE_REFERENCES.garage]: Object.freeze({
    waiverFile: undefined,
    allowMissingInventory: true,
    localOnly: true,
  }),
});

export function imageScanPolicy(reference) {
  return (
    exactImageScanPolicies[reference] ?? {
      waiverFile: undefined,
      allowMissingInventory: false,
      localOnly: false,
    }
  );
}

export function summarizeTrivyInventory(report) {
  const results = Array.isArray(report?.Results) ? report.Results : [];
  const osResults = results.filter(
    (result) => result?.Class === "os-pkgs" && Array.isArray(result.Packages),
  );
  const packageCount = osResults.reduce((total, result) => total + result.Packages.length, 0);
  return { present: packageCount > 0, packageCount };
}

export function summarizeTrivyVulnerabilities(report) {
  const results = Array.isArray(report?.Results) ? report.Results : [];
  const vulnerabilities = results.flatMap((result) =>
    Array.isArray(result?.Vulnerabilities) ? result.Vulnerabilities : [],
  );
  return {
    count: vulnerabilities.length,
    ids: [...new Set(vulnerabilities.map((item) => item.VulnerabilityID).filter(Boolean))].sort(),
  };
}

export function productionImageBlockers(results) {
  const blockers = [];
  for (const result of results) {
    if (result.localOnly) blockers.push(`${result.reference}: local-only image`);
    if (!result.inventoryPresent) blockers.push(`${result.reference}: no vulnerability inventory`);
    if (result.vulnerabilityCount > 0) {
      blockers.push(
        `${result.reference}: ${result.vulnerabilityCount} unsuppressed high/critical vulnerability findings (${result.vulnerabilityIDs.join(", ")})`,
      );
    }
    if (!result.sbomVerified) blockers.push(`${result.reference}: release SBOM not verified`);
    if (!result.provenanceVerified)
      blockers.push(`${result.reference}: publisher provenance not verified`);
  }
  return blockers;
}

export function assertProductionImagePromotionAllowed(results) {
  const blockers = productionImageBlockers(results);
  if (blockers.length > 0) {
    throw new Error(`production image promotion blocked:\n- ${blockers.join("\n- ")}`);
  }
}

export function trivyScanArguments(reference, { production = false, rootPath = root } = {}) {
  const policy = imageScanPolicy(reference);
  const dockerArguments = ["run", "--rm", "-v", "repforge-trivy-cache:/root/.cache"];
  const waiverFile = production ? undefined : policy.waiverFile;
  if (waiverFile) {
    const waiverPath = join(rootPath, ...waiverFile.split("/"));
    dockerArguments.push("-v", `${waiverPath}:/etc/trivy/ignore.yaml:ro`);
  }
  dockerArguments.push(
    TRIVY_IMAGE,
    "image",
    "--skip-version-check",
    "--image-src",
    "remote",
    "--exit-code",
    production ? "0" : "1",
    "--severity",
    "HIGH,CRITICAL",
    "--scanners",
    "vuln",
    "--pkg-types",
    "os",
    "--format",
    "json",
  );
  if (waiverFile) {
    dockerArguments.push("--ignorefile", "/etc/trivy/ignore.yaml", "--show-suppressed");
  }
  dockerArguments.push(reference);
  return dockerArguments;
}

function scanImage(reference, { production = false } = {}) {
  const policy = imageScanPolicy(reference);
  const dockerArguments = trivyScanArguments(reference, { production });

  const rawReport = execute(dockerCommand(), dockerArguments, { capture: true });
  let report;
  try {
    report = JSON.parse(rawReport);
  } catch (error) {
    throw new Error(`Trivy returned invalid JSON for ${reference}: ${error.message}`);
  }
  const inventory = summarizeTrivyInventory(report);
  const vulnerabilities = production
    ? summarizeTrivyVulnerabilities(report)
    : { count: 0, ids: [] };
  if (!inventory.present && !policy.allowMissingInventory) {
    throw new Error(`Trivy found no OS package inventory for ${reference}`);
  }
  if (inventory.present) {
    console.log(`Trivy inventoried ${inventory.packageCount} OS packages: ${reference}`);
  } else {
    console.warn(
      `MISSING COVERAGE: ${reference} has no OS package inventory and is permitted for local use only.`,
    );
  }
  if (!production && policy.waiverFile) {
    console.log(`Applied exact-image waiver policy ${policy.waiverFile}: ${reference}`);
  }
  if (production && vulnerabilities.count > 0) {
    console.warn(
      `PRODUCTION BLOCKER: ${reference} has ${vulnerabilities.count} unsuppressed high/critical findings (${vulnerabilities.ids.join(", ")}).`,
    );
  }
  return {
    reference,
    inventoryPresent: inventory.present,
    packageCount: inventory.packageCount,
    localOnly: policy.localOnly,
    vulnerabilityCount: vulnerabilities.count,
    vulnerabilityIDs: vulnerabilities.ids,
    sbomVerified: false,
    provenanceVerified: false,
  };
}

function verifyImages({ production = false } = {}) {
  const references = configuredImageReferences();
  assertImmutableImageReferences([TRIVY_IMAGE]);
  execute(dockerCommand(), ["pull", TRIVY_IMAGE]);
  verifyRegistryManifest(TRIVY_IMAGE);
  const results = [];
  for (const reference of references) {
    verifyRegistryManifest(reference);
    results.push(scanImage(reference, { production }));
  }
  if (production) assertProductionImagePromotionAllowed(results);
  console.log("Immutable image manifests verified; local image coverage status reported above.");
  return results;
}

function inspectGit(repositoryRoot, argumentsList) {
  return spawnSync("git", argumentsList, {
    cwd: repositoryRoot,
    encoding: "utf8",
    stdio: "pipe",
    shell: false,
  });
}

function gitInspectionError(label, result) {
  if (result.error) return result.error;
  const detail = (result.stderr ?? "").trim() || `exit status ${String(result.status)}`;
  return new Error(`Git repository inspection failed during ${label}: ${detail}`);
}

export function inspectGitRepository(repositoryRoot) {
  const workTree = inspectGit(repositoryRoot, ["rev-parse", "--is-inside-work-tree"]);
  if (workTree.status !== 0 || workTree.stdout.trim() !== "true") {
    throw gitInspectionError("work-tree verification", workTree);
  }

  const head = inspectGit(repositoryRoot, ["rev-parse", "--verify", "HEAD^{commit}"]);
  if (head.error) throw head.error;
  if (head.status === 0) {
    const headCommit = head.stdout.trim();
    if (!/^[a-f0-9]{40}$/.test(headCommit)) {
      throw new Error("Git repository inspection returned an invalid HEAD commit");
    }
    return { hasHistory: true, headCommit };
  }

  const symbolicHead = inspectGit(repositoryRoot, ["symbolic-ref", "--quiet", "HEAD"]);
  if (
    symbolicHead.status !== 0 ||
    !/^refs\/heads\/[A-Za-z0-9._\/-]+$/.test(symbolicHead.stdout.trim())
  ) {
    throw gitInspectionError("unborn HEAD verification", symbolicHead);
  }
  const anyCommit = inspectGit(repositoryRoot, ["rev-list", "--all", "--max-count=1"]);
  if (anyCommit.status !== 0) throw gitInspectionError("reachable-history verification", anyCommit);
  const reachableCommit = anyCommit.stdout.trim();
  if (reachableCommit) {
    if (!/^[a-f0-9]{40}$/.test(reachableCommit)) {
      throw new Error("Git repository inspection returned an invalid reachable commit");
    }
    return { hasHistory: true, headCommit: undefined };
  }
  return { hasHistory: false, headCommit: undefined };
}

function gitCommit() {
  return inspectGitRepository(root).headCommit;
}

export function gitleaksCanary() {
  return ["ghp", "_", randomBytes(18).toString("hex")].join("");
}

const syntheticLocalSecretRules = Object.freeze([
  ["DEV_AUTH_TOKEN", /^dev_[A-Za-z0-9_-]{32}$/],
  ["POSTGRES_PASSWORD", /^[A-Za-z0-9_-]{32}$/],
  ["REDIS_PASSWORD", /^[A-Za-z0-9_-]{32}$/],
  ["GARAGE_ACCESS_KEY", /^GK[A-F0-9]{32}$/],
  ["GARAGE_SECRET_KEY", /^[a-f0-9]{64}$/],
]);

export function syntheticFixtureAllowlist(environmentText = "") {
  const allowedLines = new Map();
  if (!environmentText.trim()) return allowedLines;
  const values = parseEnv(environmentText);
  const exactSecrets = new Map();
  for (const [name, pattern] of syntheticLocalSecretRules) {
    const value = values[name];
    if (value === undefined) continue;
    if (!pattern.test(value)) {
      throw new Error(`${name} is not a recognized generated local fixture; refusing allowlist`);
    }
    exactSecrets.set(name, value);
  }
  if (exactSecrets.has("DEV_AUTH_TOKEN")) {
    exactSecrets.set("EXPO_PUBLIC_DEV_AUTH_TOKEN", exactSecrets.get("DEV_AUTH_TOKEN"));
  }
  for (const [index, raw] of environmentText.split(/\r?\n/).entries()) {
    const line = raw.trim();
    const separator = line.indexOf("=");
    if (separator < 1) continue;
    const name = line.slice(0, separator);
    const value = line.slice(separator + 1);
    if (exactSecrets.get(name) === value) {
      allowedLines.set(index + 1, name);
    }
  }
  return allowedLines;
}

const gitleaksCommonArguments = ["--no-banner", "--no-color", "--redact=100"];

function gitleaksEnvironment(config) {
  return { ...process.env, GITLEAKS_CONFIG_TOML: config };
}

export function gitleaksDirectoryDockerArguments(sourceMount, { reportMount } = {}) {
  const argumentsList = ["run", "--rm", "-v", sourceMount];
  if (reportMount) {
    argumentsList.push("-v", reportMount);
  }
  argumentsList.push(
    "-e",
    "GITLEAKS_CONFIG_TOML",
    GITLEAKS_IMAGE,
    "dir",
    ...gitleaksCommonArguments,
  );
  if (reportMount) {
    argumentsList.push(
      "--exit-code",
      "0",
      "--report-format",
      "json",
      "--report-path",
      "/report/findings.json",
    );
  }
  argumentsList.push("/repo");
  return argumentsList;
}

export function gitleaksHistoryDockerArguments(sourceMount) {
  return [
    "run",
    "--rm",
    "-v",
    sourceMount,
    "-e",
    "GITLEAKS_CONFIG_TOML",
    "-e",
    "GIT_CONFIG_COUNT=1",
    "-e",
    "GIT_CONFIG_KEY_0=safe.directory",
    "-e",
    "GIT_CONFIG_VALUE_0=/repo",
    GITLEAKS_IMAGE,
    "git",
    ...gitleaksCommonArguments,
    "/repo",
  ];
}

function runGitleaks(argumentsList, config, { input } = {}) {
  return spawnSync(dockerCommand(), argumentsList, {
    cwd: root,
    env: gitleaksEnvironment(config),
    encoding: "utf8",
    input,
    stdio: [input === undefined ? "ignore" : "pipe", "pipe", "pipe"],
    shell: false,
  });
}

function requireGitleaksStatus(result, expected, label) {
  if (result.error) throw result.error;
  if (result.status !== expected) {
    throw new Error(`${label} returned ${result.status}; expected ${expected}`);
  }
}

function gitleaksDirectoryFindings(sourceMount, config) {
  const reportRoot = mkdtempSync(join(tmpdir(), "repforge-gitleaks-report-"));
  try {
    const result = runGitleaks(
      gitleaksDirectoryDockerArguments(sourceMount, {
        reportMount: `${reportRoot}:/report`,
      }),
      config,
    );
    requireGitleaksStatus(result, 0, "Gitleaks directory report");
    const reportPath = join(reportRoot, "findings.json");
    if (!existsSync(reportPath)) throw new Error("Gitleaks did not create its JSON report");
    const findings = JSON.parse(readFileSync(reportPath, "utf8") || "[]");
    if (!Array.isArray(findings)) throw new Error("report is not an array");
    return findings;
  } catch (error) {
    throw new Error(`Gitleaks returned invalid JSON: ${error.message}`);
  } finally {
    rmSync(reportRoot, { recursive: true, force: true });
  }
}

export function unexpectedGitleaksFindings(findings, allowedFixtureLines = new Map()) {
  return findings.filter((finding) => {
    const file = posix.normalize(
      String(finding.File ?? "")
        .replaceAll("\\", "/")
        .replace(/\/{2,}/g, "/"),
    );
    const isLocalFixture = file === ".env.local" || file === "/repo/.env.local";
    return !(
      isLocalFixture &&
      finding.RuleID === "generic-api-key" &&
      allowedFixtureLines.has(Number(finding.StartLine))
    );
  });
}

function safeFindingSummary(findings) {
  return findings
    .map(
      (finding) =>
        `${String(finding.RuleID ?? "unknown-rule")} ${String(finding.File ?? "unknown-file")}:${Number(finding.StartLine) || 0}`,
    )
    .join(", ");
}

function requireNoUnexpectedDirectoryFindings(sourceMount, config, allowedFixtureLines, label) {
  const findings = gitleaksDirectoryFindings(sourceMount, config);
  const unexpected = unexpectedGitleaksFindings(findings, allowedFixtureLines);
  if (unexpected.length > 0) {
    throw new Error(
      `${label} detected ${unexpected.length} finding(s): ${safeFindingSummary(unexpected)}`,
    );
  }
  return findings.length;
}

function verifySecretScannerCanary(config) {
  const result = runGitleaks(
    [
      "run",
      "--rm",
      "-i",
      "-e",
      "GITLEAKS_CONFIG_TOML",
      GITLEAKS_IMAGE,
      "stdin",
      ...gitleaksCommonArguments,
    ],
    config,
    { input: `GITHUB_TOKEN=${gitleaksCanary()}\n` },
  );
  requireGitleaksStatus(result, 1, "Gitleaks stdin canary");
  console.log("Pinned Gitleaks canary rejected a generated GitHub token pattern.");
}

function verifyNarrowFixtureAllowlist(baseConfig) {
  const temporaryRoot = mkdtempSync(join(tmpdir(), "repforge-gitleaks-fixture-"));
  try {
    const syntheticPassword = randomBytes(24).toString("base64url");
    const fixtureText = `POSTGRES_PASSWORD=${syntheticPassword}\n`;
    const allowedFixtureLines = syntheticFixtureAllowlist(fixtureText);
    const fixturePath = join(temporaryRoot, ".env.local");
    writeFileSync(fixturePath, fixtureText, { encoding: "utf8", mode: 0o600 });
    const sourceMount = `${temporaryRoot}:/repo:ro`;
    requireNoUnexpectedDirectoryFindings(
      sourceMount,
      baseConfig,
      allowedFixtureLines,
      "Gitleaks exact-fixture allowlist canary",
    );
    writeFileSync(fixturePath, `${fixtureText}UNRELATED_TOKEN=${gitleaksCanary()}\n`, {
      encoding: "utf8",
      mode: 0o600,
    });
    const findings = gitleaksDirectoryFindings(sourceMount, baseConfig);
    const unexpected = unexpectedGitleaksFindings(findings, allowedFixtureLines);
    if (unexpected.length === 0) {
      throw new Error(
        `Gitleaks unrelated-credential canary was not detected; report contained ${findings.length} finding(s): ${safeFindingSummary(findings)}`,
      );
    }
    console.log("Gitleaks detected an unrelated credential beside an exact allowed fixture value.");

    writeFileSync(fixturePath, fixtureText, { encoding: "utf8", mode: 0o600 });
    const nestedDirectory = join(temporaryRoot, "apps", "web");
    mkdirSync(nestedDirectory, { recursive: true });
    writeFileSync(join(nestedDirectory, ".env.local"), `GITHUB_TOKEN=${gitleaksCanary()}\n`, {
      encoding: "utf8",
      mode: 0o600,
    });
    const nestedFindings = gitleaksDirectoryFindings(sourceMount, baseConfig);
    const unexpectedNested = unexpectedGitleaksFindings(nestedFindings, allowedFixtureLines);
    if (!unexpectedNested.some((finding) => String(finding.File ?? "").includes("apps/web"))) {
      throw new Error(
        `Gitleaks nested .env.local canary was not detected: ${safeFindingSummary(nestedFindings)}`,
      );
    }
    console.log("Gitleaks detected an unrelated credential in a nested .env.local file.");
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
}

function verifyGitHistoryScanner(config) {
  const temporaryRoot = mkdtempSync(join(tmpdir(), "repforge-gitleaks-history-"));
  try {
    execute("git", ["init", "--quiet"], { cwd: temporaryRoot, capture: true });
    const fixturePath = join(temporaryRoot, "credential.txt");
    writeFileSync(fixturePath, `GITHUB_TOKEN=${gitleaksCanary()}\n`, {
      encoding: "utf8",
      mode: 0o600,
    });
    execute("git", ["add", "credential.txt"], { cwd: temporaryRoot, capture: true });
    execute(
      "git",
      [
        "-c",
        "user.name=RepForge Security Test",
        "-c",
        "user.email=security-test@invalid.example",
        "-c",
        "commit.gpgsign=false",
        "commit",
        "--quiet",
        "-m",
        "add synthetic credential fixture",
      ],
      { cwd: temporaryRoot, capture: true },
    );
    writeFileSync(fixturePath, "credential removed from current tree\n", "utf8");
    execute("git", ["add", "credential.txt"], { cwd: temporaryRoot, capture: true });
    execute(
      "git",
      [
        "-c",
        "user.name=RepForge Security Test",
        "-c",
        "user.email=security-test@invalid.example",
        "-c",
        "commit.gpgsign=false",
        "commit",
        "--quiet",
        "-m",
        "remove synthetic credential fixture",
      ],
      { cwd: temporaryRoot, capture: true },
    );
    const sourceMount = `${temporaryRoot}:/repo:ro`;
    requireGitleaksStatus(
      runGitleaks(gitleaksHistoryDockerArguments(sourceMount), config),
      1,
      "Gitleaks Git-history canary",
    );
    console.log("Gitleaks detected a credential retained only in disposable Git history.");
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
}

function scanSecrets() {
  assertImmutableImageReferences([GITLEAKS_IMAGE]);
  execute(dockerCommand(), ["pull", GITLEAKS_IMAGE]);
  verifyRegistryManifest(GITLEAKS_IMAGE);
  const baseConfig = readFileSync(join(root, ".gitleaks.toml"), "utf8");
  const environmentText = existsSync(envPath) ? readFileSync(envPath, "utf8") : "";
  const allowedFixtureLines = syntheticFixtureAllowlist(environmentText);
  verifySecretScannerCanary(baseConfig);
  verifyNarrowFixtureAllowlist(baseConfig);
  verifyGitHistoryScanner(baseConfig);
  const sourceMount = `${root}:/repo:ro`;
  const findingCount = requireNoUnexpectedDirectoryFindings(
    sourceMount,
    baseConfig,
    allowedFixtureLines,
    "Gitleaks working-tree scan",
  );
  console.log(
    `Gitleaks working-tree scan passed with ${findingCount} exact synthetic fixture finding(s) reviewed.`,
  );
  const repositoryState = inspectGitRepository(root);
  if (repositoryState.hasHistory) {
    execute(dockerCommand(), gitleaksHistoryDockerArguments(sourceMount), {
      env: gitleaksEnvironment(baseConfig),
    });
    console.log("Gitleaks project Git-history scan passed.");
  } else {
    console.warn("Project Git history scan skipped only after verifying an unborn repository.");
  }
}

export function resolveBuildMetadata({ env = process.env, commit, now = new Date() } = {}) {
  const metadata = {
    version: env.REPFORGE_BUILD_VERSION ?? bootstrapVersion,
    commit: env.REPFORGE_BUILD_COMMIT ?? env.GITHUB_SHA ?? commit ?? "uncommitted",
    builtAt: env.REPFORGE_BUILD_TIME ?? now.toISOString(),
  };
  if (!/^[0-9A-Za-z][0-9A-Za-z._+-]*$/.test(metadata.version)) {
    throw new Error("REPFORGE_BUILD_VERSION must be a non-empty linker-safe version");
  }
  if (metadata.commit !== "uncommitted" && !/^[a-f0-9]{40}$/.test(metadata.commit)) {
    throw new Error("build commit must be a 40-character lowercase Git SHA or uncommitted");
  }
  if (!metadata.builtAt.endsWith("Z") || Number.isNaN(Date.parse(metadata.builtAt))) {
    throw new Error("REPFORGE_BUILD_TIME must be a UTC RFC 3339 timestamp");
  }
  return metadata;
}

function buildMetadata() {
  return resolveBuildMetadata({ commit: gitCommit() });
}

function goLinkerFlags(metadata) {
  return [
    `-X=main.version=${metadata.version}`,
    `-X=main.commit=${metadata.commit}`,
    `-X=main.builtAt=${metadata.builtAt}`,
  ].join(" ");
}

function buildGoCommands(extraArgs = []) {
  const metadata = buildMetadata();
  console.log(
    `API build metadata: version=${metadata.version} commit=${metadata.commit} builtAt=${metadata.builtAt}`,
  );
  execute(
    go,
    ["build", "-buildvcs=false", "-trimpath", "-ldflags", goLinkerFlags(metadata), ...extraArgs],
    { cwd: backend, env: pureGoEnv },
  );
  return metadata;
}

function filesUnder(path) {
  if (!existsSync(path)) return [];
  const result = [];
  for (const entry of readdirSync(path)) {
    const child = join(path, entry);
    if (statSync(child).isDirectory()) result.push(...filesUnder(child));
    else result.push(child);
  }
  return result.sort();
}

function snapshot(paths) {
  const state = new Map();
  for (const path of paths) {
    for (const file of filesUnder(path)) state.set(relative(root, file), readFileSync(file));
  }
  return state;
}

function sameSnapshot(before, after) {
  if (before.size !== after.size) return false;
  for (const [name, value] of before) {
    const candidate = after.get(name);
    if (!candidate || !value.equals(candidate)) return false;
  }
  return true;
}

function generate() {
  execute(go, ["run", "github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1", "generate"], {
    cwd: backend,
    env: pureGoEnv,
  });
  executePnpm(["exec", "redocly", "lint", "backend/openapi/openapi.yaml"]);
  executePnpm(["--filter", "@repforge/api-client", "run", "generate"]);
  executePnpm(["exec", "prettier", "--write", "packages/api-client/src/generated/schema.d.ts"]);
}

function generateCheck() {
  const paths = [
    join(backend, "internal", "users", "db"),
    join(root, "packages", "api-client", "src", "generated"),
  ];
  const before = snapshot(paths);
  generate();
  if (!sameSnapshot(before, snapshot(paths))) {
    throw new Error("generated files were stale; run `pnpm run generate`");
  }
}

function doctor() {
  console.log(`workspace: ${root}`);
  const checks = [
    tryVersion("git", "git", ["--version"]),
    tryVersion("go", go, ["version"]),
    tryVersion("node", process.execPath, ["--version"]),
    (() => {
      try {
        console.log(`pnpm: ${executePnpm(["--version"], { capture: true })}`);
        return true;
      } catch (error) {
        console.error(`pnpm: unavailable (${error.message})`);
        return false;
      }
    })(),
    tryVersion("docker", dockerCommand(), ["--version"]),
    tryVersion("compose", dockerCommand(), ["compose", "version"]),
    tryVersion("buildx", dockerCommand(), ["buildx", "version"]),
    tryVersion("docker daemon", dockerCommand(), ["info", "--format", "{{.ServerVersion}}"]),
  ];
  if (checks.some((value) => !value)) throw new Error("environment doctor failed");
}

function format(write) {
  if (write) {
    const goFiles = filesUnder(backend).filter((path) => path.endsWith(".go"));
    execute("gofmt", ["-w", ...goFiles], { cwd: backend });
    executePnpm(["exec", "prettier", "--write", ".", "--ignore-path", ".gitignore"]);
    return;
  }
  const unformatted = execute("gofmt", ["-l", "."], { cwd: backend, capture: true });
  if (unformatted) throw new Error(`gofmt required for:\n${unformatted}`);
  executePnpm(["exec", "prettier", "--check", ".", "--ignore-path", ".gitignore"]);
}

async function waitFor(url, timeoutMs) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // Process may still be starting.
    }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 250));
  }
  throw new Error(`timed out waiting for ${url}`);
}

async function smoke() {
  const env = localGoEnv();
  validateLocalPlatform();
  const metadata = buildGoCommands([
    "-o",
    join("bin", process.platform === "win32" ? "repforge-api.exe" : "repforge-api"),
    "./cmd/api",
  ]);
  const binary = join(
    backend,
    "bin",
    process.platform === "win32" ? "repforge-api.exe" : "repforge-api",
  );
  const child = spawn(binary, [], { cwd: backend, env, stdio: "inherit", shell: false });
  try {
    await waitFor("http://127.0.0.1:8080/healthz", 20000);
    for (const path of ["healthz", "readyz"]) {
      const response = await fetch(`http://127.0.0.1:8080/${path}`);
      if (!response.ok) throw new Error(`${path} returned ${response.status}`);
    }
    const versionResponse = await fetch("http://127.0.0.1:8080/version");
    if (!versionResponse.ok) throw new Error(`version returned ${versionResponse.status}`);
    const actualMetadata = await versionResponse.json();
    for (const [name, value] of Object.entries(metadata)) {
      if (actualMetadata[name] !== value) {
        throw new Error(`/version ${name}=${actualMetadata[name]}, want ${value}`);
      }
    }
    const headers = { Authorization: `Bearer ${env.DEV_AUTH_TOKEN}` };
    const meResponse = await fetch("http://127.0.0.1:8080/v1/me", { headers });
    if (!meResponse.ok) throw new Error(`/v1/me returned ${meResponse.status}`);
    const me = await meResponse.json();
    const update = await fetch("http://127.0.0.1:8080/v1/me", {
      method: "PATCH",
      headers: { ...headers, "Content-Type": "application/json" },
      body: JSON.stringify({ displayName: "Local Athlete", expectedVersion: me.profile.version }),
    });
    if (!update.ok) throw new Error(`PATCH /v1/me returned ${update.status}`);
    const unauthenticated = await fetch("http://127.0.0.1:8080/v1/me");
    if (unauthenticated.status !== 401)
      throw new Error(`unauthenticated request returned ${unauthenticated.status}`);
    console.log("Smoke test passed.");
  } finally {
    child.kill("SIGTERM");
  }
}

function safeClean() {
  const targets = [
    "apps/mobile/.expo",
    "apps/mobile/dist",
    "apps/mobile/coverage",
    "apps/web/.next",
    "apps/web/coverage",
    "packages/api-client/coverage",
    "backend/bin",
    "backend/coverage.out",
  ];
  const prefix = `${root}${process.platform === "win32" ? "\\" : "/"}`.toLowerCase();
  for (const item of targets) {
    const target = resolve(root, item);
    if (!target.toLowerCase().startsWith(prefix))
      throw new Error(`refusing to clean outside workspace: ${target}`);
    rmSync(target, { recursive: true, force: true });
  }
  console.log("Removed allowlisted build/test outputs.");
}

function spawnDevelopment() {
  const env = workspaceToolEnv(localGoEnv());
  const commands = [
    [go, ["run", "./cmd/api"], backend],
    [go, ["run", "./cmd/worker"], backend],
    [process.execPath, [pnpmScript, "--filter", "@repforge/web", "run", "dev"], root],
    [process.execPath, [pnpmScript, "--filter", "@repforge/mobile", "run", "start"], root],
  ];
  const children = commands.map(([command, args, cwd]) =>
    spawn(command, args, { cwd, env, stdio: "inherit", shell: false }),
  );
  const stop = () => children.forEach((child) => child.kill("SIGTERM"));
  process.on("SIGINT", stop);
  process.on("SIGTERM", stop);
  return Promise.race(
    children.map(
      (child) =>
        new Promise((resolvePromise, rejectPromise) => {
          child.on("error", rejectPromise);
          child.on("exit", (code) =>
            code === 0
              ? resolvePromise()
              : rejectPromise(new Error(`development process exited ${code}`)),
          );
        }),
    ),
  ).finally(stop);
}

function help() {
  console.log(`RepForge commands:
  help              Show this list
  doctor            Check local toolchain and Docker daemon
  setup             Create local env, start infrastructure, migrate, and seed
  infra:up/down     Start or stop local containers without deleting volumes
  db:migrate/seed   Apply migrations or seed the synthetic local profile
  dev               Start API, worker, web, and Expo development servers
  generate          Generate SQL and OpenAPI clients
  format/check      Format or verify formatting
  lint/typecheck    Run static checks
  test              Run unit/component tests
  test:integration  Run real-container integration tests
  build             Build API/worker/web and export the mobile JS bundle
  images:verify     Verify immutable manifests and scan images for high/critical OS CVEs
  images:production-gate
                    Fail unless every production image has inventory, SBOM, and provenance evidence
  secrets           Scan the working tree and Git history with digest-pinned Gitleaks
  security          Run secret, vulnerability, SAST, dependency, and image scans
  smoke             Exercise the running profile vertical slice
  verify            Run the core bootstrap verification suite
  clean             Remove only allowlisted build/test outputs`);
}

async function main() {
  const command = process.argv[2] ?? "help";
  switch (command) {
    case "help":
      help();
      break;
    case "doctor":
      doctor();
      break;
    case "infra:up":
      validateLocalPlatform();
      compose(["up", "-d", "--wait"]);
      break;
    case "infra:down":
      compose(["down"]);
      break;
    case "db:migrate":
      execute(go, ["run", "./cmd/migrate", "up"], {
        cwd: backend,
        env: localGoEnv(),
      });
      break;
    case "db:seed":
      execute(go, ["run", "./cmd/devseed"], {
        cwd: backend,
        env: localGoEnv(),
      });
      break;
    case "setup":
      doctor();
      createLocalEnv();
      validateLocalPlatform();
      compose(["up", "-d", "--wait"]);
      execute(go, ["run", "./cmd/migrate", "up"], {
        cwd: backend,
        env: localGoEnv(),
      });
      execute(go, ["run", "./cmd/devseed"], {
        cwd: backend,
        env: localGoEnv(),
      });
      break;
    case "generate":
      generate();
      break;
    case "generate:check": {
      generateCheck();
      break;
    }
    case "format":
      format(true);
      break;
    case "format:check":
      format(false);
      break;
    case "lint":
      execute(go, ["vet", "./..."], { cwd: backend, env: pureGoEnv });
      executePnpm(["-r", "--if-present", "run", "lint"]);
      break;
    case "typecheck":
      executePnpm(["-r", "--if-present", "run", "typecheck"]);
      break;
    case "test":
      execute(go, ["test", "./..."], { cwd: backend, env: pureGoEnv });
      execute(process.execPath, ["--test", "tools/repforge.test.mjs"], {
        env: workspaceToolEnv(),
      });
      executePnpm(["-r", "--if-present", "run", "test"]);
      break;
    case "test:integration":
      validateLocalPlatform();
      compose(["up", "-d", "--wait"]);
      execute(go, ["test", "-tags=integration", "./..."], {
        cwd: backend,
        env: localGoEnv(),
      });
      break;
    case "build":
      buildGoCommands(["./cmd/..."]);
      executePnpm(["-r", "--if-present", "run", "build"]);
      break;
    case "images:verify":
      verifyImages();
      break;
    case "images:production-gate":
      verifyImages({ production: true });
      break;
    case "secrets":
      scanSecrets();
      break;
    case "security":
      scanSecrets();
      execute(go, ["run", "golang.org/x/vuln/cmd/govulncheck@v1.1.4", "./..."], {
        cwd: backend,
        env: pureGoEnv,
      });
      execute(go, ["run", "github.com/securego/gosec/v2/cmd/gosec@v2.22.10", "-quiet", "./..."], {
        cwd: backend,
        env: pureGoEnv,
      });
      executePnpm(PNPM_AUDIT_ARGS);
      verifyImages();
      break;
    case "smoke":
      await smoke();
      break;
    case "verify":
      doctor();
      validateLocalPlatform();
      configuredImageReferences();
      compose(["config", "--quiet"]);
      generateCheck();
      format(false);
      execute(go, ["vet", "./..."], { cwd: backend, env: pureGoEnv });
      executePnpm(["-r", "--if-present", "run", "lint"]);
      executePnpm(["-r", "--if-present", "run", "typecheck"]);
      execute(go, ["test", "./..."], { cwd: backend, env: pureGoEnv });
      execute(process.execPath, ["--test", "tools/repforge.test.mjs"], {
        env: workspaceToolEnv(),
      });
      executePnpm(["-r", "--if-present", "run", "test"]);
      compose(["up", "-d", "--wait"]);
      execute(go, ["test", "-tags=integration", "./..."], {
        cwd: backend,
        env: localGoEnv(),
      });
      buildGoCommands(["./cmd/..."]);
      executePnpm(["-r", "--if-present", "run", "build"]);
      break;
    case "clean":
      safeClean();
      break;
    case "dev":
      validateLocalPlatform();
      await spawnDevelopment();
      break;
    default:
      throw new Error(`unknown command: ${command}`);
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch((error) => {
    console.error(error.message);
    process.exitCode = 1;
  });
}
