# JoCoHunt CLI Auth Completion

## TL;DR
> Summary:      Finish the current RED auth work so `jocohunt auth login/status/logout` has a stable session-file contract, can initiate JoCoHunt GitHub OAuth, can store an explicit JoCoHunt session cookie safely, and lets `jocohunt submit` reuse that stored session without per-submit cookie flags.
> Deliverables:
> - Green auth/session tests in `internal/jocohunt` and `internal/cli`
> - `--auth-file` global override plus secure default session storage
> - Auth login/status/logout UX aligned with current tests and no secret leakage
> - Submit auth reuse from stored session, env vars, or explicit flags
> - Korean README, skill guidance, npm/CI smoke coverage, rebuilt npm binaries, and non-mutating tmux/HTTP evidence
> Effort:       Medium
> Risk:         Medium - live JoCoHunt rejects loopback callback URLs, so the CLI can initiate hosted OAuth but cannot automatically capture the browser session cookie without server-side CLI exchange support.

## Scope
### Must have
- Start from the current RED baseline: `go test ./...` fails because RED tests reference missing auth APIs and `--auth-file` support.
- Keep the RED tests; do not delete or weaken them. Make these tests green:
  - `internal/jocohunt/auth_test.go:16` expects `Client.StartGitHubLogin`.
  - `internal/jocohunt/auth_test.go:48` expects `Client.VerifySession`.
  - `internal/jocohunt/auth_test.go:80` expects `AuthSession`, `SaveAuthSession`, `LoadAuthSession`, and `ErrNoAuthSession`.
  - `internal/cli/auth_test.go:13` expects `--auth-file`, "Logged in", "Logged out", "Not logged in", and OAuth URL "not saved" guidance.
  - `internal/cli/submit_test.go:127` expects submit to reuse a saved session when no cookie flag is supplied.
- Add a global `--auth-file PATH` before the command name. Keep `JOCOHUNT_CONFIG_DIR` as compatibility for existing `internal/cli/submit_auth_test.go:12`.
- Store the JoCoHunt session as JSON with `0600` file permissions and a `0700` containing directory; use `os.UserConfigDir()/jocohunt/session.json` by default.
- `jocohunt auth login --session-cookie ...` stores the session, optional CSRF token, current base URL metadata, and creation time; it must never print the raw cookie or CSRF token.
- `jocohunt auth login --no-open` and `--print-url` initiate JoCoHunt GitHub OAuth by POSTing `/api/auth/sign-in/social` and printing the returned GitHub URL plus "not saved" guidance.
- `jocohunt auth status` is local by default; add `--verify` to call `/api/auth/get-session` with the saved cookie and report whether the server accepts it.
- `jocohunt submit --confirm` must reuse the stored session cookie and CSRF token when `--session-cookie`/`--csrf-token` and env vars are absent.
- Preserve submit auth precedence: explicit flags first, then `JOCOHUNT_SESSION_COOKIE`/`JOCOHUNT_CSRF_TOKEN`, then stored auth file.
- Preserve existing read-only commands and submit safety: `submit`/`upload` still require `--confirm` for live writes and `--dry-run` performs no network call.
- Keep all edited Go files under 250 nonblank, non-comment lines; split helpers into new files when needed.

### Must NOT have (guardrails, anti-slop, scope boundaries)
- No real production product submission in automated QA.
- No POST/PUT/PATCH/DELETE to `https://jocohunt.jocoding.io/api/submit` in QA.
- No GitHub password collection, credential scraping, browser cookie extraction, browser-profile database reads, or automated GitHub account actions.
- No claim that `auth login` can automatically capture a browser session cookie through a local callback. Live probing on May 31, 2026 showed JoCoHunt returns `INVALID_CALLBACKURL` for `http://127.0.0.1:<port>/callback`.
- No GitHub device-flow implementation unless JoCoHunt adds a server endpoint that exchanges a GitHub access token for a JoCoHunt session cookie. GitHub device flow gives a GitHub token, not a JoCoHunt Better Auth session.
- No off-origin submit endpoint support and no sending JoCoHunt cookies to caller-supplied absolute URLs.
- No logging, printing, test snapshots, or evidence files containing raw session cookies or CSRF tokens.
- No `git commit`, `git push`, or history manipulation; this workspace is currently not a git repository.
- No broad refactor of list/parse/read-only behavior beyond keeping it green.

## Verification strategy
> Zero human intervention - all verification is agent-executed.
- Test decision: TDD RED -> GREEN with Go `testing`/`httptest`, Node `node:test`, curl HTTP checks, and tmux CLI transcripts
- QA policy: every task has agent-executed scenarios
- Evidence: `evidence/task-<N>-jocohunt-auth-cli.<ext>`

## Execution strategy
### Parallel execution waves
> Target 5-8 tasks per wave. <3 per wave (except final) = under-splitting.
> Extract shared dependencies as Wave-1 tasks to maximize parallelism.

Wave 1 (no dependencies):
- Task 1: Green `internal/jocohunt` auth client methods
- Task 2: Green `internal/jocohunt` session persistence API
- Task 6: Document OAuth feasibility and safety guardrails

Wave 2 (after Wave 1):
- Task 3: depends [2]
- Task 4: depends [1, 2, 3]
- Task 5: depends [2, 3]

Wave 3 (after Wave 2):
- Task 7: depends [4, 5, 6]
- Task 8: depends [7]

Critical path: Task 2 -> Task 3 -> Task 4 -> Task 7 -> Task 8

### Dependency matrix
| Task | Depends on | Blocks | Can parallelize with |
|------|------------|--------|----------------------|
| 1    | none       | 4      | 2, 6                 |
| 2    | none       | 3, 4, 5 | 1, 6                |
| 3    | 2          | 4, 5, 7 | none                |
| 4    | 1, 2, 3    | 7, 8   | 5                   |
| 5    | 2, 3       | 7, 8   | 4                   |
| 6    | none       | 7, 8   | 1, 2                |
| 7    | 4, 5, 6    | 8      | none                |
| 8    | 7          | final  | none                |

## Todos
> Implementation + Test = ONE task. Never separate.
> Every task MUST have: References + Acceptance Criteria + QA Scenarios + Commit.

- [ ] 1. Green `internal/jocohunt` auth client methods

  What to do: In `internal/jocohunt`, expose the client methods required by the RED tests. Keep the existing request-building behavior in `GitHubLoginURL`, but add `StartGitHubLogin(ctx, callbackPath)` as the public method expected by `internal/jocohunt/auth_test.go:34`; either have `GitHubLoginURL` call it or keep `GitHubLoginURL` as a compatibility wrapper. Add `SessionStatus` with at least `Authenticated bool` and implement `VerifySession(ctx, sessionCookie)` to GET `/api/auth/get-session`, send `Cookie`, set `User-Agent` and `Accept: application/json`, return authenticated for a 2xx JSON response containing a non-null user/session object, return unauthenticated for 401/403/null, and return clear errors for malformed responses or network failures.
  Must NOT do: Do not implement GitHub device flow, token exchange, local callback capture, or any browser automation. Do not remove `GitHubLoginURL` if existing callers still use it.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [4] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/jocohunt/auth.go:18` - existing GitHub OAuth URL request logic, payload, headers, and response decode.
  - Pattern:  `internal/jocohunt/auth.go:27` - existing `/api/auth/sign-in/social` endpoint path.
  - Pattern:  `internal/jocohunt/client.go:59` - existing context-aware GET request style.
  - Pattern:  `internal/jocohunt/submit.go:62` - header-setting style for user agent, accept, origin, and cookies.
  - Test:     `internal/jocohunt/auth_test.go:16` - RED test for `StartGitHubLogin`.
  - Test:     `internal/jocohunt/auth_test.go:48` - RED test for `VerifySession`.
  - External: `https://better-auth.com/docs/basic-usage` - Better Auth `signIn.social` supports provider plus `callbackURL`.
  - External: `https://better-auth.com/docs/concepts/session-management` - Better Auth sessions are cookie-based and `/get-session` returns active session data.

  Acceptance criteria (agent-executable only):
  - [ ] Before editing, `mkdir -p evidence && go test ./internal/jocohunt -run 'TestClient(StartGitHubLoginReturnsOAuthURLWhenServerProvidesRedirect|VerifySessionReportsAuthenticatedWhenSessionEndpointReturnsObject)' -count=1 | tee evidence/task-1-jocohunt-auth-client-red.txt` exits non-zero with missing method errors.
  - [ ] After implementation, `go test ./internal/jocohunt -run 'TestClient(StartGitHubLoginReturnsOAuthURLWhenServerProvidesRedirect|VerifySessionReportsAuthenticatedWhenSessionEndpointReturnsObject)' -count=1 | tee evidence/task-1-jocohunt-auth-client-green.txt` exits 0.
  - [ ] Add or keep a test proving `VerifySession` sends exactly the saved `Cookie` header to `/api/auth/get-session`.
  - [ ] `rg -n 'device|password|cookie jar|browser profile' internal/jocohunt/auth.go internal/jocohunt/*auth*.go` returns no implementation of device flow, password collection, or browser cookie scraping.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: GitHub login URL starts through JoCoHunt auth endpoint
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run 'TestClientStartGitHubLoginReturnsOAuthURLWhenServerProvidesRedirect' -count=1 | tee evidence/task-1-jocohunt-auth-cli-login-url.txt
    Expected: command exits 0 and the test asserts POST `/api/auth/sign-in/social` with provider `github` and callbackURL `/submit`
    Evidence: evidence/task-1-jocohunt-auth-cli-login-url.txt

  Scenario: saved cookie is used for session verification
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run 'TestClientVerifySessionReportsAuthenticatedWhenSessionEndpointReturnsObject' -count=1 | tee evidence/task-1-jocohunt-auth-cli-verify.txt
    Expected: command exits 0 and the test asserts `/api/auth/get-session` receives `Cookie: better-auth.session_token=abc`
    Evidence: evidence/task-1-jocohunt-auth-cli-verify.txt
  ```

  Commit: NO | Message: `feat(auth): add jocohunt auth client methods` | Files: [`internal/jocohunt/auth.go`, `internal/jocohunt/auth_test.go`]

- [ ] 2. Green `internal/jocohunt` session persistence API

  What to do: Add session persistence in `internal/jocohunt` to satisfy the RED tests. Define `AuthSession` with JSON fields for `baseURL`, `sessionCookie`, optional `csrfToken`, and `createdAt`. Define `ErrNoAuthSession`. Implement `SaveAuthSession(path string, session AuthSession) error` and `LoadAuthSession(path string) (AuthSession, error)`. Create the containing directory with `0700`, write the file with `0600`, `chmod` to `0600` after write, return `ErrNoAuthSession` for missing files, return contextual decode errors for corrupt JSON, and reject saving an empty session cookie.
  Must NOT do: Do not keep two incompatible session formats. Do not store GitHub tokens. Do not print or log session secrets.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [3, 4, 5] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/session_store.go:12` - current CLI-local JSON fields that should migrate into `jocohunt.AuthSession`.
  - Pattern:  `internal/cli/session_store.go:44` - existing `0700` directory creation.
  - Pattern:  `internal/cli/session_store.go:51` - existing `0600` file creation.
  - Pattern:  `internal/cli/session_store.go:62` - existing post-write chmod.
  - Test:     `internal/jocohunt/auth_test.go:80` - RED test for round-trip storage and permissions.
  - Test:     `internal/jocohunt/auth_test.go:112` - RED test for `ErrNoAuthSession`.
  - External: `https://pkg.go.dev/os#UserConfigDir` - official per-user config root API.
  - External: `https://pkg.go.dev/os#MkdirAll` - directory permission API.
  - External: `https://pkg.go.dev/os#WriteFile` - file permission API; use `OpenFile` if preserving the explicit close/chmod pattern.

  Acceptance criteria (agent-executable only):
  - [ ] Before editing, `mkdir -p evidence && go test ./internal/jocohunt -run 'Test(SessionStoreRoundTripsSessionWithUserOnlyPermissions|LoadAuthSessionReturnsErrNoAuthSessionWhenFileMissing)' -count=1 | tee evidence/task-2-jocohunt-auth-session-red.txt` exits non-zero with missing type/function errors.
  - [ ] After implementation, `go test ./internal/jocohunt -run 'Test(SessionStoreRoundTripsSessionWithUserOnlyPermissions|LoadAuthSessionReturnsErrNoAuthSessionWhenFileMissing)' -count=1 | tee evidence/task-2-jocohunt-auth-session-green.txt` exits 0.
  - [ ] Add a corrupt JSON test such as `TestLoadAuthSessionReturnsDecodeErrorForCorruptFile` and verify it exits 0.
  - [ ] Add an empty-cookie save test such as `TestSaveAuthSessionRejectsEmptyCookie` and verify it exits 0.
  - [ ] `awk 'NF && $1 !~ /^\\/\\// {n++} END {exit n>250}' internal/jocohunt/auth.go internal/jocohunt/auth_session.go` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: auth session round-trips with user-only permissions
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run 'TestSessionStoreRoundTripsSessionWithUserOnlyPermissions' -count=1 | tee evidence/task-2-jocohunt-auth-cli-session-store.txt
    Expected: command exits 0 and the test asserts the session file mode is `0600`
    Evidence: evidence/task-2-jocohunt-auth-cli-session-store.txt

  Scenario: missing auth file is typed as no session
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/jocohunt -run 'TestLoadAuthSessionReturnsErrNoAuthSessionWhenFileMissing' -count=1 | tee evidence/task-2-jocohunt-auth-cli-missing-session.txt
    Expected: command exits 0 and the test asserts `errors.Is(err, ErrNoAuthSession)`
    Evidence: evidence/task-2-jocohunt-auth-cli-missing-session.txt
  ```

  Commit: NO | Message: `feat(auth): persist jocohunt auth sessions` | Files: [`internal/jocohunt/auth_session.go`, `internal/jocohunt/auth_test.go`]

- [ ] 3. Add global `--auth-file` and unify CLI session path resolution

  What to do: Add `--auth-file PATH` to the top-level `flag.NewFlagSet` in `internal/cli/run.go` so tests can pass it before the command. Thread the resolved auth file path into `runAuth` and `runSubmit`. Replace CLI-local `storedSession` persistence in `internal/cli/session_store.go` with thin path-resolution helpers and wrappers around `jocohunt.SaveAuthSession`/`LoadAuthSession`. Preserve `JOCOHUNT_CONFIG_DIR` compatibility by resolving to `$JOCOHUNT_CONFIG_DIR/session.json` when `--auth-file` is blank; otherwise default to `os.UserConfigDir()/jocohunt/session.json`.
  Must NOT do: Do not require `--auth-file` for normal users. Do not remove `JOCOHUNT_CONFIG_DIR` because `internal/cli/submit_auth_test.go:12` still uses it. Do not change `--base-url` or `--timeout` semantics.

  Parallelization: Can parallel: NO | Wave 2 | Blocks: [4, 5, 7] | Blocked by: [2]

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/run.go:18` - current top-level `Run` entry.
  - Pattern:  `internal/cli/run.go:21` - current global flags; add `--auth-file` here.
  - Pattern:  `internal/cli/run.go:47` - auth command dispatch that must receive auth path.
  - Pattern:  `internal/cli/run.go:49` - submit/upload dispatch that must receive auth path.
  - Pattern:  `internal/cli/session_store.go:79` - current default path resolver.
  - Test:     `internal/cli/auth_test.go:20` - RED global `--auth-file` use before `auth`.
  - Test:     `internal/cli/submit_test.go:147` - RED global `--auth-file` use before `submit`.
  - Test:     `internal/cli/submit_auth_test.go:12` - existing `JOCOHUNT_CONFIG_DIR` compatibility test.
  - External: `https://pkg.go.dev/flag` - top-level `FlagSet` parsing behavior.

  Acceptance criteria (agent-executable only):
  - [ ] Before editing, `mkdir -p evidence && go test ./internal/cli -run 'TestRunAuthLoginSavesSessionCookieWhenCookieProvided|TestRunSubmitUsesSavedSessionWhenCookieFlagMissing' -count=1 | tee evidence/task-3-jocohunt-auth-file-red.txt` exits non-zero because `--auth-file` is undefined or auth session APIs are missing.
  - [ ] After implementation, `go test ./internal/cli -run 'TestRunAuthLoginSavesSessionCookieWhenCookieProvided|TestRunSubmitUsesSavedSessionWhenCookieFlagMissing|TestRunSubmitUsesStoredSessionCookie' -count=1 | tee evidence/task-3-jocohunt-auth-file-green.txt` exits 0.
  - [ ] `go run ./cmd/jocohunt --auth-file "$(mktemp -u)/session.json" --help` exits 0.
  - [ ] Add a test proving `--auth-file` takes precedence over `JOCOHUNT_CONFIG_DIR`.
  - [ ] `awk 'NF && $1 !~ /^\\/\\// {n++} END {exit n>250}' internal/cli/run.go internal/cli/session_store.go` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: explicit auth file stores session
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunAuthLoginSavesSessionCookieWhenCookieProvided' -count=1 | tee evidence/task-3-jocohunt-auth-cli-auth-file.txt
    Expected: command exits 0 and the test proves `auth status` reads the same `--auth-file`
    Evidence: evidence/task-3-jocohunt-auth-cli-auth-file.txt

  Scenario: config-dir compatibility remains
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunSubmitUsesStoredSessionCookie' -count=1 | tee evidence/task-3-jocohunt-auth-cli-config-dir.txt
    Expected: command exits 0 and the existing `JOCOHUNT_CONFIG_DIR` test still passes
    Evidence: evidence/task-3-jocohunt-auth-cli-config-dir.txt
  ```

  Commit: NO | Message: `feat(cli): add auth file session storage` | Files: [`internal/cli/run.go`, `internal/cli/session_store.go`, `internal/cli/auth.go`, `internal/cli/submit.go`, `internal/cli/auth_test.go`, `internal/cli/submit_test.go`]

- [ ] 4. Align auth login/status/logout UX with the CLI contract

  What to do: Update `internal/cli/auth.go` to use `jocohunt.AuthSession` and the auth file path from Task 3. `auth login --session-cookie` should save `BaseURL`, `SessionCookie`, optional `CSRFToken`, and `CreatedAt`, then print a "Logged in" confirmation and the file path without revealing secrets. `auth login --no-open` and `--print-url` should call `StartGitHubLogin`, print the GitHub URL, and print guidance containing "not saved". Normal `auth login` should open the browser and also state that the session is not saved until a JoCoHunt cookie is explicitly stored. `auth status` should print "Logged in" or "Not logged in" locally. Add `auth status --verify` to call `VerifySession` and print a non-secret server-verified result. `auth logout` should remove the auth file and print "Logged out".
  Must NOT do: Do not attempt to read browser cookies after opening GitHub. Do not print cookie/token values in any status, login, logout, error, or evidence path.

  Parallelization: Can parallel: PARTIAL | Wave 2 | Blocks: [7, 8] | Blocked by: [1, 2, 3]

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/auth.go:16` - auth subcommand dispatcher.
  - Pattern:  `internal/cli/auth.go:34` - current login flag parser.
  - Pattern:  `internal/cli/auth.go:48` - current manual cookie store branch.
  - Pattern:  `internal/cli/auth.go:59` - current OAuth URL initiation call.
  - Pattern:  `internal/cli/auth.go:74` - current status implementation.
  - Pattern:  `internal/cli/auth.go:91` - current logout implementation.
  - Test:     `internal/cli/auth_test.go:13` - RED login-save/status test and expected "Logged in".
  - Test:     `internal/cli/auth_test.go:44` - RED OAuth URL/no-open test and expected "not saved".
  - Test:     `internal/cli/auth_test.go:75` - RED logout/status-after-logout test and expected "Logged out"/"Not logged in".
  - External: `https://better-auth.com/docs/basic-usage` - social login redirects to the provider and then returns to the application.
  - External: `https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps` - web OAuth redirects and device flow constraints.

  Acceptance criteria (agent-executable only):
  - [ ] Before editing, `mkdir -p evidence && go test ./internal/cli -run 'TestRunAuth(LoginSavesSessionCookieWhenCookieProvided|LoginPrintsOAuthURLWhenNoCookieProvided|LogoutRemovesSavedSession)' -count=1 | tee evidence/task-4-jocohunt-auth-ux-red.txt` exits non-zero or fails assertions on current output.
  - [ ] After implementation, `go test ./internal/cli -run 'TestRunAuth(LoginSavesSessionCookieWhenCookieProvided|LoginPrintsOAuthURLWhenNoCookieProvided|LogoutRemovesSavedSession)' -count=1 | tee evidence/task-4-jocohunt-auth-ux-green.txt` exits 0.
  - [ ] Add `TestRunAuthStatusVerifyUsesSavedSessionCookie` with `httptest.Server`; it must pass and assert `/api/auth/get-session` receives the saved cookie.
  - [ ] Add `TestRunAuthOutputDoesNotLeakSecrets` and verify stdout/stderr do not contain `better-auth.session_token=abc` or `csrf`.
  - [ ] `go run ./cmd/jocohunt auth --help` exits 0 or returns help without an application error.
  - [ ] `awk 'NF && $1 !~ /^\\/\\// {n++} END {exit n>250}' internal/cli/auth.go internal/cli/session_store.go` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: login, status, logout use one auth file
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunAuth(LoginSavesSessionCookieWhenCookieProvided|LogoutRemovesSavedSession)' -count=1 | tee evidence/task-4-jocohunt-auth-cli-login-logout.txt
    Expected: command exits 0 and tests assert "Logged in", "Logged out", and "Not logged in"
    Evidence: evidence/task-4-jocohunt-auth-cli-login-logout.txt

  Scenario: OAuth URL is printed but not claimed as saved
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunAuthLoginPrintsOAuthURLWhenNoCookieProvided' -count=1 | tee evidence/task-4-jocohunt-auth-cli-oauth-url.txt
    Expected: command exits 0 and output contains a GitHub OAuth URL plus "not saved"
    Evidence: evidence/task-4-jocohunt-auth-cli-oauth-url.txt
  ```

  Commit: NO | Message: `feat(auth): align login status logout ux` | Files: [`internal/cli/auth.go`, `internal/cli/auth_test.go`, `internal/cli/session_store.go`]

- [ ] 5. Make `submit` reuse saved auth without manual cookie flags

  What to do: Update `internal/cli/submit.go` to load `jocohunt.AuthSession` from the auth file path from Task 3. Preserve source precedence: `--session-cookie`/`--csrf-token`, then `JOCOHUNT_SESSION_COOKIE`/`JOCOHUNT_CSRF_TOKEN`, then saved session. Keep current validation before dry-run, keep dry-run no-network behavior, and keep `--confirm` required. Include saved CSRF token in `SubmitOptions`. Do not block saved-session reuse when `AuthSession.BaseURL` differs from `--base-url`; the current RED test at `internal/cli/submit_test.go:130` stores the production base URL and submits to an `httptest.Server`.
  Must NOT do: Do not require a cookie flag after the user has stored an auth session. Do not print saved cookies or tokens in dry-run JSON. Do not make live submit possible without `--confirm`.

  Parallelization: Can parallel: PARTIAL | Wave 2 | Blocks: [7, 8] | Blocked by: [2, 3]

  References (executor has NO interview context - be exhaustive):
  - Pattern:  `internal/cli/submit.go:16` - current submit command entry.
  - Pattern:  `internal/cli/submit.go:51` - current stored session load point.
  - Pattern:  `internal/cli/submit.go:55` - current flag/env/stored cookie precedence.
  - Pattern:  `internal/cli/submit.go:60` - dry-run branch that must remain no-network.
  - Pattern:  `internal/cli/submit.go:63` - `--confirm` guard.
  - Pattern:  `internal/cli/submit.go:69` - live submit call.
  - Test:     `internal/cli/submit_test.go:16` - dry-run no-network test.
  - Test:     `internal/cli/submit_test.go:57` - live write confirmation guard.
  - Test:     `internal/cli/submit_test.go:95` - confirmed submit cookie header test.
  - Test:     `internal/cli/submit_test.go:127` - RED saved-session submit test.
  - Test:     `internal/cli/submit_auth_test.go:11` - existing saved session and CSRF reuse test.
  - API/Type: `internal/jocohunt/submit.go:28` - `SubmitOptions` cookie and CSRF fields.

  Acceptance criteria (agent-executable only):
  - [ ] Before editing, `mkdir -p evidence && go test ./internal/cli -run 'TestRunSubmitUsesSavedSessionWhenCookieFlagMissing' -count=1 | tee evidence/task-5-jocohunt-submit-auth-red.txt` exits non-zero due missing `jocohunt.SaveAuthSession`/`AuthSession` or `--auth-file`.
  - [ ] After implementation, `go test ./internal/cli -run 'TestRunSubmit(UsesSavedSessionWhenCookieFlagMissing|UsesStoredSessionCookie|PostsWhenConfirmed|RequiresConfirmationForLiveWrite|DryRunPrintsPayloadWithoutNetwork|RejectsMissingFields)' -count=1 | tee evidence/task-5-jocohunt-submit-auth-green.txt` exits 0.
  - [ ] Add `TestRunSubmitPrefersCookieFlagOverSavedSession` and `TestRunSubmitPrefersEnvCookieOverSavedSession`; both pass.
  - [ ] Dry-run output contains only auth booleans and does not contain `better-auth.session_token` or raw CSRF values.
  - [ ] `awk 'NF && $1 !~ /^\\/\\// {n++} END {exit n>250}' internal/cli/submit.go internal/cli/session_store.go` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: submit uses auth file without cookie flag
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunSubmitUsesSavedSessionWhenCookieFlagMissing' -count=1 | tee evidence/task-5-jocohunt-auth-cli-submit-saved.txt
    Expected: command exits 0 and the httptest server receives `Cookie: better-auth.session_token=saved`
    Evidence: evidence/task-5-jocohunt-auth-cli-submit-saved.txt

  Scenario: dry-run remains no-network and redacted
    Tool:     bash
    Steps:    mkdir -p evidence && go test ./internal/cli -run 'TestRunSubmitDryRunPrintsPayloadWithoutNetwork' -count=1 | tee evidence/task-5-jocohunt-auth-cli-submit-dry-run.txt
    Expected: command exits 0 and the test proves no HTTP request was made
    Evidence: evidence/task-5-jocohunt-auth-cli-submit-dry-run.txt
  ```

  Commit: NO | Message: `feat(submit): reuse saved auth sessions` | Files: [`internal/cli/submit.go`, `internal/cli/submit_test.go`, `internal/cli/submit_auth_test.go`]

- [ ] 6. Document OAuth feasibility and safety guardrails

  What to do: Update `README.md`, `skills/jocohunt/SKILL.md`, and `skills/jocohunt/SKILL.test.md` to match the supported contract. Explain that `auth login` starts the JoCoHunt-hosted GitHub OAuth flow and can open or print the URL, but the session is only stored after the user explicitly provides a JoCoHunt session cookie through `auth login --session-cookie` or an auth-file workflow. Document `auth status`, `auth status --verify`, `auth logout`, `--auth-file`, `JOCOHUNT_CONFIG_DIR`, `JOCOHUNT_SESSION_COOKIE`, and `JOCOHUNT_CSRF_TOKEN`. State that `submit` reuses a stored session and therefore does not need cookie flags after login. Include the live server guardrail that arbitrary loopback callback URLs are rejected, so automatic cookie capture is not in scope.
  Must NOT do: Do not include real-looking secrets in docs. Do not say the CLI collects GitHub passwords, reads browser cookies, completes device flow, or captures a local callback.

  Parallelization: Can parallel: YES | Wave 1 | Blocks: [7, 8] | Blocked by: []

  References (executor has NO interview context - be exhaustive):
  - Docs:     `README.md:28` - current product registration section only explains env/flag cookies.
  - Docs:     `README.md:50` - current direct cookie flag example.
  - Docs:     `README.md:57` - current CSRF wording.
  - Docs:     `README.md:59` - current security scope.
  - Skill:    `skills/jocohunt/SKILL.md:18` - current `auth login --print-url` command example.
  - Skill:    `skills/jocohunt/SKILL.md:19` - current manual session cookie store command.
  - Skill:    `skills/jocohunt/SKILL.md:28` - current submit authorization policy.
  - Skill:    `skills/jocohunt/SKILL.test.md:11` - current GitHub auth contract.
  - Live:     Planning probe on May 31, 2026: POST `/api/auth/sign-in/social` with `callbackURL:"/submit"` returned 200 and a GitHub OAuth URL; the same endpoint with `callbackURL:"http://127.0.0.1:43119/callback"` returned 403 `INVALID_CALLBACKURL`.
  - External: `https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps` - GitHub device flow is for CLI/headless apps, but it returns a GitHub token rather than a JoCoHunt Better Auth session.
  - External: `https://better-auth.com/docs/basic-usage` - Better Auth social login redirects to a provider and then back to the application callback URL.

  Acceptance criteria (agent-executable only):
  - [ ] `rg -n 'auth login|auth status|auth logout|--auth-file|JOCOHUNT_CONFIG_DIR|JOCOHUNT_SESSION_COOKIE|JOCOHUNT_CSRF_TOKEN|not saved|--confirm' README.md skills/jocohunt/SKILL.md skills/jocohunt/SKILL.test.md | tee evidence/task-6-jocohunt-auth-docs-terms.txt` exits 0 and includes each expected term.
  - [ ] `if rg -n 'GitHub password|browser cookie|device flow completes|automatically capture|better-auth\\.session_token=[A-Za-z0-9_-]{6,}|csrf[A-Za-z0-9_-]{4,}' README.md skills/jocohunt; then exit 1; else echo 'docs guardrails clean'; fi | tee evidence/task-6-jocohunt-auth-docs-guardrails.txt` exits 0.
  - [ ] `rg -n 'read-only mode|must not call write endpoints such as /api/upvote|explicitly authorized|auth login' skills/jocohunt/SKILL.test.md skills/jocohunt/SKILL.md` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: docs describe stored-session submit flow
    Tool:     bash
    Steps:    mkdir -p evidence && rg -n 'auth login|auth status|auth logout|--auth-file|JOCOHUNT_SESSION_COOKIE|JOCOHUNT_CSRF_TOKEN|--confirm' README.md skills/jocohunt/SKILL.md | tee evidence/task-6-jocohunt-auth-cli-docs.txt
    Expected: command exits 0 and output includes all auth and submit safety terms
    Evidence: evidence/task-6-jocohunt-auth-cli-docs.txt

  Scenario: docs do not promise impossible callback capture
    Tool:     bash
    Steps:    mkdir -p evidence && if rg -n 'automatically capture|browser cookie scraping|GitHub password|device flow completes' README.md skills/jocohunt; then exit 1; else echo 'no impossible auth claims'; fi | tee evidence/task-6-jocohunt-auth-cli-docs-error.txt
    Expected: command exits 0 and output is exactly `no impossible auth claims`
    Evidence: evidence/task-6-jocohunt-auth-cli-docs-error.txt
  ```

  Commit: NO | Message: `docs(auth): document stored session workflow` | Files: [`README.md`, `skills/jocohunt/SKILL.md`, `skills/jocohunt/SKILL.test.md`]

- [ ] 7. Align npm, CI smoke coverage, and shipped binaries

  What to do: Update package smoke coverage so npm users can reach the auth and submit help surfaces without performing writes. Prefer changing `package.json` `smoke` to run top-level help, `auth login --help`, `auth status --help`, `auth logout --help`, and `submit --help`; update `.github/workflows/ci.yml` to call `npm run smoke`. If release workflow has a smoke step, keep it aligned. Rebuild npm binaries after Go changes with `npm run build:binaries`, then verify `node npm/bin.js auth status --auth-file <missing>` or the correct global ordering equivalent works without accessing real credentials.
  Must NOT do: Do not add live submit to CI. Do not add CI secrets. Do not publish to npm or create tags.

  Parallelization: Can parallel: NO | Wave 3 | Blocks: [8] | Blocked by: [4, 5, 6]

  References (executor has NO interview context - be exhaustive):
  - Package:  `package.json:22` - scripts block.
  - Package:  `package.json:25` - current smoke checks only top-level help.
  - Wrapper:  `npm/bin.js:5` - npm wrapper forwards all CLI args.
  - Wrapper:  `npm/bin.js:14` - npm wrapper propagates exit status.
  - Test:     `npm/wrapper.test.js:8` - existing Node `node:test` style.
  - CI:       `.github/workflows/ci.yml:23` - Go test step.
  - CI:       `.github/workflows/ci.yml:25` - Node test step.
  - CI:       `.github/workflows/ci.yml:30` - current CLI smoke step.
  - Build:    `package.json:23` - `build:binaries` script.
  - Docs:     `README.md:63` - npm binary distribution section.

  Acceptance criteria (agent-executable only):
  - [ ] `npm test | tee evidence/task-7-jocohunt-auth-npm-test.txt` exits 0.
  - [ ] `npm run build:binaries | tee evidence/task-7-jocohunt-auth-build-binaries.txt` exits 0.
  - [ ] `npm run smoke | tee evidence/task-7-jocohunt-auth-smoke.txt` exits 0 and output includes top-level, auth, and submit help text.
  - [ ] `node npm/bin.js --auth-file "$(mktemp -u)/session.json" auth status | tee evidence/task-7-jocohunt-auth-wrapper-status.txt` exits 0 and prints `Not logged in`.
  - [ ] `npm pack --dry-run | tee evidence/task-7-jocohunt-auth-pack-dry-run.txt` exits 0 and lists `npm/bin.js`, platform binaries, `README.md`, and `LICENSE`.
  - [ ] `awk 'NF && $1 !~ /^\\/\\// {n++} END {exit n>250}' internal/cli/*.go internal/jocohunt/*.go` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: npm smoke reaches auth and submit help
    Tool:     bash
    Steps:    mkdir -p evidence && npm run smoke | tee evidence/task-7-jocohunt-auth-cli-npm-smoke.txt
    Expected: command exits 0 and output contains `auth`, `login`, `logout`, `submit`, `--session-cookie`, and `--confirm`
    Evidence: evidence/task-7-jocohunt-auth-cli-npm-smoke.txt

  Scenario: packaged wrapper reports no saved session without secrets
    Tool:     bash
    Steps:    mkdir -p evidence && node npm/bin.js --auth-file "$(mktemp -u)/session.json" auth status | tee evidence/task-7-jocohunt-auth-cli-wrapper-status.txt
    Expected: command exits 0, output contains `Not logged in`, and output contains no `better-auth.session_token`
    Evidence: evidence/task-7-jocohunt-auth-cli-wrapper-status.txt
  ```

  Commit: NO | Message: `build(auth): refresh npm auth smoke coverage` | Files: [`package.json`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `npm/jocohunt`, `npm/jocohunt-darwin-amd64`, `npm/jocohunt-darwin-arm64`, `npm/jocohunt-linux-amd64`, `npm/jocohunt-linux-arm64`, `npm/jocohunt-windows-amd64.exe`]

- [ ] 8. Capture non-mutating tmux/HTTP auth evidence

  What to do: Run the full automated suite and capture real non-mutating QA evidence. Include at least three tmux/HTTP scenarios: live OAuth URL initiation with `auth login --no-open`, live loopback callback rejection via curl, and local stored-session submit dry-run without cookie flags. Use fake cookie values only. If live evidence contradicts docs, update docs in this task and rerun Task 6 checks. This task is evidence-producing and may touch docs only if live behavior requires correction.
  Must NOT do: Do not perform a production submit, do not pass a real cookie, do not click through GitHub login, and do not store real secrets in evidence.

  Parallelization: Can parallel: NO | Wave 3 | Blocks: [final] | Blocked by: [7]

  References (executor has NO interview context - be exhaustive):
  - CLI:      `cmd/jocohunt/main.go:13` - binary entrypoint for manual CLI smoke.
  - CLI:      `internal/cli/auth.go:34` - auth login flags.
  - CLI:      `internal/cli/submit.go:60` - dry-run path must be no-network.
  - CLI:      `internal/cli/submit.go:63` - live submit `--confirm` guard.
  - Docs:     `README.md:28` - user-facing submit/auth section to adjust only if QA contradicts it.
  - CI:       `.github/workflows/ci.yml:23` - full Go test shape.
  - External: `https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps` - GitHub OAuth flows and CLI/device-flow context.
  - External: `https://pkg.go.dev/net/http#Client` - Go HTTP client redirect behavior.

  Acceptance criteria (agent-executable only):
  - [ ] `go test -race -shuffle=on -count=1 ./... | tee evidence/task-8-jocohunt-auth-full-go.txt` exits 0.
  - [ ] `npm test | tee evidence/task-8-jocohunt-auth-npm-test.txt` exits 0.
  - [ ] `npm run smoke | tee evidence/task-8-jocohunt-auth-npm-smoke.txt` exits 0.
  - [ ] HTTP evidence for live normal callback contains HTTP 200 and a GitHub OAuth URL.
  - [ ] HTTP evidence for live loopback callback contains HTTP 403 and `INVALID_CALLBACKURL`.
  - [ ] tmux evidence proves `submit --dry-run` succeeds without a cookie flag after `auth login --session-cookie` stores a fake session in a temp auth file.
  - [ ] `if rg -n 'better-auth\\.session_token=[A-Za-z0-9_-]{6,}|csrf-[A-Za-z0-9_-]{4,}' evidence README.md skills/jocohunt; then exit 1; else echo 'no raw secrets found'; fi | tee evidence/task-8-jocohunt-auth-secret-scan.txt` exits 0.

  QA scenarios (MANDATORY - task incomplete without these):
  ```
  Scenario: live JoCoHunt OAuth URL initiation works without saving a session
    Tool:     tmux
    Steps:    mkdir -p evidence && tmux new-session -d -s jocohunt-auth-login 'cd /Users/yeongyu/local-workspaces/jocohunt && go run ./cmd/jocohunt auth login --no-open > evidence/task-8-jocohunt-auth-cli-tmux-login.txt 2>&1'; sleep 8; tmux has-session -t jocohunt-auth-login 2>/dev/null && tmux kill-session -t jocohunt-auth-login || true; cat evidence/task-8-jocohunt-auth-cli-tmux-login.txt
    Expected: command exits 0, evidence contains `https://github.com/login/oauth/authorize`, contains `not saved`, and contains no raw session cookie
    Evidence: evidence/task-8-jocohunt-auth-cli-tmux-login.txt

  Scenario: live JoCoHunt rejects loopback callback URLs
    Tool:     curl
    Steps:    mkdir -p evidence && curl -sS -D - -X POST https://jocohunt.jocoding.io/api/auth/sign-in/social -H 'Content-Type: application/json' -H 'Accept: application/json' -H 'Origin: https://jocohunt.jocoding.io' -d '{"provider":"github","callbackURL":"http://127.0.0.1:43119/callback"}' | tee evidence/task-8-jocohunt-auth-cli-loopback-rejected.txt
    Expected: output contains `HTTP/2 403` or `HTTP/3 403` and JSON containing `INVALID_CALLBACKURL`
    Evidence: evidence/task-8-jocohunt-auth-cli-loopback-rejected.txt

  Scenario: normal live callback request returns GitHub OAuth URL
    Tool:     curl
    Steps:    mkdir -p evidence && curl -sS -D - -X POST https://jocohunt.jocoding.io/api/auth/sign-in/social -H 'Content-Type: application/json' -H 'Accept: application/json' -H 'Origin: https://jocohunt.jocoding.io' -d '{"provider":"github","callbackURL":"/submit"}' | tee evidence/task-8-jocohunt-auth-cli-live-oauth-url.txt
    Expected: output contains `HTTP/2 200` or `HTTP/3 200`, `github.com/login/oauth/authorize`, and `redirect_uri=https%3A%2F%2Fjocohunt.jocoding.io%2Fapi%2Fauth%2Fcallback%2Fgithub`
    Evidence: evidence/task-8-jocohunt-auth-cli-live-oauth-url.txt

  Scenario: stored fake session enables submit dry-run without cookie flags
    Tool:     tmux
    Steps:    mkdir -p evidence && AUTH_FILE="$(mktemp -u)" && tmux new-session -d -s jocohunt-auth-submit "cd /Users/yeongyu/local-workspaces/jocohunt && go run ./cmd/jocohunt --auth-file '$AUTH_FILE' auth login --session-cookie 'better-auth.session_token=...' --csrf-token 'csrf-...' && go run ./cmd/jocohunt --auth-file '$AUTH_FILE' auth status && go run ./cmd/jocohunt --auth-file '$AUTH_FILE' submit --title 'QA Dry Run' --url 'https://example.com' --tagline 'No write' --dry-run > evidence/task-8-jocohunt-auth-cli-tmux-submit-dry-run.txt 2>&1"; sleep 8; tmux has-session -t jocohunt-auth-submit 2>/dev/null && tmux kill-session -t jocohunt-auth-submit || true; cat evidence/task-8-jocohunt-auth-cli-tmux-submit-dry-run.txt
    Expected: output contains `QA Dry Run`, `"sessionCookie": true`, and no production write was performed because `--dry-run` was used
    Evidence: evidence/task-8-jocohunt-auth-cli-tmux-submit-dry-run.txt
  ```

  Commit: NO | Message: `test(auth): capture non-mutating auth evidence` | Files: [`README.md`, `skills/jocohunt/SKILL.md`, `evidence/`]

## Final verification wave (MANDATORY - after all implementation tasks)
> Runs in PARALLEL. ALL must APPROVE. Surface results to the caller and wait for an explicit "okay" before declaring complete.
- [ ] F1. Plan compliance audit - every task done, every acceptance criterion met, every RED command captured before GREEN for code tasks
- [ ] F2. Code quality review - diagnostics clean, idioms match, no dead code, no auth secrets logged, edited Go files under 250 nonblank non-comment lines
- [ ] F3. Real manual QA - every QA scenario executed with evidence captured, including the tmux and HTTP scenarios in Task 8
- [ ] F4. Scope fidelity - no browser cookie scraping, no password collection, no device-flow token exchange, no production submit, and no Must-NOT-Have introduced

## Commit strategy
- The user requested no git commit, and this workspace reports `fatal: not a git repository`; every task above is `Commit: NO`.
- If a later user explicitly asks for commits inside a real git worktree, use one logical Conventional Commit per completed task (`<type>(<scope>): <subject>` body + footer).
- Atomic: every optional future commit builds and passes tests on its own.
- No "WIP" / "fix typo squash later" commits on the final branch - clean up before merge.
- Reference the plan file path in any future final commit footer: `Plan: plans/jocohunt-auth-cli.md`.

## Success criteria
- All Must-Have shipped; all current RED auth tests and full suites pass; all QA scenarios pass with captured evidence; F1-F4 approved; no production write or secret leakage occurs; no commit is created unless the user later explicitly authorizes it.
