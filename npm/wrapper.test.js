import { strict as assert } from "node:assert";
import { mkdtempSync, readFileSync, writeFileSync, chmodSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";
import { resolveBinary, toGoTarget } from "./wrapper.js";

test("resolveBinary returns platform binary when present", () => {
  // Given
  const dir = mkdtempSync(join(tmpdir(), "jocohunt-wrapper-"));
  const target = toGoTarget(process.platform, process.arch);
  const bin = join(dir, `jocohunt-${target.goos}-${target.goarch}${target.ext}`);
  writeFileSync(bin, "#!/bin/sh\n");
  chmodSync(bin, 0o755);

  // When
  const resolved = resolveBinary(dir);

  // Then
  assert.equal(resolved, bin);
});

test("resolveBinary returns source fallback when platform binary is absent", () => {
  // Given
  const dir = mkdtempSync(join(tmpdir(), "jocohunt-wrapper-"));
  const target = toGoTarget(process.platform, process.arch);
  const fallback = join(dir, `jocohunt${target.ext}`);
  writeFileSync(fallback, "#!/bin/sh\n");
  chmodSync(fallback, 0o755);

  // When
  const resolved = resolveBinary(dir);

  // Then
  assert.equal(resolved, fallback);
});

test("toGoTarget maps Node win32 x64 to Go windows amd64", () => {
  // Given
  const platform = "win32";
  const arch = "x64";

  // When
  const target = toGoTarget(platform, arch);

  // Then
  assert.deepEqual(target, { goos: "windows", goarch: "amd64", ext: ".exe" });
});

test("toGoTarget maps supported npm platforms to Go targets", () => {
  // Given
  const cases = [
    { platform: "darwin", arch: "arm64", target: { goos: "darwin", goarch: "arm64", ext: "" } },
    { platform: "darwin", arch: "x64", target: { goos: "darwin", goarch: "amd64", ext: "" } },
    { platform: "linux", arch: "arm", target: { goos: "linux", goarch: "arm", ext: "" } },
    { platform: "linux", arch: "arm64", target: { goos: "linux", goarch: "arm64", ext: "" } },
    { platform: "linux", arch: "x64", target: { goos: "linux", goarch: "amd64", ext: "" } },
    { platform: "win32", arch: "arm64", target: { goos: "windows", goarch: "arm64", ext: ".exe" } },
    { platform: "win32", arch: "x64", target: { goos: "windows", goarch: "amd64", ext: ".exe" } },
  ];

  for (const { platform, arch, target } of cases) {
    // When
    const actual = toGoTarget(platform, arch);

    // Then
    assert.deepEqual(actual, target);
  }
});

test("resolveBinary uses Go target names for Node platform names", () => {
  // Given
  const dir = mkdtempSync(join(tmpdir(), "jocohunt-wrapper-"));
  const bin = join(dir, "jocohunt-windows-amd64.exe");
  writeFileSync(bin, "#!/bin/sh\n");
  chmodSync(bin, 0o755);

  // When
  const resolved = resolveBinary(dir, "win32", "x64");

  // Then
  assert.equal(resolved, bin);
});

test("package smoke script and CI exercise auth and submit help", () => {
  // Given
  const packageJSON = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8"));
  const ciWorkflow = readFileSync(new URL("../.github/workflows/ci.yml", import.meta.url), "utf8");

  // When
  const smoke = packageJSON.scripts.smoke;

  // Then
  assert.match(smoke, /auth login --help/);
  assert.match(smoke, /auth status --help/);
  assert.match(smoke, /auth logout --help/);
  assert.match(smoke, /submit --help/);
  assert.match(ciWorkflow, /npm run smoke/);
});

test("package metadata exposes jocohunt-cli as the npm command", () => {
  // Given
  const packageJSON = JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8"));

  // When
  const bin = packageJSON.bin;

  // Then
  assert.equal(packageJSON.name, "jocohunt-cli");
  assert.equal(bin["jocohunt-cli"], "npm/bin.js");
  assert.match(packageJSON.description, /jocohunt-cli/);
});

test("README documents npm-first jocohunt-cli installation", () => {
  // Given
  const readme = readFileSync(new URL("../README.md", import.meta.url), "utf8");

  // When
  const installSection = readme.match(/## 설치[\s\S]*?## 사용법/)?.[0] ?? "";

  // Then
  assert.match(installSection, /npm install -g jocohunt-cli/);
  assert.match(installSection, /jocohunt-cli products --limit 5/);
  assert.match(readme, /https:\/\/www\.npmjs\.com\/package\/jocohunt-cli/);
});

test("GitHub Actions run CLI checks and npm release", () => {
  // Given
  const ciWorkflow = readFileSync(new URL("../.github/workflows/ci.yml", import.meta.url), "utf8");
  const releaseWorkflow = readFileSync(new URL("../.github/workflows/release.yml", import.meta.url), "utf8");

  // Then
  assert.match(ciWorkflow, /go test -race -shuffle=on -count=1 \.\/\.\.\./);
  assert.match(ciWorkflow, /npm test/);
  assert.match(ciWorkflow, /npm pack --dry-run/);
  assert.match(releaseWorkflow, /id-token: write/);
  assert.match(releaseWorkflow, /environment: npm-publish/);
  assert.match(releaseWorkflow, /package-manager-cache: false/);
  assert.match(releaseWorkflow, /npm install -g npm@\^11\.15\.0/);
  assert.match(releaseWorkflow, /npm publish --access public/);
  assert.doesNotMatch(releaseWorkflow, /NODE_AUTH_TOKEN|NPM_TOKEN|--provenance/);
});

test("binary build script packages every supported npm platform target", () => {
  // Given
  const buildScript = readFileSync(new URL("../scripts/build-npm-binaries.sh", import.meta.url), "utf8");

  // When
  const targets = [
    "darwin arm64",
    "darwin amd64",
    "linux arm",
    "linux arm64",
    "linux amd64",
    "windows arm64",
    "windows amd64",
  ];

  // Then
  for (const target of targets) {
    assert.match(buildScript, new RegExp(target));
  }
});
