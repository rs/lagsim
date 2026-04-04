package cmd

import (
	"lagsim/pkg/discovery"
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List known LAN clients and their profiles",
	RunE:  listRun,
}

func listRun(cmd *cobra.Command, args []string) error {
	if cfg.Interfaces.LAN == "" {
		return fmt.Errorf("no LAN interface configured, run 'lagsim init' first")
	}
	clients, err := discovery.DiscoverClients(cfg.Interfaces.LAN)
	if err != nil {
		return fmt.Errorf("discover clients: %w", err)
	}

	// Build a merged view: discovered clients + any assigned IPs not yet discovered
	type entry struct {
		IP      string
		Name    string
		MAC     string
		State   string
		Profile string
	}

	seen := make(map[string]bool)
	var entries []entry

	for _, c := range clients {
		profile := cfg.Assignments[c.IP]
		if profile == "" {
			profile = "(none)"
		}
		entries = append(entries, entry{
			IP:      c.IP,
			Name:    resolveListName(c.MAC, c.IP),
			MAC:     c.MAC,
			State:   c.State,
			Profile: profile,
		})
		seen[c.IP] = true
	}

	// Add assigned IPs not discovered on the network
	for ip, profile := range cfg.Assignments {
		if !seen[ip] {
			entries = append(entries, entry{
				IP:      ip,
				Name:    reverseLookup(ip),
				MAC:     "-",
				State:   "OFFLINE",
				Profile: profile,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].IP < entries[j].IP
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IP\tNAME\tMAC\tSTATE\tPROFILE")
	fmt.Fprintln(w, "--\t----\t---\t-----\t-------")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.IP, e.Name, e.MAC, e.State, e.Profile)
	}
	w.Flush()

	return nil
}

func resolveListName(mac, ip string) string {
	if name, ok := cfg.Names[mac]; ok && name != "" {
		return name
	}
	return reverseLookup(ip)
}

func reverseLookup(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return "-"
	}
	name := names[0]
	return strings.TrimSuffix(name, ".")
}
