#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
command -v go >/dev/null || { echo 'Install Go 1.26+ on the build machine first.' >&2; exit 1; }
mkdir -p bin
build_os="${GOOS:-linux}"
build_arch="${GOARCH:-$(go env GOHOSTARCH)}"
GOOS="$(go env GOHOSTOS)" GOARCH="$(go env GOHOSTARCH)" go run ./cmd/featuregen
build_temp=$(mktemp "bin/.build.XXXXXX")
trap 'rm -f "$build_temp"' EXIT
CGO_ENABLED=0 GOOS="$build_os" GOARCH="$build_arch" go build -trimpath -ldflags='-s -w' -o "$build_temp" .
chmod 755 "$build_temp"
mv "$build_temp" "bin/discord-bot-$build_os-$build_arch"
echo "Built bin/discord-bot-$build_os-$build_arch"
