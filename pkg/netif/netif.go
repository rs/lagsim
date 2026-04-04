package netif

import (
	"fmt"
	"net"
	"strings"
)

type Interface struct {
	Name   string
	Addrs  []string // IPv4 CIDRs
	MAC    string
	IsUp   bool
	IsVlan bool
}

// ListCandidates returns network interfaces suitable for LAN conditioning.
// Excludes loopback, ifb*, and interfaces without IPv4 addresses.
func ListCandidates() ([]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}

	var result []Interface
	for _, iface := range ifaces {
		name := iface.Name

		// Skip loopback, ifb, and tun/tap
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if strings.HasPrefix(name, "ifb") {
			continue
		}
		if strings.HasPrefix(name, "tailscale") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		var ipv4Addrs []string
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipNet.IP.To4() != nil {
				ipv4Addrs = append(ipv4Addrs, ipNet.String())
			}
		}

		if len(ipv4Addrs) == 0 {
			continue
		}

		isUp := iface.Flags&net.FlagUp != 0
		isVlan := strings.Contains(name, ".")

		result = append(result, Interface{
			Name:   name,
			Addrs:  ipv4Addrs,
			MAC:    iface.HardwareAddr.String(),
			IsUp:   isUp,
			IsVlan: isVlan,
		})
	}

	return result, nil
}

// SubnetFromCIDR returns the network address in CIDR notation from an address CIDR.
// e.g. "10.0.30.1/24" -> "10.0.30.0/24"
func SubnetFromCIDR(cidr string) string {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr
	}
	return ipNet.String()
}
