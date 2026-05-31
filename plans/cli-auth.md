# JoCoHunt Real CLI Auth

## TL;DR
> Summary:      Make `jocohunt auth login` complete real JoCoHunt GitHub OAuth by launching controlled Chromium with chromedp, capturing JoCoHunt HttpOnly cookies through CDP, saving them locally, and letting `jocohunt submit` reuse the saved session safely.
> Deliverables:
> - Controlled Chromium auth runner with deterministic test seam
> - CDP cookie capture for JoCoHunt cookies, persisted in the existing auth session store
> - `auth login` default flow that saves a verified session without manual cookie copy
> - `submit` stored-session reuse with base-URL and endpoint leak guards
> - Updated README, skill docs, npm smoke coverage, and evidence scripts
> Effort:       Medium
> Risk:         High - real OAuth depends on browser availability and external GitHub/JoCoHunt redirects, so automated QA must use a mocked browser flow.

## Scope
### Must have
- Preserve current CLI shape: `internal/cli/run.go:48` dispatches `auth`; `internal/cli/run.go:50` dispatches `submit` and `upload`.
- Treat default `jocohunt auth login` as the real end-user path: start JoCoHunt GitHub OAuth via `internal/jocohunt/auth.go:29`, open controlled Chromium through chromedp, wait until JoCoHunt cookies verify against `/api/auth/get-session`, and save the session.
- Keep `auth login --print-url` and `auth login --no-open` as URL-only escape hatches that do not save a session.
- Keep `auth login --session-cookie` as an explicit automation/debug fallback, but remove it as the primary documented user path.
- Use a temp browser profile, clean it after success/failure, and never read the user's existing Chrome profile database.
- Capture HttpOnly cookies through CDP, not page JavaScript.
- Persist a `Cookie` header assembled from all non-expired cookies matching the configured JoCoHunt base host; preserve the existing `AuthSession.SessionCookie` field for submit compatibility.
- Preserve existing auth-file path precedence from `internal/jocohunt/session_store.go:24`: explicit `--auth-file`, `JOCOHUNT_AUTH_FILE`, `JOCOHUNT_CONFIG_DIR/session.json`, then `os.UserConfigDir()/jocohunt/session.json`.
- Ensure `auth status --verify` treats accepted sessions as verified, `null`/401/403 as not logged in, and 5xx/transport failures as errors without leaking cookies.
- Ensure `jocohunt submit` uses stored auth only when the stored session `BaseURL` matches the current `--base-url`; explicit `--session-cookie` and env vars remain opt-in overrides.
- Preserve submit auth precedence from `internal/cli/submit.go:55`: flags, env vars, stored session.
- Reject absolute or scheme-relative `--submit-endpoint` values before any cookie header can leave the configured base origin.
- Add chromedp dependencies to `go.mod`, pin via `go mod tidy`, and keep `scripts/build-npm-binaries.sh:20` cross-compilation working with `CGO_ENABLED=0`.
- Replace or repair the current dirty RED test at `internal/cli/auth_test.go:76` so it uses a deterministic fake browser runner instead of expecting a real GitHub login from a plain `httptest` OAuth URL.

### Must NOT have (guardrails, anti-slop, scope boundaries)
- Do not ask for, collect, log, or store GitHub passwords or GitHub access tokens.
- Do not depend on Codex plugins, browser extensions, a local agent browser, or the user's existing authenticated Chrome profile.
- Do not read Chrome cookie SQLite files or browser profile directories.
- Do not require manual cookie copy for the normal `auth login` path.
- Do not perform production product submission during QA.
- Do not store real cookies or CSRF tokens in tests, docs, logs, error messages, or evidence.
- Do not send stored JoCoHunt cookies to a different `--base-url`, absolute `--submit-endpoint`, or scheme-relative endpoint.
- Do not make browser-dependent tests mandatory in CI unless they are guarded and skipped cleanly when the explicit integration env var is absent.
- Do not remove `--confirm` from live submit.

## Verification strategy
> Zero human intervention - all verification is agent-executed.
- Test decision: TDD with Go `testing`/`httptest` plus chromedp integration tests gated by `JOCOHUNT_AUTH_BROWSER_TEST=1`; Node `node:test` for package/docs smoke
- QA policy: every task has agent-executed scenarios
- Evidence: `evidence/task-<N>-cli-auth.<ext>`

## Execution strategy
### Parallel execution waves
> Target 5-8 tasks per wave. <3 per wave (except final) = under-splitting.
> Extract shared dependencies as Wave-1 tasks to maximize parallelism.

Wave 1 (no dependencies):
- Task 1: Stabilize baseline tests and add a browser-auth seam
- Task 2: Define captured-cookie session persistence and verify semantics
- Task 6: Update user and agent-facing auth contract docs

Wave 2 (after Wave 1):
- Task 3: depends [1, 2]
- Task 4: depends [1, 2, 3]
- Task 5: depends [2]

Wave 3 (after Wave 2):
- Task 7: depends [3, 4, 5, 6]

Critical path: Task 1 -> Task 3 -> Task 4 -> Task 7

### Dependency matrix
| Task | Depends on | Blocks | Can parallelize with |
|------|------------|--------|----------------------|
| 1    | none       | 3, 4, 7 | 2, 6                |
| 2    | none       | 3, 4, 5, 7 | 1, 6             |
| 3    | 1, 2       | 4, 7   | 5                   |
| 4    | 1, 2, 3    | 7      | 5                   |
| 5    | 2          | 7      | 3, 4                |
| 6    | none       | 7      | 1, 2                |
| 7    | 3, 4, 5, 6 | final  | none                |

## Todos
> Implementation + Test = ONE task. Never separate.
> Every task MUST have: References + Acceptance Criteria + QA Scenarios + Commit.

- [ ] 1. Stabilize baseline tests and add a browser-auth seam

  What to do: Record current dirty state first, especially `internal/cli/auth_test.go:76`. Replace the naive no-cookie login test with deterministic TDD coverage that injects a fake browser auth runner. Add the smallest production seam needed in `internal/cli/auth.go`: a package-private runner variable or interface that `runAuthLogin` calls only for the default no-cookie, no-print-url path. The fake runner returns a `jocohunt.AuthSession` with `BaseURL`, a fake multi-cookie `SessionCookie`, and `CreatedAt`; `runAuthLogin` saves it through the existing store. Preserve `--session-cookie`, `--print-url`, and `--no-open` branches for now.
  Must NOT do: Do not add chromedp in this task. Do not launch a real browser from unit tests. Do not weaken existing secret-redaction tests.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [3, 4, 7] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/run.go:18` - root `Run` seam used by CLI tests.
  - Pattern:  `internal/cli/run.go:21` - `--base-url` value that must be stored with captured sessions.
  - Pattern:  `internal/cli/run.go:23` - global `--auth-file` flag.
  - Pattern:  `internal/cli/run.go:48` - auth dispatch point.
  - Pattern:  `internal/cli/auth.go:34` - login flag parser.
  - Pattern:  `internal/cli/auth.go:48` - existing explicit `--session-cookie` storage branch.
  - Pattern:  `internal/cli/auth.go:64` - current no-cookie OAuth branch to replace with browser runner.
  - Test:     `internal/cli/auth_test.go:45` - URL-only login test to preserve.
  - Test:     `internal/cli/auth_test.go:76` - current dirty RED capture test to replace with a fake-runner test.
  - Test:     `internal/cli/auth_test.go:183` - secret leak guard.
  - API/Type: `internal/jocohunt/session_store.go:17` - saved session contract.
  - External: `https://pkg.go.dev/flag#FlagSet` - subcommand flag parsing behavior.

  Acceptance criteria (agent-executable only):
  - [ ] Before editing, capture `git diff -- internal/cli/auth_test.go internal/cli/auth.go | tee evidence/task-1-cli-auth-starting-diff.txt`.
  - [ ] TDD test id written or repaired first: `internal/cli/auth_test.go::TestRunAuthLoginUsesBrowserRunnerAndStoresCapturedSession`. Its initial RED must fail because the production code still opens/prints the old flow or lacks the runner seam; capture with `mkdir -p evidence && go test ./internal/cli -run '^TestRunAuthLoginUsesBrowserRunnerAndStoresCapturedSession$' -count=1 | tee evidence/task-1-cli-auth-runner-red.txt`.
  - [ ] TDD test id written first: `internal/cli/auth_test.go::TestRunAuthLoginPrintURLDoesNotInvokeBrowserRunner`.
  - [ ] TDD test id written first: `internal/cli/auth_test.go::TestRunAuthLoginBrowserRunnerFailureDoesNotWriteSession`.
  - [ ] After implementation, `go test ./internal/cli -run 'TestRunAuthLogin(UsesBrowserRunnerAndStoresCapturedSession|PrintURLDoesNotInvokeBrowserRunner|BrowserRunnerFailureDoesNotWriteSession|SavesSessionCookieWhenCookieProvided|PrintsOAuthURLWhenNoCookieProvided)' -count=1 | tee evidence/task-1-cli-auth-runner-green.txt` exits 0.
  - [ ] `go test ./internal/cli -run 'TestRunAuth(OutputDoesNotLeakSecrets|FileFlagTakesPrecedenceOverConfigDir)' -count=1 | tee evidence/task-1-cli-auth-redaction-green.txt` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: fake browser runner stores a session without launching Chrome
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run '^TestRunAuthLoginUsesBrowserRunnerAndStoresCapturedSession$' -count=1 -v | tee evidence/task-1-cli-auth-runner-store.txt
    Expected: command exits 0; test asserts auth file exists, output contains `Logged in`, and output contains no `better-auth.session_token=`
    Evidence: evidence/task-1-cli-auth-runner-store.txt

  Scenario: URL-only auth path does not save or invoke browser runner
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run '^TestRunAuthLoginPrintURLDoesNotInvokeBrowserRunner$' -count=1 -v | tee evidence/task-1-cli-auth-print-url.txt
    Expected: command exits 0; test asserts OAuth URL output and no auth file write
    Evidence: evidence/task-1-cli-auth-print-url.txt
  ```

  Commit: NO | Message: `test(auth): add browser login seam` | Files: [`internal/cli/auth.go`, `internal/cli/auth_test.go`, `evidence/`]

- [ ] 2. Define captured-cookie session persistence and verify semantics

  What to do: Keep `AuthSession.SessionCookie` as the submit-compatible storage field. Add a small helper in `internal/jocohunt` that converts captured cookies into a stable `Cookie` header by filtering for the current base host, dropping expired cookies, and sorting deterministically. If adding optional cookie metadata to `AuthSession`, keep it additive and preserve old JSON compatibility. Update `VerifySession` so `null`, 401, and 403 return `Authenticated:false` without error; keep 5xx and transport failures as errors. Add redaction for cookie-like values in auth errors before any remote body is returned.
  Must NOT do: Do not change existing JSON field names. Do not require a CSRF token from the browser flow. Do not hardcode a single Better Auth cookie name.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [3, 4, 5, 7] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - API/Type: `internal/jocohunt/session_store.go:17` - `AuthSession` JSON fields.
  - Pattern:  `internal/jocohunt/session_store.go:24` - auth-file path precedence.
  - Pattern:  `internal/jocohunt/session_store.go:41` - session save validation and file write.
  - Pattern:  `internal/jocohunt/session_store.go:81` - session load behavior.
  - Pattern:  `internal/jocohunt/auth.go:77` - `VerifySession` entry point.
  - Pattern:  `internal/jocohunt/auth.go:104` - current non-2xx error handling.
  - Test:     `internal/jocohunt/auth_test.go:48` - accepted session verification test.
  - Test:     `internal/jocohunt/auth_test.go:80` - session-store permission round trip.
  - Test:     `internal/jocohunt/auth_test.go:112` - missing session sentinel.
  - External: `https://better-auth.com/docs/concepts/cookies` - Better Auth cookies are HttpOnly/secure in production and names/prefixes are configurable.
  - External: `https://pkg.go.dev/net/http#Cookie` - cookie header/name-value rules.

  Acceptance criteria (agent-executable only):
  - [ ] TDD test id written first: `internal/jocohunt/auth_test.go::TestCapturedCookiesBuildHeaderForMatchingBaseURLOnly`; capture RED/GREEN under `evidence/task-2-cli-auth-cookie-header-red.txt` and `evidence/task-2-cli-auth-cookie-header-green.txt`.
  - [ ] TDD test id written first: `internal/jocohunt/auth_test.go::TestCapturedCookiesDropExpiredAndForeignDomainCookies`.
  - [ ] TDD test id written first: `internal/jocohunt/auth_test.go::TestClientVerifySessionReportsNotAuthenticatedForNullSession`.
  - [ ] TDD test id written first: `internal/jocohunt/auth_test.go::TestClientVerifySessionReportsNotAuthenticatedForUnauthorizedSession`.
  - [ ] Existing tests `TestClientVerifySessionReportsAuthenticatedWhenSessionEndpointReturnsObject`, `TestSessionStoreRoundTripsSessionWithUserOnlyPermissions`, `TestLoadAuthSessionReturnsErrNoAuthSessionWhenFileMissing`, `TestLoadAuthSessionReturnsDecodeErrorForCorruptFile`, and `TestSaveAuthSessionRejectsEmptyCookie` pass.
  - [ ] `go test ./internal/jocohunt -run 'Test(CapturedCookies|ClientVerifySession|SessionStore|LoadAuthSession|SaveAuthSession)' -count=1 | tee evidence/task-2-cli-auth-session-green.txt` exits 0.
  - [ ] `if rg -n 'better-auth\.session_token=[A-Za-z0-9_-]{3,}|csrf-secret' evidence/task-2-* internal/jocohunt/*.go; then exit 1; else echo 'no auth secrets leaked'; fi | tee evidence/task-2-cli-auth-secret-scan.txt` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: captured cookies become an origin-scoped Cookie header
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run '^TestCapturedCookies(BuildHeaderForMatchingBaseURLOnly|DropExpiredAndForeignDomainCookies)$' -count=1 -v | tee evidence/task-2-cli-auth-cookie-header.txt
    Expected: command exits 0; tests assert matching JoCoHunt cookies are retained and foreign/expired cookies are absent
    Evidence: evidence/task-2-cli-auth-cookie-header.txt

  Scenario: rejected sessions are status results, not secret-bearing errors
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run '^TestClientVerifySessionReportsNotAuthenticatedFor(NullSession|UnauthorizedSession)$' -count=1 -v | tee evidence/task-2-cli-auth-verify-rejected.txt
    Expected: command exits 0; tests assert `Authenticated:false` and no raw cookie in errors/output
    Evidence: evidence/task-2-cli-auth-verify-rejected.txt
  ```

  Commit: NO | Message: `feat(auth): persist captured cookies safely` | Files: [`internal/jocohunt/auth.go`, `internal/jocohunt/auth_test.go`, `internal/jocohunt/session_store.go`, `evidence/`]

- [ ] 3. Implement controlled Chromium cookie capture with chromedp

  What to do: Add chromedp and cdproto imports through `go get`/`go mod tidy`, then implement the real browser runner behind the seam from Task 1. Put the implementation in a new focused file such as `internal/cli/browser_auth.go`. Use a temporary user-data-dir, headful mode by default, optional test/headless options, and `storage.GetCookies` or `network.GetCookies` to read HttpOnly cookies. Poll until `client.VerifySession` accepts the captured cookie header or the timeout expires. Provide actionable errors when Chrome/Chromium is unavailable. Always close chromedp contexts and remove the temp profile.
  Must NOT do: Do not read the user's browser profile. Do not use page JavaScript for cookies. Do not make GitHub credentials visible to the CLI. Do not run browser integration tests by default in CI.

  Parallelization: Can parallel: YES | Wave 2 | Blocks: [4, 7] | Blocked by: [1, 2]

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/auth.go:64` - no-cookie login path that will call the browser runner.
  - Pattern:  `internal/jocohunt/auth.go:77` - verification API used as login completion signal.
  - Pattern:  `internal/jocohunt/session_store.go:17` - captured session result shape.
  - Test:     `internal/cli/auth_test.go:76` - dirty RED capture intent to preserve deterministically.
  - External: `https://pkg.go.dev/github.com/chromedp/chromedp` - chromedp context, allocator, and navigation APIs.
  - External: `https://pkg.go.dev/github.com/chromedp/cdproto/storage#GetCookies` - browser-wide CDP cookie retrieval.
  - External: `https://pkg.go.dev/github.com/chromedp/cdproto/network#Cookie` - CDP cookie fields include `httpOnly`, `secure`, `sameSite`, and expiry.
  - External: `https://github.com/chromedp/cdproto/blob/5737772c319b7b47c7f0d19327f5fca0f369381a/network/types.go#L1059-L1074` - source shape for CDP cookie metadata.

  Acceptance criteria (agent-executable only):
  - [ ] `go mod tidy | tee evidence/task-3-cli-auth-go-mod-tidy.txt` exits 0 and `git diff -- go.mod go.sum | tee evidence/task-3-cli-auth-deps-diff.txt` shows chromedp/cdproto dependencies only.
  - [ ] TDD integration test id written first: `internal/cli/browser_auth_test.go::TestBrowserAuthCapturesHttpOnlyCookieAndVerifiesSessionWithChrome`. It must be skipped unless `JOCOHUNT_AUTH_BROWSER_TEST=1` is set.
  - [ ] TDD test id written first: `internal/cli/browser_auth_test.go::TestBrowserAuthTimeoutRemovesTempProfile`.
  - [ ] TDD test id written first: `internal/cli/browser_auth_test.go::TestBrowserAuthReturnsActionableErrorWhenChromeMissing`.
  - [ ] `go test ./internal/cli -run 'TestBrowserAuth(TimeoutRemovesTempProfile|ReturnsActionableErrorWhenChromeMissing)' -count=1 | tee evidence/task-3-cli-auth-browser-unit.txt` exits 0 without launching a real browser.
  - [ ] `JOCOHUNT_AUTH_BROWSER_TEST=1 go test ./internal/cli -run '^TestBrowserAuthCapturesHttpOnlyCookieAndVerifiesSessionWithChrome$' -count=1 -v | tee evidence/task-3-cli-auth-browser-chrome.txt` exits 0 on a machine with Chrome/Chromium; if Chrome is absent, the actionable missing-browser test must pass and evidence must record the absence.
  - [ ] `go test ./... -run 'TestBrowserAuth|TestRunAuthLogin' -count=1 | tee evidence/task-3-cli-auth-targeted-go.txt` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: controlled Chrome captures an HttpOnly JoCoHunt cookie from a fake login flow
    Tool:     bash
    Steps:    mkdir -p evidence && JOCOHUNT_AUTH_BROWSER_TEST=1 go test ./internal/cli -run '^TestBrowserAuthCapturesHttpOnlyCookieAndVerifiesSessionWithChrome$' -count=1 -v | tee evidence/task-3-cli-auth-chrome-http-only.txt
    Expected: command exits 0 on Chrome-capable hosts; test logs show fake auth server accepted the captured HttpOnly session and temp profile cleanup completed
    Evidence: evidence/task-3-cli-auth-chrome-http-only.txt

  Scenario: browser login timeout cleans temporary profile
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run '^TestBrowserAuthTimeoutRemovesTempProfile$' -count=1 -v | tee evidence/task-3-cli-auth-timeout-cleanup.txt
    Expected: command exits 0; test asserts the temp profile path no longer exists after timeout
    Evidence: evidence/task-3-cli-auth-timeout-cleanup.txt
  ```

  Commit: NO | Message: `feat(auth): capture browser session with chromedp` | Files: [`go.mod`, `go.sum`, `internal/cli/browser_auth.go`, `internal/cli/browser_auth_test.go`, `internal/cli/auth.go`, `evidence/`]

- [ ] 4. Wire real browser login into `jocohunt auth login`

  What to do: Make default `auth login` call `StartGitHubLogin`, then the chromedp runner, then `SaveAuthSession`. Add focused flags only if needed for end-user and QA control: `--browser-exec PATH`, `--browser-timeout DURATION` defaulting to 5 minutes, and `--headless` for CI/local fake OAuth QA. Keep `--print-url` and `--no-open` as no-save URL output. Update messages: success says `Logged in. Stored JoCoHunt session at <path>`; timeout says the login was not completed and no session was saved; no output prints raw cookies. Keep `auth status` and `logout` behavior stable.
  Must NOT do: Do not tell users to copy cookies in the default path. Do not remove the explicit `--session-cookie` fallback. Do not require a network call for local `auth status` without `--verify`.

  Parallelization: Can parallel: YES | Wave 2 | Blocks: [7] | Blocked by: [1, 2, 3]

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/auth.go:34` - login flag parser to extend.
  - Pattern:  `internal/cli/auth.go:37` - explicit session-cookie fallback.
  - Pattern:  `internal/cli/auth.go:39` - callback path.
  - Pattern:  `internal/cli/auth.go:40` - `--print-url`.
  - Pattern:  `internal/cli/auth.go:41` - `--no-open`.
  - Pattern:  `internal/cli/auth.go:64` - OAuth initiation branch.
  - Pattern:  `internal/cli/auth.go:79` - status command.
  - Pattern:  `internal/cli/auth.go:117` - logout command.
  - API/Type: `internal/jocohunt/auth.go:19` - OAuth URL response.
  - API/Type: `internal/jocohunt/session_store.go:41` - session save.
  - Test:     `internal/cli/auth_test.go:14` - explicit cookie fallback.
  - Test:     `internal/cli/auth_test.go:45` - URL-only branch.
  - Test:     `internal/cli/auth_test.go:141` - status verify sends saved cookie.

  Acceptance criteria (agent-executable only):
  - [ ] TDD test id written first: `internal/cli/auth_test.go::TestRunAuthLoginDefaultCapturesBrowserSessionAndSavesVerifiedSession`.
  - [ ] TDD test id written first: `internal/cli/auth_test.go::TestRunAuthLoginNoOpenPrintsURLAndDoesNotSaveSession`.
  - [ ] TDD test id written first: `internal/cli/auth_test.go::TestRunAuthLoginBrowserTimeoutDoesNotSaveSession`.
  - [ ] TDD test id written first: `internal/cli/auth_test.go::TestRunAuthLoginHelpIncludesBrowserFlags`.
  - [ ] Existing tests `TestRunAuthLoginSavesSessionCookieWhenCookieProvided`, `TestRunAuthLoginPrintsOAuthURLWhenNoCookieProvided`, `TestRunAuthLogoutRemovesSavedSession`, `TestRunAuthLogoutHelpDoesNotRemoveSavedSession`, `TestRunAuthStatusVerifyUsesSavedSessionCookie`, `TestRunAuthOutputDoesNotLeakSecrets`, and `TestRunAuthFileFlagTakesPrecedenceOverConfigDir` pass.
  - [ ] `go test ./internal/cli -run 'TestRunAuth' -count=1 | tee evidence/task-4-cli-auth-run-auth-green.txt` exits 0.
  - [ ] `if rg -n 'finish GitHub login in the browser, then run `jocohunt auth login --session-cookie|After GitHub finishes, store the JoCoHunt session|better-auth\.session_token=[A-Za-z0-9_-]{3,}' internal/cli/auth.go README.md skills/jocohunt; then exit 1; else echo 'no manual-cookie primary guidance'; fi | tee evidence/task-4-cli-auth-guidance-scan.txt` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: default login stores a verified session through the fake browser runner
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run '^TestRunAuthLoginDefaultCapturesBrowserSessionAndSavesVerifiedSession$' -count=1 -v | tee evidence/task-4-cli-auth-default-login.txt
    Expected: command exits 0; test asserts saved auth file contains a fake JoCoHunt cookie header and stdout contains `Logged in` without raw cookie values
    Evidence: evidence/task-4-cli-auth-default-login.txt

  Scenario: timeout path leaves no auth file
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run '^TestRunAuthLoginBrowserTimeoutDoesNotSaveSession$' -count=1 -v | tee evidence/task-4-cli-auth-timeout-no-save.txt
    Expected: command exits 0; test asserts error mentions timeout/no completed login and auth file is absent
    Evidence: evidence/task-4-cli-auth-timeout-no-save.txt
  ```

  Commit: NO | Message: `feat(auth): make login capture browser session` | Files: [`internal/cli/auth.go`, `internal/cli/auth_test.go`, `internal/cli/browser_auth.go`, `evidence/`]

- [ ] 5. Keep submit on stored sessions without cookie leaks

  What to do: Update submit path only as needed. Pass current base URL into `runSubmit` from `internal/cli/run.go:50` so stored sessions can be compared to the active base URL. Use stored session only when `stored.BaseURL` is blank or normalized-equal to current base URL; for captured sessions, BaseURL must be set, so mismatched submit fails before network with a non-secret message. Keep explicit `--session-cookie` and `JOCOHUNT_SESSION_COOKIE` as deliberate overrides. Add endpoint validation in `internal/jocohunt/submit.go` before request creation: allow empty or root-relative endpoints, reject absolute and scheme-relative endpoints.
  Must NOT do: Do not send saved cookies to a different base URL. Do not weaken flag/env/stored precedence. Do not remove dry-run no-network behavior. Do not remove `--confirm`.

  Parallelization: Can parallel: YES | Wave 2 | Blocks: [7] | Blocked by: [2]

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/run.go:50` - submit/upload dispatch currently passes auth file only.
  - Pattern:  `internal/cli/submit.go:16` - submit command entry.
  - Pattern:  `internal/cli/submit.go:28` - `--submit-endpoint` flag.
  - Pattern:  `internal/cli/submit.go:51` - stored session load point.
  - Pattern:  `internal/cli/submit.go:55` - cookie precedence.
  - Pattern:  `internal/cli/submit.go:60` - dry-run branch.
  - Pattern:  `internal/cli/submit.go:63` - `--confirm` guard.
  - Pattern:  `internal/jocohunt/submit.go:53` - endpoint parse point.
  - Pattern:  `internal/jocohunt/submit.go:57` - base URL reference resolution.
  - Pattern:  `internal/jocohunt/submit.go:67` - cookie header set point.
  - Test:     `internal/cli/submit_test.go:127` - saved-session submit fallback.
  - Test:     `internal/cli/submit_test.go:169` - explicit cookie overrides saved session.
  - Test:     `internal/cli/submit_test.go:209` - env cookie overrides saved session.
  - Test:     `internal/cli/submit_auth_test.go:11` - stored session and CSRF reuse.
  - API/Type: `internal/jocohunt/submit.go:28` - submit auth options.
  - External: `https://pkg.go.dev/net/url#Parse` - URL parse behavior for absolute and relative refs.

  Acceptance criteria (agent-executable only):
  - [ ] TDD test id written first: `internal/cli/submit_test.go::TestRunSubmitUsesStoredSessionWhenBaseURLMatches`.
  - [ ] TDD test id written first: `internal/cli/submit_test.go::TestRunSubmitRejectsStoredSessionWhenBaseURLDiffers`.
  - [ ] TDD test id written first: `internal/cli/submit_test.go::TestRunSubmitAllowsExplicitCookieWhenStoredSessionBaseURLDiffers`.
  - [ ] TDD test id written first: `internal/cli/submit_test.go::TestRunSubmitRejectsAbsoluteSubmitEndpointBeforeSendingCookie`.
  - [ ] TDD test id written first: `internal/jocohunt/submit_test.go::TestSubmitProductRejectsSchemeRelativeEndpointBeforeCookieHeader`.
  - [ ] Update existing saved-session tests so test fixtures use the active `httptest.Server` URL in `AuthSession.BaseURL` where stored session reuse is expected.
  - [ ] Existing tests `TestRunSubmitDryRunPrintsPayloadWithoutNetwork`, `TestRunSubmitRequiresConfirmationForLiveWrite`, `TestRunSubmitPostsWhenConfirmed`, `TestRunSubmitPrefersCookieFlagOverSavedSession`, `TestRunSubmitPrefersEnvCookieOverSavedSession`, `TestRunSubmitDryRunRedactsSavedSessionSecrets`, and `TestRunSubmitUsesStoredSessionCookie` pass.
  - [ ] `go test ./internal/cli -run 'TestRunSubmit' -count=1 | tee evidence/task-5-cli-auth-submit-cli-green.txt` exits 0.
  - [ ] `go test ./internal/jocohunt -run 'TestSubmitProduct' -count=1 | tee evidence/task-5-cli-auth-submit-core-green.txt` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: stored matching session enables submit without cookie flags
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run '^TestRunSubmitUsesStoredSessionWhenBaseURLMatches$' -count=1 -v | tee evidence/task-5-cli-auth-submit-stored-match.txt
    Expected: command exits 0; test server receives expected fake Cookie header from saved auth file
    Evidence: evidence/task-5-cli-auth-submit-stored-match.txt

  Scenario: stored session is not sent to a mismatched base URL or absolute endpoint
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run '^TestRunSubmitRejects(StoredSessionWhenBaseURLDiffers|AbsoluteSubmitEndpointBeforeSendingCookie)$' -count=1 -v | tee evidence/task-5-cli-auth-submit-leak-guards.txt
    Expected: command exits 0; tests assert no request receives the saved cookie
    Evidence: evidence/task-5-cli-auth-submit-leak-guards.txt
  ```

  Commit: NO | Message: `fix(submit): keep stored auth on origin` | Files: [`internal/cli/run.go`, `internal/cli/submit.go`, `internal/cli/submit_test.go`, `internal/cli/submit_auth_test.go`, `internal/jocohunt/submit.go`, `internal/jocohunt/submit_test.go`, `evidence/`]

- [ ] 6. Update docs, skill contract, and npm smoke for real login

  What to do: Update `README.md`, `skills/jocohunt/SKILL.md`, `skills/jocohunt/SKILL.test.md`, `package.json`, and npm wrapper tests so the documented contract matches the new default. README should say `jocohunt auth login` opens controlled Chromium, the user completes GitHub in that browser, the CLI saves a verified JoCoHunt session, and `submit` reuses it. Keep `--print-url`/`--no-open` documented as URL-only troubleshooting modes. Mention `--session-cookie` only as an advanced automation fallback. Ensure smoke coverage only calls help commands and never launches login.
  Must NOT do: Do not document cookie-copy as the normal path. Do not include real-looking secrets. Do not add live submit to CI.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [7] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - Docs:     `README.md:42` - current login section.
  - Docs:     `README.md:56` - current manual cookie wording to replace.
  - Docs:     `README.md:68` - submit stored-session wording.
  - Docs:     `README.md:95` - security scope.
  - Skill:    `skills/jocohunt/SKILL.md:18` - current `auth login --print-url` example.
  - Skill:    `skills/jocohunt/SKILL.md:19` - current manual cookie example.
  - Skill:    `skills/jocohunt/SKILL.md:28` - current submit authorization policy.
  - Skill:    `skills/jocohunt/SKILL.test.md:11` - auth contract test.
  - Package:  `package.json:22` - scripts block.
  - Package:  `package.json:25` - smoke script.
  - Test:     `npm/wrapper.test.js:63` - smoke script contract.
  - CI:       `.github/workflows/ci.yml:30` - smoke step.

  Acceptance criteria (agent-executable only):
  - [ ] TDD Node test updated first: `npm/wrapper.test.js` test name `package smoke script and docs describe real browser auth without launching login`; capture RED/GREEN under `evidence/task-6-cli-auth-docs-node-red.txt` and `evidence/task-6-cli-auth-docs-node-green.txt`.
  - [ ] `rg -n 'controlled Chromium|GitHub OAuth|auth login|auth status|auth logout|--browser-timeout|--print-url|--no-open|--session-cookie|--confirm|JOCOHUNT_AUTH_FILE|JOCOHUNT_CONFIG_DIR|JOCOHUNT_SESSION_COOKIE' README.md skills/jocohunt/SKILL.md skills/jocohunt/SKILL.test.md | tee evidence/task-6-cli-auth-docs-terms.txt` exits 0 and includes all expected terms.
  - [ ] `if rg -n 'copy.*cookie|manual cookie copy|finish GitHub login.*--session-cookie|After GitHub finishes, store|GitHub password|GitHub token|browser profile database|better-auth\.session_token=[A-Za-z0-9_-]{3,}' README.md skills/jocohunt; then exit 1; else echo 'docs guardrails clean'; fi | tee evidence/task-6-cli-auth-docs-guardrails.txt` exits 0.
  - [ ] `npm test | tee evidence/task-6-cli-auth-npm-test.txt` exits 0.
  - [ ] `npm run smoke | tee evidence/task-6-cli-auth-smoke.txt` exits 0 and only exercises help/read-only commands.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: docs describe real browser login and stored-session submit
    Tool:     bash
    Steps:    mkdir -p evidence && rg -n 'controlled Chromium|GitHub OAuth|jocohunt auth login|jocohunt submit|--confirm' README.md skills/jocohunt/SKILL.md skills/jocohunt/SKILL.test.md | tee evidence/task-6-cli-auth-docs-real-login.txt
    Expected: command exits 0 and output includes the default browser-login flow plus submit reuse
    Evidence: evidence/task-6-cli-auth-docs-real-login.txt

  Scenario: docs do not present manual cookie copy as the normal path
    Tool:     bash
    Steps:    mkdir -p evidence && if rg -n 'manual cookie copy|finish GitHub login.*--session-cookie|After GitHub finishes, store|GitHub password|GitHub token' README.md skills/jocohunt; then exit 1; else echo 'docs safe'; fi | tee evidence/task-6-cli-auth-docs-safe.txt
    Expected: command exits 0 and output is exactly `docs safe`
    Evidence: evidence/task-6-cli-auth-docs-safe.txt
  ```

  Commit: NO | Message: `docs(auth): document real browser login` | Files: [`README.md`, `skills/jocohunt/SKILL.md`, `skills/jocohunt/SKILL.test.md`, `package.json`, `npm/wrapper.test.js`, `evidence/`]

- [ ] 7. Run full automated and real-browser QA evidence

  What to do: Run the full verification stack after Tasks 1-6 are complete. Capture browser-capable evidence with the fake OAuth flow, not real GitHub credentials. Rebuild npm binaries after Go changes. Capture cleanup receipts for temp auth files and temp browser profiles. If Chrome/Chromium is absent locally, capture the actionable missing-browser error and run all non-browser checks; do not mark the browser QA approved until it passes on a Chrome-capable host.
  Must NOT do: Do not authenticate against production GitHub in automated evidence. Do not submit to production. Do not leave temp auth directories or temp browser profiles behind.

  Parallelization: Can parallel: NO | Wave 3 | Blocks: [] | Blocked by: [3, 4, 5, 6]

  References (executor has NO interview context - be exhaustive):
  - CLI:      `cmd/jocohunt/main.go:16` - production CLI entrypoint.
  - CLI:      `internal/cli/auth.go:34` - auth login flags.
  - CLI:      `internal/cli/submit.go:60` - dry-run must be no-network.
  - Build:    `scripts/build-npm-binaries.sh:20` - cross-platform `CGO_ENABLED=0` build.
  - Package:  `package.json:23` - build script.
  - Package:  `package.json:24` - npm tests.
  - Package:  `package.json:25` - smoke script.
  - CI:       `.github/workflows/ci.yml:23` - full Go test command.
  - CI:       `.github/workflows/ci.yml:25` - Node test command.
  - CI:       `.github/workflows/ci.yml:27` - binary build command.
  - CI:       `.github/workflows/ci.yml:30` - smoke command.
  - Release:  `.github/workflows/release.yml:23` - release test command.

  Acceptance criteria (agent-executable only):
  - [ ] `go test -race -shuffle=on -count=1 ./... | tee evidence/task-7-cli-auth-full-go.txt` exits 0.
  - [ ] `JOCOHUNT_AUTH_BROWSER_TEST=1 go test ./internal/cli -run 'Test(BrowserAuthCapturesHttpOnlyCookieAndVerifiesSessionWithChrome|RunAuthLoginE2EFakeOAuthChromeStoresSessionForSubmitDryRun)' -count=1 -v | tee evidence/task-7-cli-auth-real-browser.txt` exits 0 on a Chrome-capable host.
  - [ ] `npm test | tee evidence/task-7-cli-auth-npm-test.txt` exits 0.
  - [ ] `npm run build:binaries | tee evidence/task-7-cli-auth-build-binaries.txt` exits 0.
  - [ ] `npm run smoke | tee evidence/task-7-cli-auth-smoke.txt` exits 0.
  - [ ] `npm pack --dry-run | tee evidence/task-7-cli-auth-pack-dry-run.txt` exits 0 and lists `npm/bin.js`, platform binaries, `README.md`, and `LICENSE`.
  - [ ] Secret scan `if rg -n 'better-auth\.session_token=[A-Za-z0-9_-]{3,}|csrf-[A-Za-z0-9_-]{3,}|github_pat_|ghp_' evidence README.md skills/jocohunt internal/cli internal/jocohunt; then exit 1; else echo 'no secrets found'; fi | tee evidence/task-7-cli-auth-secret-scan.txt` exits 0.
  - [ ] Cleanup scan `find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'jocohunt-auth-*' -print | tee evidence/task-7-cli-auth-temp-scan.txt` shows no leftover temp profile directories from the QA run.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: real Chrome fake OAuth flow stores a session and submit dry-run reuses it
    Tool:     bash
    Steps:    mkdir -p evidence && JOCOHUNT_AUTH_BROWSER_TEST=1 go test ./internal/cli -run '^TestRunAuthLoginE2EFakeOAuthChromeStoresSessionForSubmitDryRun$' -count=1 -v | tee evidence/task-7-cli-auth-e2e-fake-oauth.txt
    Expected: command exits 0; test logs include `Logged in`, submit dry-run JSON contains `"sessionCookie": true`, no production write occurs, and cleanup logs confirm auth/profile temp dirs were removed
    Evidence: evidence/task-7-cli-auth-e2e-fake-oauth.txt

  Scenario: packaged CLI help surfaces remain non-interactive
    Tool:     bash
    Steps:    mkdir -p evidence && npm run build:binaries && npm run smoke | tee evidence/task-7-cli-auth-packaged-smoke.txt
    Expected: command exits 0; output includes auth login/status/logout help and submit help, and does not launch a browser
    Evidence: evidence/task-7-cli-auth-packaged-smoke.txt
  ```

  Commit: NO | Message: `test(auth): verify real browser auth flow` | Files: [`npm/jocohunt`, `npm/jocohunt-darwin-amd64`, `npm/jocohunt-darwin-arm64`, `npm/jocohunt-linux-amd64`, `npm/jocohunt-linux-arm64`, `npm/jocohunt-windows-amd64.exe`, `evidence/`]

## Final verification wave (MANDATORY - after all implementation tasks)
> Runs in PARALLEL. ALL must APPROVE. Surface results to the caller and wait for an explicit "okay" before declaring complete.
- [ ] F1. Plan compliance audit - every task done, every acceptance criterion met, every evidence path exists
- [ ] F2. Code quality review - diagnostics clean, idioms match, no dead code, chromedp seam remains testable, no source file becomes a broad auth framework
- [ ] F3. Real manual QA - browser fake-OAuth, CLI lifecycle, submit dry-run, npm smoke, and cleanup scenarios executed with evidence captured
- [ ] F4. Scope fidelity - no password/token collection, no existing-browser profile reads, no manual-cookie requirement for default login, no production submit, no off-origin cookie send

## Commit strategy
- No commits until the caller explicitly authorizes git commits.
- When commits are authorized, use one logical change per commit with Conventional Commits (`<type>(<scope>): <subject>` body + footer).
- Atomic: every commit builds and passes tests on its own.
- No "WIP" / "fix typo squash later" commits on the final branch - clean up before merge.
- Reference the plan file path in the final commit footer: `Plan: plans/cli-auth.md`.

## Success criteria
- `jocohunt auth login` default path saves a verified JoCoHunt session through controlled Chromium and CDP cookie capture; `submit` reuses that stored session only on the matching base URL; all QA scenarios pass with captured evidence; F1-F4 approve; no real secrets or production writes appear in code, docs, logs, or evidence.
