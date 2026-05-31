# JoCoHunt Submit Safety Hardening

## TL;DR
> Summary:      Finish the in-progress `jocohunt submit` command by hardening payload validation, same-origin endpoint safety, secret redaction, docs, skill guidance, npm packaging, and non-mutating live QA.
> Deliverables:
> - Safe `submit`/`upload` CLI behavior with dry-run, confirm, cookie, CSRF, and stable JSON contracts
> - Hardened `internal/jocohunt` submit client with same-origin endpoint validation and sanitized errors
> - Updated README, skill contract, npm/CI smoke coverage, rebuilt package binaries as needed
> - Evidence for automated tests and live `/submit` redirect/browser checks without production submission
> Effort:       Medium
> Risk:         Medium - live submit API schema is not authenticated/authoritatively verified; production writes must stay out of automated QA.

## Scope
### Must have
- Treat this as an in-progress branch: `internal/cli/submit.go`, `internal/cli/submit_test.go`, `internal/jocohunt/submit.go`, and `internal/jocohunt/submit_test.go` already exist and must be re-read before editing.
- Keep authentication strictly cookie/token based: `--session-cookie`, `JOCOHUNT_SESSION_COOKIE`, `--csrf-token`, and `JOCOHUNT_CSRF_TOKEN`; no OAuth/browser login automation.
- Require `--confirm` plus an authenticated session cookie for any live submit path.
- Make `--dry-run` validate the same required fields as live submit and print the exact canonical JSON payload that would be posted, while redacting secrets to booleans only.
- Restrict `--submit-endpoint` to a same-origin, root-relative API path; reject absolute URLs, scheme-relative URLs, relative paths, and traversal attempts before any network call.
- Keep `submit` and `upload` as supported aliases, and document them consistently in usage/help/docs.
- Preserve read-only commands: `products`, `ideas`, `leaderboard`, and `inspect`.
- Update Korean README and Codex skill text/tests to explain authorized submission without weakening read-only safety for inspection/ranking tasks.
- Verify via Go `testing`/`httptest`, Node `node:test`, npm/package smoke, and live non-mutating `/submit` checks.

### Must NOT have (guardrails, anti-slop, scope boundaries)
- No real production product submission in automated QA.
- No POST/PUT/PATCH/DELETE to `https://jocohunt.jocoding.io` unless a later user message explicitly supplies test-product details and authorizes that exact write.
- No signup, GitHub OAuth automation, credential scraping, browser cookie extraction, voting, comments, or upvote writes.
- No off-origin submit endpoint support; never send JoCoHunt cookies to a caller-supplied absolute URL.
- No logging, printing, committing, or evidence-capturing raw session cookies or CSRF tokens.
- No git auto-commit/rollback feature and no `npm run create-submission` script in this scope; `package.json:22-26` has no such script and the current objective is product submission.
- No broad refactor of the read-only scraper/parser beyond what is needed for submit safety.

## Verification strategy
> Zero human intervention - all verification is agent-executed.
- Test decision: TDD for hardening gaps + Go `testing`/`httptest`, Node `node:test`
- QA policy: every task has agent-executed scenarios
- Evidence: `evidence/task-<N>-<slug>.<ext>`

## Execution strategy
### Parallel execution waves
> Target 5-8 tasks per wave. <3 per wave (except final) = under-splitting.
> Extract shared dependencies as Wave-1 tasks to maximize parallelism.

Wave 1 (no dependencies):
- Task 1: Harden submit endpoint resolution and same-origin request safety
- Task 2: Stabilize submit validation, canonical payload, result JSON, and sanitized errors
- Task 4: Reconcile `submit`/`upload` alias help and read-command regressions
- Task 5: Align README and Codex skill policy with explicit authorized submission

Wave 2 (after Wave 1):
- Task 3: depends [1, 2]
- Task 6: depends [4]

Wave 3 (after Wave 2):
- Task 7: Rebuild/package binaries and verify npm artifact contents
- Task 8: Refresh live non-mutating `/submit` evidence and lock production-write boundaries

Wave 4 (after Wave 3):
- Final verification wave only

Critical path: Task 1 -> Task 3 -> Task 6 -> Task 7 -> Task 8

### Dependency matrix
| Task | Depends on | Blocks | Can parallelize with |
|------|------------|--------|----------------------|
| 1    | none       | 3, 7, 8 | 2, 4, 5 |
| 2    | none       | 3, 7, 8 | 1, 4, 5 |
| 3    | 1, 2       | 7, 8 | 6 |
| 4    | none       | 5, 6, 7 | 1, 2 |
| 5    | 4          | 8 | 1, 2 |
| 6    | 4          | 7, 8 | 3 |
| 7    | 6          | 8 | none |
| 8    | 5, 7       | final | none |

## Todos
> Implementation + Test = ONE task. Never separate.
> Every task MUST have: References + Acceptance Criteria + QA Scenarios + Commit.

- [ ] 1. Harden submit endpoint resolution and same-origin request safety

  What to do: In `internal/jocohunt/submit.go`, replace direct `url.Parse` + `ResolveReference` endpoint handling with a narrow helper that accepts only root-relative API paths such as `/api/submit`. Reject absolute URLs, scheme-relative URLs, non-root-relative paths, empty traversal-normalized paths, and paths that escape `/api/`. Add focused table tests in `internal/jocohunt/submit_test.go` proving rejected endpoints do not hit the test server.
  Must NOT do: Do not allow caller-supplied absolute endpoints; do not send cookies to any host other than the configured JoCoHunt base URL; do not change read-only `Client.get` behavior.

  Parallelization: Can parallel: PARTIAL | Wave 1 | Blocks: [3, 6, 7, 8] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/jocohunt/client.go:59` - existing request construction style and context-aware HTTP calls.
  - Pattern:  `internal/jocohunt/submit.go:53` - current endpoint parse/resolve logic that must be narrowed.
  - Pattern:  `internal/jocohunt/submit.go:57` - current `ResolveReference` use accepts absolute references unless guarded.
  - Pattern:  `internal/jocohunt/submit.go:67` - cookie header is set after URL resolution, so endpoint safety must happen first.
  - Test:     `internal/jocohunt/submit_test.go:13` - current `httptest.Server` request assertion style.
  - External: `https://pkg.go.dev/net/url#URL.ResolveReference` - resolving an absolute reference can replace the base URL, so validate before resolving.
  - External: `https://pkg.go.dev/net/http#NewRequestWithContext` - keep context-bound request construction.

  Acceptance criteria (agent-executable only):
  - [ ] `go test ./internal/jocohunt -run 'TestSubmitProduct(PostsPayloadWithAuthHeaders|RejectsUnsafeEndpoint)'` passes.
  - [ ] A test asserts `SubmitProduct(..., SubmitOptions{Endpoint: "https://evil.example/api/submit", SessionCookie: "better-auth.session_token=abc"})` returns an error before the `httptest.Server` handler is called.
  - [ ] A test asserts a safe custom endpoint such as `/api/products` still posts to the configured test server origin.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: same-origin endpoint accepted
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run 'TestSubmitProductPostsPayloadWithAuthHeaders' -count=1 | tee evidence/task-1-endpoint-happy.txt
    Expected: command exits 0 and output contains "ok  	github.com/yeongyu/jocohunt/internal/jocohunt"
    Evidence: evidence/task-1-endpoint-happy.txt

  Scenario: off-origin endpoint rejected without network
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run 'TestSubmitProductRejectsUnsafeEndpoint' -count=1 | tee evidence/task-1-endpoint-error.txt
    Expected: command exits 0 and the test proves the handler was not called
    Evidence: evidence/task-1-endpoint-error.txt
  ```

  Commit: NO | Message: `fix(submit): restrict submit endpoints to same origin` | Files: [`internal/jocohunt/submit.go`, `internal/jocohunt/submit_test.go`]

- [ ] 2. Stabilize submit validation, canonical payload, result JSON, and sanitized errors

  What to do: Make validation and payload construction reusable by the CLI dry-run path without duplicating rules. Add an exported or package-appropriate helper that validates `SubmitProductInput` and returns the exact canonical payload map used for live POSTs. Add JSON tags to submit result/plan structs so JSON output is stable lower camel case. Sanitize non-2xx error bodies before returning errors: cap length, trim control noise, and redact any cookie/token values available in `SubmitOptions`.
  Must NOT do: Do not leak raw cookie/token values in errors, dry-run output, logs, or evidence; do not change the required field set beyond title, URL, and tagline unless live contract evidence is added.

  Parallelization: Can parallel: PARTIAL | Wave 1 | Blocks: [3, 6, 7, 8] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/jocohunt/submit.go:40` - `SubmitProduct` currently owns validation and payload generation.
  - Pattern:  `internal/jocohunt/submit.go:87` - current error returns raw response body.
  - Pattern:  `internal/jocohunt/submit.go:93` - required field validation rules.
  - Pattern:  `internal/jocohunt/submit.go:110` - canonical submit payload keys: `name`, `title`, `url`, `tagline`, plus optional fields.
  - API/Type: `internal/jocohunt/submit.go:16` - `SubmitProductInput` contract.
  - API/Type: `internal/jocohunt/submit.go:28` - `SubmitOptions` includes secrets that must be redacted.
  - API/Type: `internal/jocohunt/submit.go:34` - `SubmitResult` currently lacks JSON tags.
  - Test:     `internal/jocohunt/submit_test.go:73` - missing field validation pattern.

  Acceptance criteria (agent-executable only):
  - [ ] `go test ./internal/jocohunt -run 'TestSubmitProduct(RejectsMissingRequiredFields|BuildsCanonicalPayload|RedactsSecretValuesFromErrors|ReturnsHelpfulErrorForAuthFailure)'` passes.
  - [ ] A test proves invalid URL values such as `javascript:alert(1)` and `/relative` fail with `url must be an absolute http(s) URL`.
  - [ ] A test proves an error body containing `better-auth.session_token=abc` and a CSRF token is redacted in `err.Error()`.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: canonical payload produced
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run 'TestSubmitProductBuildsCanonicalPayload' -count=1 | tee evidence/task-2-payload-happy.txt
    Expected: command exits 0 and the test asserts lower-case payload keys match the live POST body
    Evidence: evidence/task-2-payload-happy.txt

  Scenario: secret-bearing error sanitized
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run 'TestSubmitProductRedactsSecretValuesFromErrors' -count=1 | tee evidence/task-2-redaction-error.txt
    Expected: command exits 0 and the test asserts no raw cookie or CSRF token appears in the error string
    Evidence: evidence/task-2-redaction-error.txt
  ```

  Commit: NO | Message: `fix(submit): stabilize payload and redact submit errors` | Files: [`internal/jocohunt/submit.go`, `internal/jocohunt/submit_test.go`]

- [ ] 3. Harden CLI dry-run, auth guardrails, CSRF propagation, and secret redaction

  What to do: In `internal/cli/submit.go`, make `--dry-run` call the shared validation/payload helper before printing. Print a request plan with endpoint, canonical payload, and `auth.sessionCookie`/`auth.csrfToken` booleans only. Keep live writes blocked unless `--confirm` and a session cookie are present. Add CLI tests for dry-run validation, secret redaction, env-var auth, flag auth, CSRF header propagation, and non-confirmed write failures.
  Must NOT do: Do not print raw cookies or tokens; do not call the network for dry-run; do not make `--confirm` optional.

  Parallelization: Can parallel: NO | Wave 1 | Blocks: [6, 7, 8] | Blocked by: [1, 2]

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/submit.go:16` - current submit flag parser.
  - Pattern:  `internal/cli/submit.go:51` - env-var fallback for session cookie and CSRF token.
  - Pattern:  `internal/cli/submit.go:53` - current dry-run branch bypasses validation.
  - Pattern:  `internal/cli/submit.go:56` - current `--confirm` guard.
  - Pattern:  `internal/cli/submit.go:62` - live call into `client.SubmitProduct`.
  - Pattern:  `internal/cli/submit.go:83` - current dry-run output struct.
  - Test:     `internal/cli/submit_test.go:13` - dry-run no-network test style.
  - Test:     `internal/cli/submit_test.go:47` - confirm-required test style.
  - Test:     `internal/cli/submit_test.go:67` - confirmed POST test style.

  Acceptance criteria (agent-executable only):
  - [ ] `go test ./internal/cli -run 'TestRunSubmit(DryRunPrintsCanonicalPayloadWithoutNetwork|DryRunRejectsInvalidPayload|RedactsSecretsInDryRun|RequiresConfirmationForLiveWrite|RequiresSessionCookieForConfirmedWrite|PostsWhenConfirmed|PropagatesCSRFToken)'` passes.
  - [ ] Dry-run output contains `"sessionCookie": true` when a cookie is present and does not contain `better-auth.session_token`.
  - [ ] Confirmed submit to `httptest.Server` includes `Cookie` and `X-CSRF-Token` headers when supplied.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: dry-run validates and redacts
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunSubmit(DryRunPrintsCanonicalPayloadWithoutNetwork|RedactsSecretsInDryRun)' -count=1 | tee evidence/task-3-cli-dry-run-happy.txt
    Expected: command exits 0 and tests assert no network call and no secret in output
    Evidence: evidence/task-3-cli-dry-run-happy.txt

  Scenario: confirmed write without cookie fails
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunSubmitRequiresSessionCookieForConfirmedWrite' -count=1 | tee evidence/task-3-cli-auth-error.txt
    Expected: command exits 0 and the test asserts an error mentioning `--session-cookie` or `JOCOHUNT_SESSION_COOKIE`
    Evidence: evidence/task-3-cli-auth-error.txt
  ```

  Commit: NO | Message: `fix(cli): validate and redact submit dry runs` | Files: [`internal/cli/submit.go`, `internal/cli/submit_test.go`]

- [ ] 4. Reconcile `submit`/`upload` alias help and read-command regressions

  What to do: Keep both `submit` and `upload` aliases because `internal/cli/run.go:47` already dispatches both and README mentions both. Update usage text to list both aliases or explicitly document `upload` as an alias. Add tests that `submit --help` and `upload --help` return no error, and that `upload --dry-run` uses the same dry-run plan as `submit`. Add a small regression test that an existing read command still works after submit changes.
  Must NOT do: Do not remove `upload` unless README and skill are changed in the same task and tests prove the intended error; the chosen plan is to keep the alias.

  Parallelization: Can parallel: PARTIAL | Wave 1 | Blocks: [5, 6, 7] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/run.go:38` - top-level command switch.
  - Pattern:  `internal/cli/run.go:47` - current `submit`, `upload` aliases.
  - Pattern:  `internal/cli/run.go:115` - usage text currently lists only `submit`.
  - Test:     `internal/cli/cli_test.go:50` - help test pattern.
  - Test:     `internal/cli/cli_test.go:67` - read-command regression style with `httptest`.
  - Docs:     `README.md:61` - README already mentions `submit`/`upload`.

  Acceptance criteria (agent-executable only):
  - [ ] `go test ./internal/cli -run 'Test_Run_printsHelpWithoutError_whenHelpFlagProvided|TestRunSubmitHelp|TestRunUploadAliasDryRun|Test_Run_printsJSONProducts_whenProductsCommandUsesJSON'` passes.
  - [ ] `go run ./cmd/jocohunt --help` output contains `submit` and `upload`.
  - [ ] `go run ./cmd/jocohunt upload --help` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: alias help works
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunSubmitHelp|TestRunUploadAliasDryRun' -count=1 | tee evidence/task-4-alias-help-happy.txt
    Expected: command exits 0 and tests prove `upload` is an alias for `submit`
    Evidence: evidence/task-4-alias-help-happy.txt

  Scenario: read-only command still works
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'Test_Run_printsJSONProducts_whenProductsCommandUsesJSON' -count=1 | tee evidence/task-4-read-regression.txt
    Expected: command exits 0 and the existing products JSON behavior still passes
    Evidence: evidence/task-4-read-regression.txt
  ```

  Commit: NO | Message: `fix(cli): document submit upload alias in help` | Files: [`internal/cli/run.go`, `internal/cli/cli_test.go`, `internal/cli/submit_test.go`]

- [ ] 5. Align README and Codex skill policy with explicit authorized submission

  What to do: Re-read `README.md`, `skills/jocohunt/SKILL.md`, and `skills/jocohunt/SKILL.test.md` immediately before editing because these files changed during planning. Update Korean docs and skill policy to match final behavior: public read commands stay read-only; product submission is allowed only when the user explicitly asks and provides/authorizes a session cookie; env vars are preferred over command-line secrets; `--dry-run` is recommended before `--confirm`; no real live write occurs without explicit user authorization. Clarify that unauthenticated `/submit` redirects to GitHub login and the CLI does not automate login.
  Must NOT do: Do not claim the live `/api/submit` schema is guaranteed unless Task 8 captures authenticated non-mutating evidence; do not include real secret-looking values in examples.

  Parallelization: Can parallel: PARTIAL | Wave 1 | Blocks: [8] | Blocked by: [3, 4]

  References (executor has NO interview context - be exhaustive):
  - Docs:     `README.md:28` - current product submission section.
  - Docs:     `README.md:57` - current `--submit-endpoint` and CSRF wording needs safer same-origin/configurability wording after Task 1.
  - Docs:     `README.md:59` - security scope section.
  - Skill:    `skills/jocohunt/SKILL.md:3` - skill description includes authenticated submission.
  - Skill:    `skills/jocohunt/SKILL.md:21` - safety section.
  - Skill:    `skills/jocohunt/SKILL.test.md:3` - read-only ranking contract.
  - Skill:    `skills/jocohunt/SKILL.test.md:7` - authorized submission contract.
  - Evidence: `.omo/ulw-submit/evidence/browser-submit-snapshot.txt:15` - unauthenticated live `/submit` shows GitHub login.
  - Evidence: `.omo/ulw-submit/evidence/http-submit-page.txt:4` - live `/submit` first returns HTTP 307.
  - Evidence: `.omo/ulw-submit/evidence/http-submit-page.txt:10` - redirect location is `/sign-in?redirect=%2Fsubmit`.

  Acceptance criteria (agent-executable only):
  - [ ] `rg -n 'better-auth\\.session_token=abc|better-auth\\.session_token=[A-Za-z0-9]|JOCOHUNT_SESSION_COOKIE=.*[^.]' README.md skills/jocohunt` returns no matches.
  - [ ] `rg -n 'dry-run|--confirm|JOCOHUNT_SESSION_COOKIE|JOCOHUNT_CSRF_TOKEN|upload' README.md skills/jocohunt/SKILL.md` returns matches for each expected term.
  - [ ] `rg -n 'read-only mode|must not call write endpoints such as /api/upvote|explicitly authorized product submission' skills/jocohunt/SKILL.test.md skills/jocohunt/SKILL.md` confirms both read and write safety contracts remain documented.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: docs include safe submit flow
    Tool:     bash
    Steps:    mkdir -p evidence && rg -n 'dry-run|--confirm|JOCOHUNT_SESSION_COOKIE|JOCOHUNT_CSRF_TOKEN|upload' README.md skills/jocohunt/SKILL.md | tee evidence/task-5-docs-happy.txt
    Expected: command exits 0 and output includes all safety-critical terms
    Evidence: evidence/task-5-docs-happy.txt

  Scenario: docs do not leak real-looking secrets
    Tool:     bash
    Steps:    mkdir -p evidence && if rg -n 'better-auth\\.session_token=[A-Za-z0-9_-]{6,}|JOCOHUNT_SESSION_COOKIE=.+' README.md skills/jocohunt; then exit 1; else echo 'no raw secrets found'; fi | tee evidence/task-5-docs-secret-error.txt
    Expected: command exits 0 and output is exactly `no raw secrets found`
    Evidence: evidence/task-5-docs-secret-error.txt
  ```

  Commit: NO | Message: `docs(submit): document authorized submit safety` | Files: [`README.md`, `skills/jocohunt/SKILL.md`, `skills/jocohunt/SKILL.test.md`]

- [ ] 6. Align npm/package and CI smoke coverage for submit help

  What to do: Update package/CI smoke so packaged users and CI prove the submit command is reachable without performing a write. Prefer changing `package.json` `smoke` to run both `node npm/bin.js --help` and `node npm/bin.js submit --help`, then update CI to call `npm run smoke`. If release workflow should gate the same behavior, add `npm run smoke` after binary build and before `npm pack`.
  Must NOT do: Do not add live submit to CI; do not add `npm run create-submission`; do not make CI require secrets.

  Parallelization: Can parallel: NO | Wave 2 | Blocks: [7, 8] | Blocked by: [3, 4]

  References (executor has NO interview context - be exhaustive):
  - Package:  `package.json:22` - current scripts object.
  - Package:  `package.json:25` - current smoke only checks top-level help.
  - Pattern:  `npm/bin.js:5` - wrapper forwards all CLI args to the Go binary.
  - Pattern:  `npm/bin.js:14` - wrapper propagates child exit status.
  - Test:     `npm/wrapper.test.js:8` - existing Node `node:test` style.
  - CI:       `.github/workflows/ci.yml:23` - Go tests gate.
  - CI:       `.github/workflows/ci.yml:25` - Node tests gate.
  - CI:       `.github/workflows/ci.yml:30` - current CLI smoke step.
  - Release:  `.github/workflows/release.yml:23` - release test step.
  - Release:  `.github/workflows/release.yml:27` - release binary build step.
  - External: `https://nodejs.org/api/child_process.html` - wrapper uses `spawnSync` and inherited stdio.
  - External: `https://docs.npmjs.com/cli/v11/using-npm/scripts/` - npm script conventions.

  Acceptance criteria (agent-executable only):
  - [ ] `npm test` passes.
  - [ ] `npm run smoke` exits 0 and prints top-level help plus submit help without network mutation.
  - [ ] `rg -n 'npm run smoke' .github/workflows package.json` shows CI references the package smoke script.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: npm smoke reaches submit help
    Tool:     bash
    Steps:    mkdir -p evidence && npm run smoke | tee evidence/task-6-npm-smoke-happy.txt
    Expected: command exits 0 and output contains `Usage of` or `submit`
    Evidence: evidence/task-6-npm-smoke-happy.txt

  Scenario: CI contains no live submit
    Tool:     bash
    Steps:    mkdir -p evidence && if rg -n 'jocohunt submit|npm run create-submission|JOCOHUNT_SESSION_COOKIE' .github/workflows; then exit 1; else echo 'ci has no live submit'; fi | tee evidence/task-6-ci-no-live-submit.txt
    Expected: command exits 0 and output is exactly `ci has no live submit`
    Evidence: evidence/task-6-ci-no-live-submit.txt
  ```

  Commit: NO | Message: `ci(submit): smoke test submit help only` | Files: [`package.json`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `npm/wrapper.test.js`]

- [ ] 7. Rebuild/package binaries and verify npm artifact contents

  What to do: Run the binary build script after Go source changes. Verify generated binaries still include the updated submit help through the npm wrapper. Run `npm pack --dry-run` and confirm package contents remain limited to `npm/`, `README.md`, and `LICENSE` as configured. If binary files change, include them in the commit because this package ships prebuilt binaries.
  Must NOT do: Do not publish to npm; do not run `npm publish`; do not edit release tags.

  Parallelization: Can parallel: NO | Wave 2 | Blocks: [8] | Blocked by: [6]

  References (executor has NO interview context - be exhaustive):
  - Package:  `package.json:13` - npm `files` whitelist.
  - Package:  `package.json:23` - `build:binaries` script.
  - Build:    `scripts/build-npm-binaries.sh:6` - target platform list.
  - Build:    `scripts/build-npm-binaries.sh:20` - cross-platform Go build command.
  - Build:    `scripts/build-npm-binaries.sh:23` - local fallback binary build.
  - Wrapper:  `npm/wrapper.js:23` - binary resolution root.
  - Wrapper:  `npm/wrapper.js:25` - platform binary naming.
  - Wrapper:  `npm/wrapper.js:29` - fallback binary naming.
  - Release:  `.github/workflows/release.yml:29` - release packs npm artifact.

  Acceptance criteria (agent-executable only):
  - [ ] `npm run build:binaries` exits 0.
  - [ ] `node npm/bin.js submit --help` exits 0 after rebuild.
  - [ ] `npm pack --dry-run` exits 0 and output includes `npm/bin.js`, platform binaries, `README.md`, and `LICENSE`.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: rebuilt wrapper exposes submit help
    Tool:     bash
    Steps:    mkdir -p evidence && npm run build:binaries && node npm/bin.js submit --help | tee evidence/task-7-binaries-happy.txt
    Expected: command exits 0 and output contains submit flag names such as `--title`, `--url`, `--dry-run`, and `--confirm`
    Evidence: evidence/task-7-binaries-happy.txt

  Scenario: dry-run package contents only
    Tool:     bash
    Steps:    mkdir -p evidence && npm pack --dry-run | tee evidence/task-7-pack-dry-run.txt
    Expected: command exits 0 and output lists package files without publishing anything
    Evidence: evidence/task-7-pack-dry-run.txt
  ```

  Commit: NO | Message: `build(submit): refresh npm binaries for submit` | Files: [`npm/jocohunt`, `npm/jocohunt-darwin-amd64`, `npm/jocohunt-darwin-arm64`, `npm/jocohunt-linux-amd64`, `npm/jocohunt-linux-arm64`, `npm/jocohunt-windows-amd64.exe`]

- [ ] 8. Refresh live non-mutating `/submit` evidence and lock production-write boundaries

  What to do: Capture current live `/submit` behavior without calling write endpoints. Verify unauthenticated access redirects to `/sign-in?redirect=%2Fsubmit`, browser snapshot shows GitHub login, local dry-run works against the production base URL without network mutation, and live read-only commands still work. Update docs only if live evidence contradicts the wording from Task 5.
  Must NOT do: Do not POST to `/api/submit`; do not pass a real session cookie; do not click GitHub login; do not create a product.

  Parallelization: Can parallel: NO | Wave 2 | Blocks: [final] | Blocked by: [5, 7]

  References (executor has NO interview context - be exhaustive):
  - Evidence: `.omo/ulw-submit/evidence/http-submit-page.txt:4` - prior evidence observed HTTP 307.
  - Evidence: `.omo/ulw-submit/evidence/http-submit-page.txt:10` - prior redirect to `/sign-in?redirect=%2Fsubmit`.
  - Evidence: `.omo/ulw-submit/evidence/browser-submit-snapshot.txt:15` - prior browser snapshot shows sign-in copy.
  - Evidence: `.omo/ulw-submit/evidence/browser-submit-snapshot.txt:17` - prior browser snapshot shows "GitHub로 계속하기".
  - CLI:      `internal/cli/submit.go:53` - dry-run path must remain no-network.
  - CLI:      `internal/cli/submit_test.go:13` - test proves no-network dry-run behavior.
  - CI:       `.github/workflows/ci.yml:32` - existing live smoke is read-only inspect only.

  Acceptance criteria (agent-executable only):
  - [ ] `curl -I -L https://jocohunt.jocoding.io/submit` evidence contains an initial `HTTP/2 307` and `location: /sign-in?redirect=%2Fsubmit`.
  - [ ] Browser snapshot evidence contains "GitHub로 계속하기".
  - [ ] `node npm/bin.js submit --title "QA Dry Run" --url "https://example.com" --tagline "No production write" --dry-run` exits 0 and output contains canonical payload JSON without secrets.
  - [ ] `node npm/bin.js products --limit 1` exits 0 against the live site.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: live submit redirects unauthenticated users
    Tool:     curl
    Steps:    mkdir -p evidence && curl -I -L https://jocohunt.jocoding.io/submit | tee evidence/task-8-live-submit-redirect.txt
    Expected: output contains `HTTP/2 307` and `location: /sign-in?redirect=%2Fsubmit`
    Evidence: evidence/task-8-live-submit-redirect.txt

  Scenario: browser sign-in surface visible without login automation
    Tool:     agent-browser
    Steps:    mkdir -p evidence && agent-browser open https://jocohunt.jocoding.io/submit && agent-browser snapshot -i | tee evidence/task-8-live-submit-browser.txt
    Expected: snapshot contains `GitHub로 계속하기` and no product form was submitted
    Evidence: evidence/task-8-live-submit-browser.txt

  Scenario: production-base dry-run performs no write
    Tool:     bash
    Steps:    mkdir -p evidence && node npm/bin.js submit --title "QA Dry Run" --url "https://example.com" --tagline "No production write" --dry-run | tee evidence/task-8-live-dry-run.txt
    Expected: command exits 0, output contains `"url": "https://example.com"`, and output contains no `better-auth.session_token`
    Evidence: evidence/task-8-live-dry-run.txt

  Scenario: live read-only command still works
    Tool:     bash
    Steps:    mkdir -p evidence && node npm/bin.js products --limit 1 | tee evidence/task-8-live-products.txt
    Expected: command exits 0 and output contains either one product row or `No items found`
    Evidence: evidence/task-8-live-products.txt
  ```

  Commit: NO | Message: `test(submit): capture non-mutating live submit evidence` | Files: [`README.md`, `skills/jocohunt/SKILL.md`]

## Final verification wave (MANDATORY - after all implementation tasks)
> Runs in PARALLEL. ALL must APPROVE. Surface results to the caller and wait for an explicit "okay" before declaring complete.
- [ ] F1. Plan compliance audit - every task done, every acceptance criterion met
- [ ] F2. Code quality review - diagnostics clean, idioms match, no dead code
- [ ] F3. Real manual QA - every QA scenario executed with evidence captured
- [ ] F4. Scope fidelity - nothing extra shipped beyond Must-Have, nothing Must-NOT-Have introduced

## Commit strategy
- The user requested no git commit, and this planning workspace reported `fatal: not a git repository`; every task above is `Commit: NO`.
- If the user later explicitly asks for commits inside a real git worktree, use one logical Conventional Commit per completed task (`<type>(<scope>): <subject>` body + footer).
- Atomic: every optional future commit builds and passes tests on its own.
- No "WIP" / "fix typo squash later" commits on the final branch - clean up before merge.
- Reference the plan file path in any future final commit footer: `Plan: plans/jocohunt-submit.md`.

## Success criteria
- All Must-Have shipped; all QA scenarios pass with captured evidence; F1-F4 approved; no-git/no-commit state recorded unless the user later explicitly authorizes commits.
