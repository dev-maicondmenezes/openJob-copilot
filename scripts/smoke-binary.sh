#!/usr/bin/env bash
# Smoke test of a built openjob binary (CP11 §6). Used by CI on Linux/macOS/
# Windows (git-bash) and runnable locally:
#
#   scripts/smoke-binary.sh bin/openjob [expected-version]
#
# Verifies, on a machine with no Docker and no Go toolchain required:
#   - `version` works and (optionally) matches the expected version
#   - `serve` boots and /api/health answers
#   - first run creates the data dir + master key with restrictive permissions
#   - no second config file is created next to the DB
#   - restart reuses the SAME master key (never regenerates)
#   - SIGTERM shuts down gracefully (non-Windows)
#
# OPENJOB_SMOKE_RUN_PREFIX lets CI run a foreign-arch binary (e.g. "arch -x86_64").
set -euo pipefail

BIN=${1:?usage: smoke-binary.sh <path-to-openjob> [expected-version]}
EXPECT_VERSION=${2:-}
RUN=${OPENJOB_SMOKE_RUN_PREFIX:-}
WORK=$(mktemp -d)
PID=""
IS_WINDOWS=0
case "$(uname -s)" in MINGW*|MSYS*|CYGWIN*) IS_WINDOWS=1 ;; esac

cleanup() {
  [ -n "$PID" ] && kill "$PID" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

fail() { echo "SMOKE FAIL: $*" >&2; [ -f "$WORK/log" ] && { echo "--- server log ---" >&2; cat "$WORK/log" >&2; }; exit 1; }
ok()   { echo "ok - $*"; }

mode_of() { # octal permission bits of a file, GNU or BSD stat
  stat -c '%a' "$1" 2>/dev/null || stat -f '%Lp' "$1"
}

start_server() {
  PORT=$((20000 + RANDOM % 20000))
  # shellcheck disable=SC2086
  $RUN "$BIN" serve --port "$PORT" --db-path "$WORK/data/openjob.db" >"$WORK/log" 2>&1 &
  PID=$!
  for _ in $(seq 1 80); do
    # shellcheck disable=SC2086
    if $RUN "$BIN" healthcheck --port "$PORT" 2>/dev/null; then return 0; fi
    kill -0 "$PID" 2>/dev/null || fail "server exited before becoming healthy"
    sleep 0.25
  done
  fail "health check never passed on port $PORT"
}

stop_server() {
  kill -TERM "$PID" 2>/dev/null || true
  local code=0
  wait "$PID" || code=$?
  PID=""
  if [ "$IS_WINDOWS" -eq 0 ]; then
    [ "$code" -eq 0 ] || fail "server exit code $code after SIGTERM (want 0)"
    grep -q "shutdown complete" "$WORK/log" || fail "no graceful shutdown line in log"
  fi
}

# --- version -----------------------------------------------------------------
# shellcheck disable=SC2086
VER_OUT=$($RUN "$BIN" version)
echo "$VER_OUT" | grep -q '^openjob ' || fail "unexpected version output: $VER_OUT"
if [ -n "$EXPECT_VERSION" ]; then
  echo "$VER_OUT" | grep -q "openjob ${EXPECT_VERSION} " || fail "version '$VER_OUT' does not match $EXPECT_VERSION"
fi
ok "version: $VER_OUT"

# --- first run ---------------------------------------------------------------
start_server
ok "server healthy on :$PORT"

grep -q "http://localhost:$PORT" "$WORK/log" || fail "startup banner does not print the local URL"
KEY="$WORK/data/.master.key"
[ -f "$KEY" ] || fail "master key not created"
[ "$(wc -c <"$KEY" | tr -d ' ')" = "32" ] || fail "master key is not 32 bytes"
ok "master key created (32 bytes)"

if [ "$IS_WINDOWS" -eq 0 ]; then
  [ "$(mode_of "$KEY")" = "600" ] || fail "master key mode is $(mode_of "$KEY"), want 600"
  [ "$(mode_of "$WORK/data")" = "700" ] || fail "data dir mode is $(mode_of "$WORK/data"), want 700"
  ok "permissions restricted (dir 700, key 600)"
fi

# Only the DB (+ SQLite WAL/SHM side files) and the key may exist: bootstrap must
# not create a second configuration file mirroring SQLite settings.
EXTRA=$(ls -A "$WORK/data" | grep -Ev '^(openjob\.db(-wal|-shm)?|\.master\.key)$' || true)
[ -z "$EXTRA" ] || fail "unexpected files in data dir: $EXTRA"
ok "no extra config files"

cp "$KEY" "$WORK/key.first"
stop_server
ok "graceful shutdown"

# --- restart: same key, same data -------------------------------------------
start_server
cmp -s "$KEY" "$WORK/key.first" || fail "master key changed after restart"
ok "restart preserved the master key"
stop_server

echo "SMOKE PASS: $BIN"
