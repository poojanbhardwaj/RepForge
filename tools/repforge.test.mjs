import assert from "node:assert/strict";
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { test } from "node:test";
import { pathToFileURL } from "node:url";

import {
  GITLEAKS_IMAGE,
  IMAGE_REFERENCES,
  PNPM_AUDIT_ARGS,
  WORKSPACE_ROOT,
  assertProductionImagePromotionAllowed,
  assertImmutableImageReferences,
  composeImageReferences,
  gitleaksCanary,
  gitleaksDirectoryDockerArguments,
  gitleaksHistoryDockerArguments,
  imageScanPolicy,
  inspectGitRepository,
  isLiteralLoopbackHTTPOrigin,
  resolveBuildMetadata,
  summarizeTrivyInventory,
  summarizeTrivyVulnerabilities,
  syntheticFixtureAllowlist,
  trivyScanArguments,
  unexpectedGitleaksFindings,
  validateDevelopmentCredentialOrigin,
  workspaceRootFromModuleURL,
} from "./repforge.mjs";

const root = join(import.meta.dirname, "..");

test("workspace discovery decodes executable module paths with spaces and Unicode", () => {
  assert.equal(WORKSPACE_ROOT, root);
  const temporaryRoot = mkdtempSync(join(tmpdir(), "Rep Forge prüfung-"));
  try {
    const toolsDirectory = join(temporaryRoot, "tools");
    mkdirSync(toolsDirectory);
    const modulePath = join(toolsDirectory, "repforge.mjs");
    copyFileSync(join(root, "tools", "repforge.mjs"), modulePath);
    assert.equal(workspaceRootFromModuleURL(pathToFileURL(modulePath)), temporaryRoot);

    const result = spawnSync(
      process.execPath,
      [
        "--input-type=module",
        "--eval",
        `import(${JSON.stringify(pathToFileURL(modulePath).href)}).then(({ WORKSPACE_ROOT }) => console.log(WORKSPACE_ROOT))`,
      ],
      { encoding: "utf8", shell: false },
    );
    assert.ifError(result.error);
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stdout.trim(), temporaryRoot);
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
});

test("every Compose image is pinned to an immutable SHA-256 manifest", () => {
  const references = composeImageReferences(readFileSync(join(root, "compose.yaml"), "utf8"));
  assert.equal(references.length, 3);
  assert.deepEqual(new Set(references), new Set(Object.values(IMAGE_REFERENCES)));
  assert.doesNotThrow(() => assertImmutableImageReferences(references));
  assert.throws(() => assertImmutableImageReferences(["redis:8.2.9-alpine"]), /not immutable/);
});

test("CVE waivers apply only to their exact justified image digests", () => {
  assert.equal(
    imageScanPolicy(IMAGE_REFERENCES.postgres).waiverFile,
    "local/trivy/postgres-ignore.yaml",
  );
  assert.equal(imageScanPolicy(IMAGE_REFERENCES.redis).waiverFile, "local/trivy/redis-ignore.yaml");
  assert.equal(imageScanPolicy(IMAGE_REFERENCES.garage).waiverFile, undefined);
  assert.equal(imageScanPolicy(`${IMAGE_REFERENCES.postgres.slice(0, -1)}0`).waiverFile, undefined);
  const localArguments = trivyScanArguments(IMAGE_REFERENCES.postgres, {
    rootPath: "C:/repo",
  });
  assert.ok(localArguments.includes("--ignorefile"));
  assert.ok(localArguments.some((argument) => argument.includes("postgres-ignore.yaml")));
  const productionArguments = trivyScanArguments(IMAGE_REFERENCES.postgres, {
    production: true,
    rootPath: "C:/repo",
  });
  assert.ok(!productionArguments.includes("--ignorefile"));
  assert.ok(!productionArguments.some((argument) => argument.includes("ignore.yaml")));
  assert.equal(productionArguments[productionArguments.indexOf("--exit-code") + 1], "0");
  for (const waiver of ["postgres-ignore.yaml", "redis-ignore.yaml"]) {
    const contents = readFileSync(join(root, "local", "trivy", waiver), "utf8");
    assert.match(contents, /CVE-2026-14456/);
    assert.match(contents, /expired_at: 2026-09-30/);
  }
});

test("missing Trivy inventory is coverage failure and blocks production", () => {
  assert.deepEqual(summarizeTrivyInventory({ Results: null }), {
    present: false,
    packageCount: 0,
  });
  assert.deepEqual(
    summarizeTrivyInventory({
      Results: [{ Class: "os-pkgs", Packages: [{ Name: "libssl3" }] }],
    }),
    { present: true, packageCount: 1 },
  );
  assert.deepEqual(
    summarizeTrivyVulnerabilities({
      Results: [
        {
          Vulnerabilities: [
            { VulnerabilityID: "CVE-2026-14456" },
            { VulnerabilityID: "CVE-2026-14456" },
          ],
        },
      ],
    }),
    { count: 2, ids: ["CVE-2026-14456"] },
  );
  assert.throws(
    () =>
      assertProductionImagePromotionAllowed([
        {
          reference: IMAGE_REFERENCES.garage,
          inventoryPresent: false,
          localOnly: true,
          vulnerabilityCount: 1,
          vulnerabilityIDs: ["CVE-2026-14456"],
          sbomVerified: false,
          provenanceVerified: false,
        },
      ]),
    /production image promotion blocked.*local-only image/s,
  );
  assert.doesNotThrow(() =>
    assertProductionImagePromotionAllowed([
      {
        reference: "example.invalid/release@sha256:" + "a".repeat(64),
        inventoryPresent: true,
        localOnly: false,
        vulnerabilityCount: 0,
        vulnerabilityIDs: [],
        sbomVerified: true,
        provenanceVerified: true,
      },
    ]),
  );
});

test("secret scanner uses exact fixture values and the real directory/history commands", () => {
  assert.doesNotThrow(() => assertImmutableImageReferences([GITLEAKS_IMAGE]));
  assert.match(gitleaksCanary(), /^ghp_[A-Za-z0-9]{36}$/);
  const baseConfig = readFileSync(join(root, ".gitleaks.toml"), "utf8");
  assert.doesNotMatch(baseConfig, /env\.local/);
  const syntheticPassword = "Ab3d".repeat(8);
  const allowedLines = syntheticFixtureAllowlist(`POSTGRES_PASSWORD=${syntheticPassword}\n`);
  assert.deepEqual([...allowedLines], [[1, "POSTGRES_PASSWORD"]]);
  const syntheticAuth0Secret = "a1b2".repeat(16);
  assert.deepEqual(
    [...syntheticFixtureAllowlist(`AUTH0_SECRET=${syntheticAuth0Secret}\n`)],
    [[1, "AUTH0_SECRET"]],
  );
  assert.throws(
    () => syntheticFixtureAllowlist("POSTGRES_PASSWORD=not-generated\n"),
    /refusing allowlist/,
  );
  assert.throws(
    () =>
      syntheticFixtureAllowlist(
        `POSTGRES_PASSWORD=${syntheticPassword} GITHUB_TOKEN=${gitleaksCanary()}\n`,
      ),
    /refusing allowlist/,
  );
  const findings = [
    { File: "/repo/.env.local", RuleID: "generic-api-key", StartLine: 1 },
    { File: "/repo/.env.local", RuleID: "github-pat", StartLine: 2 },
    { File: "/repo/apps/web/.env.local", RuleID: "generic-api-key", StartLine: 1 },
  ];
  assert.deepEqual(unexpectedGitleaksFindings(findings, allowedLines), [findings[1], findings[2]]);
  const directoryArguments = gitleaksDirectoryDockerArguments("C:/fixture:/repo:ro", {
    reportMount: "C:/report:/report",
  });
  assert.ok(directoryArguments.includes("dir"));
  assert.ok(directoryArguments.includes("GITLEAKS_CONFIG_TOML"));
  assert.ok(directoryArguments.includes("/report/findings.json"));
  const historyArguments = gitleaksHistoryDockerArguments("C:/fixture:/repo:ro");
  assert.ok(historyArguments.includes("git"));
  assert.ok(historyArguments.includes("GIT_CONFIG_VALUE_0=/repo"));
});

function runGit(repositoryRoot, argumentsList) {
  const result = spawnSync("git", argumentsList, {
    cwd: repositoryRoot,
    encoding: "utf8",
    stdio: "pipe",
    shell: false,
  });
  assert.ifError(result.error);
  assert.equal(result.status, 0, result.stderr);
  return result.stdout.trim();
}

function commitFixture(repositoryRoot, message) {
  runGit(repositoryRoot, ["add", "."]);
  runGit(repositoryRoot, [
    "-c",
    "user.name=RepForge Test",
    "-c",
    "user.email=repforge-test@invalid.example",
    "-c",
    "commit.gpgsign=false",
    "commit",
    "--quiet",
    "-m",
    message,
  ]);
}

test("Git inspection distinguishes verified unborn, committed, and corrupt repositories", () => {
  const temporaryRoot = mkdtempSync(join(tmpdir(), "repforge-git-inspection-"));
  try {
    const unborn = join(temporaryRoot, "unborn");
    mkdirSync(unborn);
    runGit(unborn, ["init", "--quiet"]);
    assert.deepEqual(inspectGitRepository(unborn), {
      hasHistory: false,
      headCommit: undefined,
    });

    const committed = join(temporaryRoot, "committed");
    mkdirSync(committed);
    runGit(committed, ["init", "--quiet"]);
    writeFileSync(join(committed, "README.md"), "disposable repository\n", "utf8");
    commitFixture(committed, "initial disposable commit");
    const committedState = inspectGitRepository(committed);
    assert.equal(committedState.hasHistory, true);
    assert.match(committedState.headCommit, /^[a-f0-9]{40}$/);

    const corrupt = join(temporaryRoot, "corrupt");
    mkdirSync(corrupt);
    runGit(corrupt, ["init", "--quiet"]);
    writeFileSync(join(corrupt, ".git", "HEAD"), "invalid git metadata\n", "utf8");
    assert.throws(() => inspectGitRepository(corrupt), /Git repository inspection failed/);
  } finally {
    rmSync(temporaryRoot, { recursive: true, force: true });
  }
});

test("Git inspection recognizes disposable history after a credential leaves the working tree", () => {
  const repositoryRoot = mkdtempSync(join(tmpdir(), "repforge-history-state-"));
  try {
    runGit(repositoryRoot, ["init", "--quiet"]);
    const fixturePath = join(repositoryRoot, "credential.txt");
    writeFileSync(fixturePath, `GITHUB_TOKEN=${gitleaksCanary()}\n`, "utf8");
    commitFixture(repositoryRoot, "add disposable credential");
    writeFileSync(fixturePath, "credential removed\n", "utf8");
    commitFixture(repositoryRoot, "remove disposable credential");
    const state = inspectGitRepository(repositoryRoot);
    assert.equal(state.hasHistory, true);
    assert.match(state.headCommit, /^[a-f0-9]{40}$/);
  } finally {
    rmSync(repositoryRoot, { recursive: true, force: true });
  }
});

test("mobile development credential preflight accepts only literal loopback HTTP origins", () => {
  for (const origin of [
    "http://127.0.0.1:8080",
    "http://127.20.30.40",
    "http://[::1]:8080",
    "http://[0:0:0:0:0:0:0:1]:8080/",
  ]) {
    assert.equal(isLiteralLoopbackHTTPOrigin(origin), true, origin);
    assert.doesNotThrow(() =>
      validateDevelopmentCredentialOrigin({
        EXPO_PUBLIC_API_URL: origin,
        EXPO_PUBLIC_AUTH_MODE: "dev",
        EXPO_PUBLIC_DEV_AUTH_TOKEN: "synthetic-token",
      }),
    );
  }
  for (const origin of [
    "https://127.0.0.1:8080",
    "http://192.0.2.10:8080",
    "http://0.0.0.0:8080",
    "http://localhost:8080",
    "http://user@127.0.0.1:8080",
    "http://2130706433:8080",
    "http://127.0.0.1:8080/v1",
    "not-an-origin",
  ]) {
    assert.equal(isLiteralLoopbackHTTPOrigin(origin), false, origin);
    assert.throws(
      () =>
        validateDevelopmentCredentialOrigin({
          EXPO_PUBLIC_API_URL: origin,
          EXPO_PUBLIC_AUTH_MODE: "dev",
          EXPO_PUBLIC_DEV_AUTH_TOKEN: "synthetic-token",
        }),
      /literal loopback IP/,
    );
  }
  assert.doesNotThrow(() =>
    validateDevelopmentCredentialOrigin({ EXPO_PUBLIC_API_URL: "https://api.example.test" }),
  );
});

test("the default Expo server is loopback-only", () => {
  const mobilePackage = JSON.parse(readFileSync(join(root, "apps/mobile/package.json"), "utf8"));
  assert.match(mobilePackage.scripts.start, /--host localhost(?:\s|$)/);
  assert.doesNotMatch(mobilePackage.scripts.start, /--host lan(?:\s|$)/);
});

test("the pnpm security audit covers the complete installed graph", () => {
  assert.deepEqual(PNPM_AUDIT_ARGS, ["audit", "--audit-level", "high"]);
  assert.ok(!PNPM_AUDIT_ARGS.includes("--prod"));
});

test("build metadata accepts a Git commit and deterministic UTC time", () => {
  const metadata = resolveBuildMetadata({
    env: { REPFORGE_BUILD_VERSION: "0.1.0-test" },
    commit: "a".repeat(40),
    now: new Date("2026-08-29T00:00:00Z"),
  });
  assert.deepEqual(metadata, {
    version: "0.1.0-test",
    commit: "a".repeat(40),
    builtAt: "2026-08-29T00:00:00.000Z",
  });
});

test("build metadata rejects unverifiable values", () => {
  assert.throws(
    () => resolveBuildMetadata({ env: { REPFORGE_BUILD_COMMIT: "unknown" } }),
    /40-character/,
  );
  assert.throws(
    () => resolveBuildMetadata({ env: { REPFORGE_BUILD_TIME: "not-built" } }),
    /UTC RFC 3339/,
  );
});
