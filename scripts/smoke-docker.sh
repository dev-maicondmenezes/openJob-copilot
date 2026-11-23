#!/usr/bin/env bash
# Smoke test of the openJob container image (CP11 §6):
#
#   scripts/smoke-docker.sh <image[:tag]> [platform]      e.g. ... openjob:ci linux/arm64
#
# Verifies: starts with no interaction and no Compose, health check passes (Docker's
# own HEALTHCHECK), runs as non-root, published port answers, bad env var fails
# fast with a clear message, and the DB + master key survive container removal
# when the volume is kept (same key, never regenerated).
set -euo pipefail

IMAGE=${1:?usage: smoke-docker.sh <image> [platform]}
PLATFORM=${2:-}
SUFFIX="$$-$RANDOM"
VOL="openjob-smoke-$SUFFIX"
NAME="openjob-smoke-$SUFFIX"
PLAT=()
[ -n "$PLATFORM" ] && PLAT=(--platform "$PLATFORM")

cleanup() {
  docker rm -f "$NAME" "$NAME-bad" >/dev/null 2>&1 || true
  docker volume rm -f "$VOL" >/dev/null 2>&1 || true
}
trap cleanup EXIT
ok()   { echo "ok - $*"; }
fail() { echo "SMOKE FAIL: $*" >&2; docker logs "$NAME" >&2 2>&1 || true; exit 1; }

# busybox only inspects the volume: the openJob image is distroless (no shell).
vol_info() {
  docker run --rm "${PLAT[@]}" -v "$VOL":/data busybox:1.37 \
    sh -c 'sha256sum /data/.master.key; stat -c "%a" /data/.master.key; ls -A /data | sort | tr "\n" " "'
}

start() {
  docker run -d --name "$NAME" "${PLAT[@]}" -v "$VOL":/data -p 127.0.0.1::8080 \
    --health-interval 2s --health-start-period 1s --health-timeout 5s \
    "$IMAGE" >/dev/null
}

wait_healthy() {
  for _ in $(seq 1 90); do
    st=$(docker inspect -f '{{.State.Health.Status}}' "$NAME" 2>/dev/null || echo missing)
    [ "$st" = "healthy" ] && return 0
    [ "$(docker inspect -f '{{.State.Running}}' "$NAME")" = "true" ] || fail "container exited before becoming healthy"
    sleep 2
  done
  fail "container never became healthy (last status: $st)"
}

# --- image metadata ----------------------------------------------------------
USER_CFG=$(docker inspect -f '{{.Config.User}}' "$IMAGE")
case "$USER_CFG" in ""|0|root|0:*|root:*) fail "image runs as root (User='$USER_CFG')" ;; esac
ok "image user is non-root ($USER_CFG)"
[ -n "$(docker inspect -f '{{.Config.Healthcheck}}' "$IMAGE" | tr -d '<nil>[] ')" ] || fail "image has no HEALTHCHECK"
ok "HEALTHCHECK defined"
for l in source version revision created licenses; do
  v=$(docker inspect -f "{{index .Config.Labels \"org.opencontainers.image.$l\"}}" "$IMAGE")
  [ -n "$v" ] || fail "missing OCI label org.opencontainers.image.$l"
done
ok "OCI labels present"

# --- start with no interaction, no Compose -----------------------------------
docker volume create "$VOL" >/dev/null
start
wait_healthy
ok "container healthy (Docker HEALTHCHECK)"

HOST_PORT=$(docker port "$NAME" 8080/tcp | head -1 | sed 's/.*://')
if command -v curl >/dev/null; then
  curl -fsS "http://127.0.0.1:$HOST_PORT/api/health" | grep -q '"status":"ok"' || fail "published port does not answer /api/health"
  ok "published port :$HOST_PORT answers /api/health"
fi

# --- persistence across removal/recreation ----------------------------------
INFO1=$(vol_info)
echo "$INFO1" | sed -n 2p | grep -qx 600 || fail "master key mode in volume is not 600: $INFO1"
echo "$INFO1" | sed -n 3p | grep -q 'openjob.db' || fail "database not in the volume: $INFO1"
ok "db + master key (600) live in the /data volume"

docker rm -f "$NAME" >/dev/null
start
wait_healthy
INFO2=$(vol_info)
[ "$(echo "$INFO1" | sed -n 1p)" = "$(echo "$INFO2" | sed -n 1p)" ] || fail "master key CHANGED after container recreation"
ok "recreated container kept the same master key"
docker rm -f "$NAME" >/dev/null

# --- env validation fails fast ----------------------------------------------
set +e
OUT=$(docker run --rm --name "$NAME-bad" "${PLAT[@]}" -e OPENJOB_PORT=not-a-port "$IMAGE" 2>&1)
CODE=$?
set -e
[ "$CODE" -eq 2 ] || fail "invalid OPENJOB_PORT should exit 2, got $CODE: $OUT"
echo "$OUT" | grep -q "invalid port" || fail "no actionable message for bad OPENJOB_PORT: $OUT"
ok "invalid env var fails fast with a clear message"

echo "SMOKE PASS: $IMAGE ${PLATFORM:+($PLATFORM)}"
