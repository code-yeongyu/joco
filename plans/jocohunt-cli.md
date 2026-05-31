# jocohunt CLI

## TL;DR
> **Summary**: Build a safe read-only Go CLI for jocohunt.jocoding.io, package it for npm installs, add a Codex skill, Korean docs, and strict CI.
> **Deliverables**: Go CLI, npm wrapper, Codex skill, Korean README, GitHub Actions CI/release workflow.
> **Effort**: Medium
> **Parallel**: YES - 3 waves
> **Critical Path**: Tests -> Go CLI -> npm wrapper -> docs/CI -> QA/review

## Context
### Original Request
- Safely inspect jocohunt.jocoding.io with browser/computer-use, identify public APIs/surfaces, build a Go CLI, create a skill, Korean README, GitHub-ready packaging.
- Follow-up scope: make install/deploy polished, support npm-based Go binary installation, and add strict CI.

### Research Findings
- Workspace is greenfield and not a git repository yet.
- The site is a Next.js Korean builder community app.
- Public read-only pages verified: `/`, `/products`, `/ideas`, `/leaderboard`.
- Public page data is available in rendered HTML/JSON-LD; write endpoints such as `/api/upvote` and auth flows are out of scope.
- Security headers observed include CSP, HSTS, X-Frame-Options, Referrer-Policy, and Permissions-Policy.

## Work Objectives
### Core Objective
Deliver a verified read-only `jocohunt` CLI and npm package that can inspect public JoCoHunt listings without credentials or intrusive testing.

### Must Have
- Read-only commands: `products`, `ideas`, `leaderboard`, `inspect`.
- JSON and table output.
- Timeout/base URL flags for tests and alternate environments.
- npm bin wrapper that selects a packaged platform binary and uses a generic local-development binary as fallback.
- Korean README with install, npm, source build, usage, CI, release notes.
- Codex skill in `skills/jocohunt/SKILL.md`.
- Strict GitHub Actions: Go tests/race, npm tests, wrapper tests, cross-platform binary build.

### Must NOT Have
- No signup automation unless required. It is not required for read-only features.
- No credential scraping, private API probing, brute force, fuzzing against production, or write endpoint mutation.
- No claims that undocumented write APIs are supported.

## Verification Strategy
- Test decision: TDD with Go unit/integration-style httptest tests and Node wrapper tests.
- Manual QA:
  - HTTP: `curl -i https://jocohunt.jocoding.io/products`
  - Browser: `agent-browser open https://jocohunt.jocoding.io/ && agent-browser snapshot -i`
  - tmux: run the built CLI against the real site and capture transcript.
  - Computer use: confirm Chrome is available/running and capture app state.
- Evidence: `.omo/ulw-loop/019e7be6-5819-7371-babc-d3ffe412a141/evidence/`

## Execution Strategy
### Parallel Execution Waves
- Wave 1: Write tests and project metadata.
- Wave 2: Implement Go CLI, npm wrapper, skill, docs, CI.
- Wave 3: Run full automated verification, browser/HTTP/tmux/computer-use QA, and reviewer audit.

### Dependency Matrix
| Task | Depends on | Blocks | Can parallelize with |
| --- | --- | --- | --- |
| T1 tests | none | T2 | docs draft |
| T2 Go CLI | T1 red | T3, QA | skill draft |
| T3 npm wrapper | T2 binary contract | QA | docs/CI |
| T4 docs/skill/CI | public scope decision | QA | T2/T3 after contracts |
| T5 QA/review | T2-T4 | completion | none |

## TODOs
- [x] 1. Write RED tests for public page parsing, CLI behavior, npm wrapper, and skill contract.
- [x] 2. Implement Go read-only CLI with robust HTML/JSON-LD parsing and bounded HTTP client.
- [x] 3. Implement npm wrapper, package scripts, and binary build scripts.
- [x] 4. Add Korean README, Codex skill, CI, release workflow, and package metadata.
- [x] 5. Run automated checks plus HTTP/browser/tmux/computer-use QA and record evidence.

## Final Verification Wave
- [x] F1. LSP/diagnostics and `go test -race -shuffle=on -count=1 ./...`.
- [x] F2. `npm test` and npm wrapper smoke checks.
- [x] F3. Real site HTTP/browser/tmux/computer-use evidence.
- [x] F4. Security/scope review confirms only public read-only behavior.

## Commit Strategy
Prepare changes for a later atomic commit. Do not commit or push automatically.
