#!/usr/bin/env bash
set -euo pipefail

mkdir -p npm

targets=(
  "darwin arm64"
  "darwin amd64"
  "linux arm"
  "linux arm64"
  "linux amd64"
  "windows arm64"
  "windows amd64"
)

for target in "${targets[@]}"; do
  read -r goos goarch <<<"${target}"
  output="npm/jocohunt-${goos}-${goarch}"
  if [[ "${goos}" == "windows" ]]; then
    output="${output}.exe"
  fi
  env_args=(GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0)
  if [[ "${goos}" == "linux" && "${goarch}" == "arm" ]]; then
    env_args+=(GOARM=7)
  fi
  env "${env_args[@]}" go build -trimpath -ldflags="-s -w" -o "${output}" ./cmd/jocohunt
done

go build -trimpath -o npm/jocohunt ./cmd/jocohunt
