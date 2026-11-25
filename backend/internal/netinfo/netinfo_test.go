package netinfo

import (
	"net/netip"
	"testing"
)

func ip(s string) netip.Addr { return netip.MustParseAddr(s) }

func TestFilterPolicy(t *testing.T) {
	ifaces := []Iface{
		{Name: "lo", Up: true, Loopback: true, Addrs: []netip.Addr{ip("127.0.0.1")}},
		{Name: "wlp3s0", Up: true, Addrs: []netip.Addr{ip("192.168.15.20"), ip("fe80::1"), ip("2804:14d::1")}},
		{Name: "enp2s0", Up: true, Addrs: []netip.Addr{ip("10.0.0.5")}},
		{Name: "eth1", Up: false, Addrs: []netip.Addr{ip("192.168.99.9")}},                // down
		{Name: "tun0", Up: true, Addrs: []netip.Addr{ip("10.8.0.2")}},                     // VPN by name
		{Name: "ppp0", Up: true, PointToPoint: true, Addrs: []netip.Addr{ip("10.1.1.1")}}, // tunnel by flag
		{Name: "docker0", Up: true, Addrs: []netip.Addr{ip("172.17.0.1")}},                // container bridge
		{Name: "br-1a2b3c", Up: true, Addrs: []netip.Addr{ip("172.18.0.1")}},              // compose bridge
		{Name: "wg0", Up: true, Addrs: []netip.Addr{ip("10.9.0.1")}},                      // wireguard
		{Name: "vEthernet (WSL)", Up: true, Addrs: []netip.Addr{ip("172.28.0.1")}},        // Hyper-V (case-insens.)
		{Name: "eth2", Up: true, Addrs: []netip.Addr{ip("169.254.10.10")}},                // link-local
		{Name: "eth3", Up: true, Addrs: []netip.Addr{ip("203.0.113.7")}},                  // public
		{Name: "eth4", Up: true, Addrs: []netip.Addr{ip("::ffff:192.168.1.50")}},          // v4-mapped
	}

	got := Filter(ifaces)
	want := []Addr{
		{"wlp3s0", ip("192.168.15.20")},
		{"eth4", ip("192.168.1.50")},
		{"enp2s0", ip("10.0.0.5")},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestFilterEmpty(t *testing.T) {
	if got := Filter(nil); len(got) != 0 {
		t.Errorf("expected no addresses, got %v", got)
	}
}

func TestDiscoverDoesNotPanic(t *testing.T) {
	// Real interfaces vary per machine; only assert the contract: every result
	// is private IPv4.
	for _, a := range Discover() {
		if !a.IP.Is4() || !a.IP.IsPrivate() {
			t.Errorf("non-private/non-IPv4 address leaked: %v", a)
		}
	}
}
