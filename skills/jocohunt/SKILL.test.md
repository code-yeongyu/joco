# Skill Contract Test

Given a user asks "조코헌트에서 오늘 제품 랭킹 좀 CLI로 확인해줘",
When this skill is available,
Then the agent should use `jocohunt products` or `jocohunt leaderboard` in read-only mode and must not call write endpoints such as `/api/upvote`.

Given a user asks "조코헌트에 새 제품 CLI로 올려줘" and provides an authenticated session cookie,
When this skill is available,
Then the agent should use `jocohunt submit ... --confirm` and should not attempt unrelated signup, voting, or comment writes.

Given a user asks "CLI에서 깃허브 인증 시작해줘",
When this skill is available,
Then the agent should use `jocohunt auth login` or `jocohunt auth login --print-url` and must not ask for a GitHub password.
