#!/usr/bin/env bash
# Visual regression tests run inside the *pinned* Playwright image so the local
# machine and CI render with the exact same Chromium and fonts. Baselines are
# canonical to that image: a Debian host and an Ubuntu runner would otherwise
# differ by anti-aliasing and even line wrapping (e.g. /settings/discovery at
# 375px), failing snapshots that are actually correct.
#
#   npm run test:visual                 # compare
#   npm run test:visual -- --update-snapshots   # regenerate baselines
#
# Docker is required (the project already runs everything in Docker). The image
# tag follows the installed @playwright/test version, so CI and local cannot
# silently drift apart.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
FRONTEND="$ROOT/frontend"
PW_BIN="$FRONTEND/node_modules/.bin/playwright"

command -v docker >/dev/null 2>&1 || {
	echo "visual-test: docker is required to run visual regression tests" >&2
	exit 1
}
[ -x "$PW_BIN" ] || {
	echo "visual-test: missing $PW_BIN (run npm install in frontend/)" >&2
	exit 1
}

VERSION=$(cd "$FRONTEND" && node -p "require('./node_modules/@playwright/test/package.json').version")
IMAGE=${OPENJOB_VISUAL_IMAGE:-mcr.microsoft.com/playwright:v${VERSION}-noble}

env_args=()
if [ -n "${CI:-}" ]; then
	env_args+=(-e CI=1)
fi

echo "visual-test: $IMAGE" >&2
docker run --rm --init --ipc=host \
	-v "$ROOT:/work" -w /work/frontend \
	--user "$(id -u):$(id -g)" -e HOME=/tmp \
	"${env_args[@]}" \
	"$IMAGE" ./node_modules/.bin/playwright test --project='visual-*' "$@"
