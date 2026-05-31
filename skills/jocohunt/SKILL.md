---
name: jocohunt
description: Use this skill whenever the user asks to inspect JoCoHunt, 조코헌트, jocohunt.jocoding.io products, ideas, rankings, makers, public security headers, or wants a CLI workflow for the Korean builder community. It runs the local `jocohunt` Go CLI, including authenticated product submission when explicitly requested.
---

# JoCoHunt

Use the local `jocohunt` CLI for JoCoHunt public reads and explicitly authorized product submission.

## Commands

```bash
jocohunt products --limit 10
jocohunt products --category ai-tools --json
jocohunt ideas --tab recent
jocohunt leaderboard --period weekly
jocohunt inspect
jocohunt auth login --print-url
jocohunt auth login --session-cookie "better-auth.session_token=..."
jocohunt auth status
jocohunt submit --title "My Product" --url "https://example.com" --tagline "One-line pitch" --confirm
```

## Safety

- Only read public pages and public JSON-LD.
- Do not call write endpoints such as `/api/upvote` unless the user explicitly requested that exact write action.
- Product submission is allowed only through `jocohunt submit`/`jocohunt upload` with `--confirm` and an authenticated session cookie from `jocohunt auth login --session-cookie`, `--session-cookie`, or `JOCOHUNT_SESSION_COOKIE`.
- `jocohunt auth login` may open the GitHub OAuth URL or print it; do not collect GitHub passwords.
- Do not automate signup, login, voting, or comments unless the user separately provides explicit authorization and the task genuinely requires it.
- If the user asks for "security", report public headers and public routes. Do not probe private endpoints or brute force parameters.

## Output

Summarize the command used, the key rows returned, and whether the data came from the live site or a user-provided base URL.
