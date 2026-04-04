package tc

import (
	"lagsim/pkg/config"
	"fmt"
)

const defaultClassMinor = "9999"

// Setup initializes the full tc infrastructure: ifb, HTB roots, ingress redirect.
func Setup(cfg *config.Config, r *Runner) error {
	lan := cfg.Interfaces.LAN
	ifb := cfg.Interfaces.IFB
	rate := cfg.RootRate

	fmt.Println("Loading ifb module...")
	if err := r.RunModprobe("ifb", "numifbs=1"); err != nil {
		return fmt.Errorf("load ifb module: %w", err)
	}

	fmt.Printf("Bringing up %s...\n", ifb)
	if err := r.RunIP("link", "set", ifb, "up"); err != nil {
		return fmt.Errorf("bring up %s: %w", ifb, err)
	}

	// Egress on LAN interface
	fmt.Printf("Setting up egress HTB on %s...\n", lan)
	if !r.QdiscExists(lan, "htb 1:") {
		if err := r.Run("qdisc", "add", "dev", lan, "root", "handle", "1:", "htb", "default", defaultClassMinor); err != nil {
			return fmt.Errorf("add HTB root on %s: %w", lan, err)
		}
	}
	if !r.ClassExists(lan, "1:"+defaultClassMinor) {
		if err := r.Run("class", "add", "dev", lan, "parent", "1:", "classid", "1:"+defaultClassMinor, "htb", "rate", rate, "ceil", rate); err != nil {
			return fmt.Errorf("add default class on %s: %w", lan, err)
		}
	}

	// Ingress redirect to IFB
	fmt.Printf("Setting up ingress redirect %s -> %s...\n", lan, ifb)
	if !r.QdiscExists(lan, "ingress") {
		if err := r.Run("qdisc", "add", "dev", lan, "handle", "ffff:", "ingress"); err != nil {
			return fmt.Errorf("add ingress qdisc on %s: %w", lan, err)
		}
	}
	// Add mirred redirect filter (idempotent: check if ifb already has traffic)
	if !r.FilterExistsForIP(lan+"@ingress", "mirred") {
		if err := r.Run("filter", "add", "dev", lan, "parent", "ffff:", "protocol", "ip",
			"u32", "match", "u32", "0", "0",
			"action", "mirred", "egress", "redirect", "dev", ifb); err != nil {
			return fmt.Errorf("add mirred redirect on %s: %w", lan, err)
		}
	}

	// Egress on IFB (for ingress traffic)
	fmt.Printf("Setting up egress HTB on %s...\n", ifb)
	if !r.QdiscExists(ifb, "htb 1:") {
		if err := r.Run("qdisc", "add", "dev", ifb, "root", "handle", "1:", "htb", "default", defaultClassMinor); err != nil {
			return fmt.Errorf("add HTB root on %s: %w", ifb, err)
		}
	}
	if !r.ClassExists(ifb, "1:"+defaultClassMinor) {
		if err := r.Run("class", "add", "dev", ifb, "parent", "1:", "classid", "1:"+defaultClassMinor, "htb", "rate", rate, "ceil", rate); err != nil {
			return fmt.Errorf("add default class on %s: %w", ifb, err)
		}
	}

	fmt.Println("TC infrastructure ready.")
	return nil
}

// Teardown removes all tc state and brings down ifb.
func Teardown(cfg *config.Config, r *Runner) {
	lan := cfg.Interfaces.LAN
	ifb := cfg.Interfaces.IFB

	fmt.Println("Removing tc infrastructure...")
	r.RunIgnoreErr("qdisc", "del", "dev", lan, "root")
	r.RunIgnoreErr("qdisc", "del", "dev", lan, "ingress")
	r.RunIgnoreErr("qdisc", "del", "dev", ifb, "root")
	r.RunIPIgnoreErr("link", "set", ifb, "down")
	fmt.Println("Done.")
}
