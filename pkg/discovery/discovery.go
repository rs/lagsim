package discovery

import (
	"net"
	"os/exec"
	"strings"
)

type Client struct {
	IP    string
	MAC   string
	State string
}

// DiscoverClients parses `ip neigh show dev <iface>` to find known LAN clients.
func DiscoverClients(iface string) ([]Client, error) {
	cmd := exec.Command("/usr/sbin/ip", "neigh", "show", "dev", iface)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var clients []Client
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: 10.0.30.93 lladdr 6a:84:05:b2:4b:80 REACHABLE
		// or:     10.0.30.93 FAILED
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		c := Client{IP: fields[0]}
		state := fields[len(fields)-1]
		c.State = state

		// Skip FAILED entries
		if state == "FAILED" {
			continue
		}

		// Skip non-IPv4 entries
		if parsed := net.ParseIP(c.IP); parsed == nil || parsed.To4() == nil {
			continue
		}

		// Find MAC address (after "lladdr")
		for i, f := range fields {
			if f == "lladdr" && i+1 < len(fields) {
				c.MAC = fields[i+1]
				break
			}
		}

		clients = append(clients, c)
	}

	return clients, nil
}
