#!/usr/bin/env bash
# openJob — local DNS setup (Checkpoint 1).
#
# Tested paths (docs/adr/0001-mdns-local-hostname.md):
#   1. avahi  — native Linux mDNS alias via /etc/avahi/hosts (RECOMMENDED on
#               Linux hosts/VMs; machine hostname stays untouched)
#   2. mdns   — docker-compose sidecar (Mac/Windows guests without avahi)
#   3. dnsmasq — host fallback (advanced)
#   Android/Chrome cannot resolve .local (Chromium limitation) — use
#   http://<vm-ip>.nip.io:<port> there.
#
# Usage:
#   ./scripts/setup-dns.sh            # status of every path
#   ./scripts/setup-dns.sh avahi      # configure native avahi alias (Linux)
#   ./scripts/setup-dns.sh mdns       # sidecar checklist (.env + compose)
#   ./scripts/setup-dns.sh dnsmasq    # print dnsmasq host configuration
set -euo pipefail

HOSTNAME_ALIAS="${OPENJOB_HOSTNAME:-openjob.local}"
IP="${OPENJOB_LAN_IP:-}"

detect_lan_ip() {
  if command -v ip >/dev/null 2>&1; then
    ip -4 addr show scope global | awk '/inet/ {split($2,a,"/"); print a[1]; exit}'
  elif command -v ifconfig >/dev/null 2>&1; then
    ifconfig | awk '/inet / && $2 !~ /^127/ {print $2; exit}' | tr -d "addr:"
  fi
}

check_mdns() {
  echo "== mDNS sidecar path (docker-compose) =="
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker not found in PATH — install Docker first."
    exit 1
  fi
  [ -z "$IP" ] && IP="$(detect_lan_ip || true)"
  echo "LAN IP detected: ${IP:-<unknown>}"
  echo
  echo "1) Create a .env file next to docker-compose.yml with:"
  echo "     OPENJOB_HOSTNAME=${HOSTNAME_ALIAS}"
  echo "     OPENJOB_LAN_IP=${IP:-<your-lan-ip>}"
  echo
  echo "2) Bring the stack up:    sudo docker compose up -d --force-recreate"
  echo "3) From the phone, open:  http://${HOSTNAME_ALIAS}:8080"
  echo "   (Android/Chrome: use http://${IP:-<vm-ip>}.nip.io:8080 instead)"
  echo
  echo "Note: on Linux guests with avahi-daemon, prefer './scripts/setup-dns.sh avahi'"
  echo "and keep only ONE mDNS responder per network."
}

setup_avahi() {
  echo "== native avahi alias path (Linux) =="
  if ! command -v avahi-daemon >/dev/null 2>&1 && [ ! -x /usr/sbin/avahi-daemon ]; then
    echo "avahi-daemon not found — install with: sudo apt install avahi-daemon"
    exit 1
  fi
  [ -z "$IP" ] && IP="$(detect_lan_ip || true)"
  [ -z "$IP" ] && { echo "could not detect LAN IP; export OPENJOB_LAN_IP first."; exit 1; }

  echo "Publishing ${HOSTNAME_ALIAS} -> ${IP} via /etc/avahi/hosts"
  echo
  echo "Run these commands (sudo needed):"
  echo
  echo "  echo '${IP} ${HOSTNAME_ALIAS}' | sudo tee -a /etc/avahi/hosts"
  echo "  echo '${IP} ${HOSTNAME_ALIAS}' | sudo tee -a /etc/hosts   # local resolution"
  echo "  sudo systemctl restart avahi-daemon"
  echo
  echo "Then validate:"
  echo "  getent hosts ${HOSTNAME_ALIAS}"
  echo "  curl -v http://${HOSTNAME_ALIAS}:8080/api/health"
  echo
  echo "From other devices: http://${HOSTNAME_ALIAS}:8080"
  echo "From Android/Chrome: http://${IP}.nip.io:8080"
}

print_dnsmasq() {
  [ -z "$IP" ] && IP="$(detect_lan_ip || true)"
  echo "== dnsmasq path (host fallback) =="
  echo "Install dnsmasq, then create /etc/dnsmasq.d/openjob.conf with:"
  echo
  cat <<CONF
  # openJob local record
  host-record=${HOSTNAME_ALIAS},${IP:-<your-lan-ip>}
  cache-size=1000
CONF
  echo
  echo "Then restart dnsmasq (sudo systemctl restart dnsmasq) and point your"
  echo "router's DHCP DNS (or the devices' DNS) at this host."
  echo "Note: avoid using the bare TLD .local for dnsmasq on macOS — Bonjour"
  echo "already owns .local there, which is why the mDNS sidecar is preferred."
}

status_all() {
  echo "== openJob local DNS status =="
  [ -z "$IP" ] && IP="$(detect_lan_ip || true)"
  echo "LAN IP: ${IP:-<unknown>}"
  echo
  if command -v avahi-daemon >/dev/null 2>&1 || [ -x /usr/sbin/avahi-daemon ]; then
    echo "[avahi]     installed"
    if [ -f /etc/avahi/hosts ] && grep -q "[[:space:]]${HOSTNAME_ALIAS}\$" /etc/avahi/hosts 2>/dev/null; then
      echo "[avahi]     alias configured in /etc/avahi/hosts"
    else
      echo "[avahi]     alias NOT configured (run: $0 avahi)"
    fi
  else
    echo "[avahi]     not installed"
  fi
  if grep -q "[[:space:]]${HOSTNAME_ALIAS}\$" /etc/hosts 2>/dev/null; then
    echo "[/etc/hosts] alias present (local resolution guaranteed)"
  else
    echo "[/etc/hosts] alias absent (optional: see './$0 avahi')"
  fi
  if command -v docker >/dev/null 2>&1 && sudo -n docker compose ps --status running 2>/dev/null | grep -q openjob-mdns; then
    echo "[sidecar]   openjob-mdns running"
  else
    echo "[sidecar]   not running (optional on Linux: native avahi covers it)"
  fi
  echo
  echo "Test URLs:"
  echo "  mDNS devices:   http://${HOSTNAME_ALIAS}:8080"
  echo "  Android/Chrome: http://${IP}.nip.io:8080"
}

case "${1:-status}" in
  status)  status_all ;;
  avahi)   setup_avahi ;;
  mdns)    check_mdns ;;
  dnsmasq) print_dnsmasq ;;
  *) echo "usage: $0 [status|avahi|mdns|dnsmasq]"; exit 2 ;;
esac
