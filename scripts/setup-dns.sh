#!/usr/bin/env bash
# openJob — local DNS setup (Checkpoint 1).
#
# Two supported paths (docs/context/spec.md §5):
#   1. mDNS sidecar in docker-compose.yml (default; native on Linux) —
#      no host configuration needed beyond Docker.
#   2. dnsmasq on the host (fallback for Mac/Windows or stricter networks) —
#      answers "openjob.local" with your LAN IP on your router's DHCP domain.
#
# Usage:
#   ./scripts/setup-dns.sh            # check the mDNS path
#   ./scripts/setup-dns.sh dnsmasq    # print dnsmasq host configuration
set -euo pipefail

HOSTNAME="${OPENJOB_HOSTNAME:-openjob.local}"
IP="${OPENJOB_LAN_IP:-}"

detect_lan_ip() {
  if command -v ip >/dev/null 2>&1; then
    ip -4 addr show scope global | awk '/inet/ {split($2,a,"/"); print a[1]; exit}'
  elif command -v ifconfig >/dev/null 2>&1; then
    ifconfig | awk '/inet / && $2 !~ /^127/ {print $2; exit}' | tr -d "addr:"
  fi
}

check_mdns() {
  echo "== mDNS path (docker-compose sidecar) =="
  if ! command -v docker >/dev/null 2>&1; then
    echo "docker not found in PATH — install Docker first."
    exit 1
  fi
  [ -z "$IP" ] && IP="$(detect_lan_ip || true)"
  echo "LAN IP detected: ${IP:-<unknown>}"
  echo
  echo "1) Create a .env file next to docker-compose.yml with:"
  echo "     OPENJOB_HOSTNAME=${HOSTNAME}"
  echo "     OPENJOB_LAN_IP=${IP:-<your-lan-ip>}"
  echo
  echo "2) Bring the stack up:    docker compose up -d"
  echo "3) From the phone, open:  http://${HOSTNAME}:8080"
  echo
  if command -v avahi-resolve >/dev/null 2>&1; then
    echo "Quick self-test from the host:"
    echo "     avahi-resolve -n ${HOSTNAME}"
  fi
}

print_dnsmasq() {
  [ -z "$IP" ] && IP="$(detect_lan_ip || true)"
  echo "== dnsmasq path (host fallback) =="
  echo "Install dnsmasq, then create /etc/dnsmasq.d/openjob.conf with:"
  echo
  cat <<CONF
  # openJob local record
  host-record=${HOSTNAME},${IP:-<your-lan-ip>}
  cache-size=1000
CONF
  echo
  echo "Then restart dnsmasq (sudo systemctl restart dnsmasq) and point your"
  echo "router's DHCP DNS (or the devices' DNS) at this host."
  echo "Note: avoid using the bare TLD .local for dnsmasq on macOS — Bonjour"
  echo "already owns .local there, which is why the mDNS sidecar is preferred."
}

case "${1:-mdns}" in
  mdns)    check_mdns ;;
  dnsmasq) print_dnsmasq ;;
  *) echo "usage: $0 [mdns|dnsmasq]"; exit 2 ;;
esac
