// Package netinfo discovers the IPv4 addresses a phone on the same Wi-Fi can
// actually use to reach the openjob server (CP11 §4.3). It only reads local
// interface state: it never opens ports, touches the firewall or reaches the
// internet.
package netinfo

import (
	"net"
	"net/netip"
	"sort"
	"strings"
)

// Addr is a usable LAN address and the interface it belongs to.
type Addr struct {
	Interface string
	IP        netip.Addr
}

// Iface is the minimal view of a network interface the filter needs. Keeping
// it separate from net.Interface makes the policy table-testable.
type Iface struct {
	Name     string
	Up       bool
	Loopback bool
	// PointToPoint is true for tunnels (VPNs, PPP): they are not a LAN route.
	PointToPoint bool
	Addrs        []netip.Addr
}

// virtualPrefixes are interface-name prefixes that never carry the physical
// LAN: VPN/tunnel devices and container/VM bridges. A container bridge IP
// (172.17.0.1) is private but unreachable from a phone, which is exactly the
// misleading URL the plan asks us not to print.
var virtualPrefixes = []string{
	"tun", "tap", "utun", "wg", "ppp", "ipsec", "zt", "tailscale", "ts",
	"docker", "br-", "veth", "virbr", "vboxnet", "vmnet", "vEthernet",
	"lo", "awdl", "llw", "bridge", "gif", "stf", "anpi",
}

// Discover lists the usable LAN addresses of this machine, best candidates
// first (stable order). The result may be empty (no network): callers then
// show only the localhost URL.
func Discover() []Addr {
	return Filter(systemInterfaces())
}

// Filter applies the CP11 policy: up, non-loopback, non-tunnel, non-virtual
// interfaces; IPv4 only; private (RFC 1918) addresses only; never link-local.
func Filter(ifaces []Iface) []Addr {
	var out []Addr
	for _, ifc := range ifaces {
		if !ifc.Up || ifc.Loopback || ifc.PointToPoint || isVirtual(ifc.Name) {
			continue
		}
		for _, a := range ifc.Addrs {
			a = a.Unmap()
			if !a.Is4() || a.IsLoopback() || a.IsLinkLocalUnicast() || !a.IsPrivate() {
				continue
			}
			out = append(out, Addr{Interface: ifc.Name, IP: a})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i]) < rank(out[j]) })
	return out
}

// rank prefers the common home/office ranges (192.168/16) over 10/8 and
// 172.16/12, which VPN clients and container engines tend to hijack.
func rank(a Addr) int {
	b := a.IP.As4()
	switch {
	case b[0] == 192 && b[1] == 168:
		return 0
	case b[0] == 10:
		return 1
	default:
		return 2
	}
}

func isVirtual(name string) bool {
	l := strings.ToLower(name)
	for _, p := range virtualPrefixes {
		if strings.HasPrefix(l, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func systemInterfaces() []Iface {
	nis, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := make([]Iface, 0, len(nis))
	for _, ni := range nis {
		ifc := Iface{
			Name:         ni.Name,
			Up:           ni.Flags&net.FlagUp != 0,
			Loopback:     ni.Flags&net.FlagLoopback != 0,
			PointToPoint: ni.Flags&net.FlagPointToPoint != 0,
		}
		addrs, err := ni.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				if ip, ok := netip.AddrFromSlice(ipn.IP); ok {
					ifc.Addrs = append(ifc.Addrs, ip)
				}
			}
		}
		out = append(out, ifc)
	}
	return out
}
