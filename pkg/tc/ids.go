package tc

import (
	"fmt"
	"net"
)

// ClassIDFromIP returns the last octet of the IP for use as HTB class minor ID.
// Valid range: 1-254 (excludes network and broadcast addresses).
func ClassIDFromIP(ip string) (uint16, error) {
	parsed := net.ParseIP(ip).To4()
	if parsed == nil {
		return 0, fmt.Errorf("invalid IPv4: %s", ip)
	}
	octet := uint16(parsed[3])
	if octet == 0 || octet == 255 {
		return 0, fmt.Errorf("cannot use network/broadcast address: %s", ip)
	}
	return octet, nil
}

// FormatClassID returns the tc class ID string, e.g. "1:5d".
func FormatClassID(id uint16) string {
	return fmt.Sprintf("1:%x", id)
}

// NetemHandle returns the netem qdisc handle string, e.g. "80a0:".
// Uses 0x8000 + id to avoid collision with HTB handles.
func NetemHandle(id uint16) string {
	return fmt.Sprintf("%x:", 0x8000+uint32(id))
}
