# jocohunt-cli

`jocohunt-cli`는 [조코헌트](https://jocohunt.jocoding.io/) 제품 목록 조회와 제품 등록을 터미널에서 처리하는 npm 배포 Go CLI입니다. **당신의 아이디어를 에이전트가 알리게 하세요.**

- 저장소: <https://github.com/code-yeongyu/joco>
- npm: <https://www.npmjs.com/package/jocohunt-cli>

## 설치

가장 빠른 설치 경로는 npm입니다. 패키지는 현재 플랫폼에 맞는 Go 바이너리를 포함하고, 전역 설치 후 `jocohunt-cli` 명령을 제공합니다.

### 1) npm으로 설치

```bash
npm install -g jocohunt-cli
jocohunt-cli products --limit 5
```

기존 스크립트 호환을 위해 `jocohunt` 별칭도 함께 제공합니다.

### 2) `go install`로 바로 설치

```bash
go install github.com/yeongyu/jocohunt/cmd/jocohunt@latest
jocohunt products --limit 5
```

### 3) 소스를 클론해 빌드

```bash
git clone https://github.com/code-yeongyu/joco.git
cd joco
go build -o jocohunt ./cmd/jocohunt
./jocohunt products --limit 5
```

> Go 1.24+ 가 필요합니다. 빌드한 바이너리를 `PATH`에 두면 어디서든 `jocohunt`로 실행할 수 있습니다.

## 사용법

```bash
jocohunt-cli products --limit 10
jocohunt-cli submit --title "내 제품" --url "https://example.com" --tagline "한 줄 소개" --dry-run

# 기존 별칭도 동작합니다.
jocohunt products --limit 10
jocohunt products --category ai-tools --json
jocohunt ideas --tab recent
jocohunt leaderboard --period weekly
jocohunt inspect
jocohunt auth login --print-url
```

## 로그인

CLI에서 GitHub OAuth URL을 만들고, 가능하면 **브라우저에서 세션 쿠키를 자동으로 캡처**해 저장합니다.

```bash
jocohunt auth login
```

브라우저를 띄우지 않고 URL만 출력하려면:

```bash
jocohunt auth login --print-url
```

브라우저 자동 캡처가 어려운 환경에서는 URL을 열고, `--session-cookie`로 세션을 수동 저장할 수 있습니다. 저장 파일은 기본적으로 사용자 config 디렉터리의 `jocohunt/session.json`이며 권한은 `0600`입니다.

```bash
jocohunt auth login --session-cookie 'better-auth.session_token=...' --csrf-token '...'
jocohunt auth status
jocohunt auth logout
```

헤드리스로 실행하려면:

```bash
jocohunt auth login --headless
```

특정 Chrome/Chromium 경로를 쓰려면:

```bash
jocohunt auth login --browser "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
```

저장된 세션은 저장 당시의 `--base-url` 오리진에만 재사용합니다. 다른 `--base-url`로 요청을 보낼 때는 저장된 세션을 자동으로 사용하지 않습니다(쿠키 유출 방지).

테스트나 자동화에서는 `JOCOHUNT_CONFIG_DIR`로 저장 위치를 바꿀 수 있습니다.

## 제품 등록

`auth login --session-cookie`로 세션을 저장해두면 CLI에서 바로 제품을 등록할 수 있습니다.

```bash
jocohunt submit \
  --title "내 제품" \
  --url "https://example.com" \
  --tagline "한 줄 소개" \
  --description "조금 더 긴 설명" \
  --category "ai-tools" \
  --confirm
```

등록 전 요청 본문만 확인하려면 `--dry-run`을 사용합니다.

```bash
jocohunt submit --title "내 제품" --url "https://example.com" --tagline "한 줄 소개" --dry-run
```

세션 쿠키를 플래그로 직접 넘길 수도 있습니다.

```bash
jocohunt submit --title "내 제품" --url "https://example.com" --tagline "한 줄 소개" \
  --session-cookie 'better-auth.session_token=...' --confirm
```

조코헌트 API 경로가 바뀌면 `--submit-endpoint`로 바꿔 찌를 수 있습니다. 기본값은 `/api/submit`입니다. 서버가 CSRF 토큰을 요구하는 배포에서는 `--csrf-token` 또는 `JOCOHUNT_CSRF_TOKEN`을 같이 넣으세요.

보안을 위해 `--submit-endpoint`는 **상대 경로만 허용**합니다(예: `/api/submit`). 절대 URL이나 `//host/path` 형태는 거부합니다.

## 보안 범위

조회 명령은 공개 HTML과 JSON-LD만 읽습니다. `auth login`은 GitHub OAuth URL을 만들고, 가능하면 브라우저에서 세션 쿠키를 캡처해 로컬에 저장합니다. `submit`/`upload` 명령만 로그인 세션 쿠키를 사용해 `/api/submit`에 POST합니다. 라이브 등록에는 실수 방지를 위해 `--confirm`이 필요합니다.

## npm 바이너리 배포

릴리스 전에 플랫폼별 Go 바이너리를 생성합니다.

```bash
npm run build:binaries
npm test
npm pack --dry-run
```

`npm/bin.js`는 Node의 플랫폼명(`win32`, `x64` 등)을 Go 대상명(`windows`, `amd64` 등)으로 변환한 뒤 `npm/jocohunt-<goos>-<goarch>` 바이너리를 실행합니다. 플랫폼 바이너리가 없으면 로컬 개발용 `npm/jocohunt` fallback을 사용합니다. npm 패키지는 `jocohunt-cli` 명령을 기본으로 노출하고, 기존 `jocohunt` 명령도 별칭으로 유지합니다.

패키지에 포함되는 플랫폼 바이너리:

- macOS: arm64, x64
- Linux: armv7, arm64, x64
- Windows: arm64, x64

로컬에서 직접 배포할 때는 npm 로그인 상태를 확인한 뒤 아래 순서로 배포합니다.

```bash
npm whoami
npm publish
```

## CI

GitHub Actions는 다음을 검사합니다.

- `go test -race -shuffle=on -count=1 ./...`
- `npm test`
- `npm pack --dry-run`
- 플랫폼별 바이너리 빌드
- 실제 사이트에 대한 읽기 전용 smoke test

## 릴리스

1. 버전을 올립니다.
2. `npm run build:binaries`를 실행합니다.
3. 태그를 푸시합니다.
4. `release.yml`이 테스트, 바이너리 빌드, `npm pack`, OIDC 기반 `npm publish`, GitHub Release 생성을 순서대로 실행합니다.
5. npm 배포는 `jocohunt-cli` 패키지의 Trusted Publisher 설정으로 GitHub Actions OIDC를 사용합니다.

릴리스 워크플로는 태그 `v*`에서만 동작합니다. 배포 전에 로컬에서 아래 명령으로 패키지 포함 파일을 확인하세요.

```bash
npm pack --dry-run
```
