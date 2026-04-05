package tc

import (
	"lagsim/pkg/config"
	"fmt"
	"strings"
)

// ApplyProfile applies a netem profile to a client IP on a single device.
// filterField is "dst" for the LAN interface (egress to client) or "src" for ifb (ingress from client).
// direction is "download" or "upload" and selects the resolved directional profile.
func ApplyProfile(r *Runner, dev, ip string, profile *config.Profile, rootRate string, filterField string, direction string) error {
	classID, err := ClassIDFromIP(ip)
	if err != nil {
		return err
	}

	resolved := profile.Resolved(direction)
	classStr := FormatClassID(classID)
	handleStr := NetemHandle(classID)
	rate := resolved.Rate
	if rate == "" {
		rate = rootRate
	}

	// Add or change HTB class
	if r.ClassExists(dev, classStr) {
		if err := r.Run("class", "change", "dev", dev, "parent", "1:", "classid", classStr,
			"htb", "rate", rate, "ceil", rate); err != nil {
			return fmt.Errorf("change class %s on %s: %w", classStr, dev, err)
		}
	} else {
		if err := r.Run("class", "add", "dev", dev, "parent", "1:", "classid", classStr,
			"htb", "rate", rate, "ceil", rate); err != nil {
			return fmt.Errorf("add class %s on %s: %w", classStr, dev, err)
		}
	}

	// Build netem args
	netemArgs := buildNetemArgs(resolved)

	// Add or change netem qdisc
	if r.QdiscExists(dev, handleStr) {
		args := append([]string{"qdisc", "change", "dev", dev, "parent", classStr, "handle", handleStr, "netem"}, netemArgs...)
		if err := r.Run(args...); err != nil {
			return fmt.Errorf("change netem on %s: %w", dev, err)
		}
	} else {
		args := append([]string{"qdisc", "add", "dev", dev, "parent", classStr, "handle", handleStr, "netem"}, netemArgs...)
		if err := r.Run(args...); err != nil {
			return fmt.Errorf("add netem on %s: %w", dev, err)
		}
	}

	// Add u32 filter if not present
	if !r.FilterExistsForIP(dev, ip) {
		if err := r.Run("filter", "add", "dev", dev, "parent", "1:", "protocol", "ip",
			"prio", "1", "u32", "match", "ip", filterField, ip+"/32", "flowid", classStr); err != nil {
			return fmt.Errorf("add filter for %s on %s: %w", ip, dev, err)
		}
	}

	return nil
}

// RemoveClient removes all tc objects for a client IP from a single device.
func RemoveClient(r *Runner, dev, ip, filterField string) error {
	classID, err := ClassIDFromIP(ip)
	if err != nil {
		return err
	}

	classStr := FormatClassID(classID)

	// Delete filter first
	if err := deleteFilterForIP(r, dev, ip); err != nil {
		return err
	}

	// Delete netem qdisc (child of the class)
	r.RunIgnoreErr("qdisc", "del", "dev", dev, "parent", classStr)

	// Delete the class
	r.RunIgnoreErr("class", "del", "dev", dev, "parent", "1:", "classid", classStr)

	return nil
}

// deleteFilterForIP finds and removes u32 filters matching the given IP.
func deleteFilterForIP(r *Runner, dev, ip string) error {
	out, err := r.Output("filter", "show", "dev", dev)
	if err != nil {
		return nil // no filters, nothing to delete
	}

	// Parse filter output to find handles for filters matching this IP.
	// tc filter show output looks like:
	//   filter parent 1: protocol ip pref 1 u32 chain 0
	//   filter parent 1: protocol ip pref 1 u32 chain 0 fh 800::800 order 2048 key ht 800 bkt 0 flowid 1:5d not_in_hw
	//     match 0a001e5d/ffffffff at 16
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "match") {
			continue
		}
		// Check if the next or previous filter line relates to our IP
		// The IP is encoded as hex in the match line, but we can also check
		// by looking at the flowid which corresponds to our class ID
		cid, _ := ClassIDFromIP(ip)
		classStr := FormatClassID(cid)
		// Look backward for the filter handle line
		for j := i - 1; j >= 0; j-- {
			if strings.Contains(lines[j], "fh ") && strings.Contains(lines[j], "flowid "+classStr) {
				// Extract filter handle: "fh 800::800"
				handle := extractFilterHandle(lines[j])
				if handle != "" {
					r.RunIgnoreErr("filter", "del", "dev", dev, "parent", "1:", "protocol", "ip",
						"prio", "1", "handle", handle, "u32")
				}
				break
			}
		}
	}
	return nil
}

func extractFilterHandle(line string) string {
	idx := strings.Index(line, "fh ")
	if idx < 0 {
		return ""
	}
	rest := line[idx+3:]
	end := strings.IndexByte(rest, ' ')
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func buildNetemArgs(p config.DirectionalProfile) []string {
	var args []string

	if p.Delay != "" {
		args = append(args, "delay", p.Delay)
		if p.Jitter != "" {
			args = append(args, p.Jitter)
			if p.Correlation != "" {
				args = append(args, p.Correlation)
			}
		}
	}

	if p.Loss != "" {
		args = append(args, "loss", p.Loss)
	}

	if p.Duplicate != "" {
		args = append(args, "duplicate", p.Duplicate)
	}

	if p.Reorder != "" {
		args = append(args, "reorder", p.Reorder)
	}

	if p.Corrupt != "" {
		args = append(args, "corrupt", p.Corrupt)
	}

	return args
}

// ApplyToAllDevices applies a profile to both LAN (egress/download) and IFB (ingress/upload) interfaces.
func ApplyToAllDevices(r *Runner, cfg *config.Config, ip string, profile *config.Profile) error {
	fmt.Printf("Applying profile to %s (download)...\n", cfg.Interfaces.LAN)
	if err := ApplyProfile(r, cfg.Interfaces.LAN, ip, profile, cfg.RootRate, "dst", "download"); err != nil {
		return err
	}

	fmt.Printf("Applying profile to %s (upload)...\n", cfg.Interfaces.IFB)
	if err := ApplyProfile(r, cfg.Interfaces.IFB, ip, profile, cfg.RootRate, "src", "upload"); err != nil {
		return err
	}

	return nil
}

// RemoveFromAllDevices removes a client from both LAN and IFB interfaces.
func RemoveFromAllDevices(r *Runner, cfg *config.Config, ip string) error {
	fmt.Printf("Removing from %s (egress)...\n", cfg.Interfaces.LAN)
	if err := RemoveClient(r, cfg.Interfaces.LAN, ip, "dst"); err != nil {
		return err
	}

	fmt.Printf("Removing from %s (ingress)...\n", cfg.Interfaces.IFB)
	if err := RemoveClient(r, cfg.Interfaces.IFB, ip, "src"); err != nil {
		return err
	}

	return nil
}
