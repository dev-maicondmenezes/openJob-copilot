#!/usr/bin/env bash
# Smoke test of the openjob .deb lifecycle (CP11 §5.2/§6). Meant to run as root
# inside a clean Debian/Ubuntu container (no Go, no Docker), e.g. from CI:
#
#   docker run --rm --platform linux/arm64 -v "$PWD/dist:/dist:ro" -v "$PWD/scripts:/scripts:ro" \
#       debian:12-slim bash /scripts/smoke-deb.sh /dist/openjob_X.Y.Z_arm64.deb
#
# Verifies: package contents (no service/scripts), clean install, first run as a
# regular user (per-user data dir, key 600), reinstall/upgrade preserves DB + key,
# and default removal AND purge preserve user data.
set -euo pipefail

DEB=${1:?usage: smoke-deb.sh <path-to.deb>}
ok()   { echo "ok - $*"; }
fail() { echo "SMOKE FAIL: $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || fail "run as root (inside a throwaway container)"

# --- package contents --------------------------------------------------------
dpkg-deb -f "$DEB" Package | grep -qx openjob || fail "Package field is not 'openjob'"
CONTENTS=$(dpkg-deb -c "$DEB")
echo "$CONTENTS" | grep -q ' ./usr/bin/openjob$' || fail "binary not installed at /usr/bin/openjob"
echo "$CONTENTS" | grep -q 'usr/share/applications/openjob.desktop' || fail "desktop entry missing"
echo "$CONTENTS" | grep -Eq 'systemd|init\.d|cron|sudoers' && fail "package ships a service/cron/sudoers file (out of scope)"
SCRIPTS=$(dpkg-deb -I "$DEB" | grep -E '^\s+(preinst|postinst|prerm|postrm)' || true)
[ -z "$SCRIPTS" ] || fail "package has maintainer scripts (must not start servers or touch data): $SCRIPTS"
ok "package contents: binary + desktop entry, no service, no maintainer scripts"

# --- clean install -----------------------------------------------------------
dpkg -i "$DEB" >/dev/null
[ -x /usr/bin/openjob ] || fail "binary missing after install"
openjob version | grep -q '^openjob ' || fail "installed binary does not run"
ok "clean install: $(openjob version)"
for c in /proc/[0-9]*/comm; do
  grep -qx openjob "$c" 2>/dev/null && fail "install started a server automatically"
done
ok "install did not start a server"

# --- first run as a regular user --------------------------------------------
id tester >/dev/null 2>&1 || useradd -m tester
DATA=/home/tester/.local/share/openjob
PORT=$((20000 + RANDOM % 20000))

run_server() {
  # setpriv exec()s the command, so $! is the openjob PID itself and SIGTERM
  # reaches it directly (runuser would stay in between as a parent process).
  setpriv --reuid=tester --regid=tester --init-groups \
    env -i HOME=/home/tester PATH=/usr/bin:/bin \
    /usr/bin/openjob serve --port "$PORT" >/tmp/openjob.log 2>&1 &
  SRV=$!
  for _ in $(seq 1 80); do
    setpriv --reuid=tester --regid=tester --init-groups /usr/bin/openjob healthcheck --port "$PORT" 2>/dev/null && return 0
    kill -0 "$SRV" 2>/dev/null || { cat /tmp/openjob.log >&2; fail "server exited early"; }
    sleep 0.25
  done
  cat /tmp/openjob.log >&2; fail "never healthy"
}
stop_server() { kill -TERM "$SRV"; wait "$SRV" || fail "unclean exit"; }

run_server
[ -f "$DATA/.master.key" ] || fail "no master key in the per-user dir $DATA"
[ "$(stat -c %a "$DATA/.master.key")" = "600" ] || fail "key mode not 600"
[ "$(stat -c %a "$DATA")" = "700" ] || fail "data dir mode not 700"
sha256sum "$DATA/.master.key" >/tmp/key.sum
ok "first run: per-user data dir created (dir 700, key 600)"
stop_server

# --- upgrade (reinstall over the top) preserves data ------------------------
dpkg -i "$DEB" >/dev/null
sha256sum -c /tmp/key.sum >/dev/null || fail "upgrade changed the master key"
[ -f "$DATA/openjob.db" ] || fail "upgrade lost the database"
run_server; stop_server
sha256sum -c /tmp/key.sum >/dev/null || fail "key changed after post-upgrade start"
ok "upgrade preserved database and master key"

# --- removal and purge preserve user data -----------------------------------
dpkg -r openjob >/dev/null
[ ! -e /usr/bin/openjob ] || fail "binary still present after removal"
[ -f "$DATA/openjob.db" ] && [ -f "$DATA/.master.key" ] || fail "default removal deleted user data"
ok "default removal preserved user data"

dpkg -i "$DEB" >/dev/null
dpkg -P openjob >/dev/null
sha256sum -c /tmp/key.sum >/dev/null || fail "purge deleted user data"
ok "purge preserved user data (remove $DATA manually if wanted)"

echo "SMOKE PASS: $DEB"
