package cmd

import (
	"bufio"
	"fmt"
	"lagsim/pkg/config"
	"lagsim/pkg/netif"
	"lagsim/pkg/tc"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize tc infrastructure and restore saved assignments",
	RunE: func(cmd *cobra.Command, args []string) error {
		exitIfNotRoot()

		// Detect LAN interface if not configured
		if cfg.Interfaces.LAN == "" || cfg.Interfaces.Subnet == "" {
			if err := detectInterface(); err != nil {
				return err
			}
		}

		r := &tc.Runner{DryRun: dryRun, Verbose: verbose}

		if err := tc.Setup(cfg, r); err != nil {
			return err
		}

		// Restore saved assignments
		if len(cfg.Assignments) > 0 {
			fmt.Printf("\nRestoring %d saved assignment(s)...\n", len(cfg.Assignments))
			for ip, profileName := range cfg.Assignments {
				profile, ok := cfg.Profiles[profileName]
				if !ok {
					fmt.Printf("  Warning: profile %q for %s not found, skipping\n", profileName, ip)
					continue
				}
				fmt.Printf("  %s -> %s\n", ip, profileName)
				if err := tc.ApplyToAllDevices(r, cfg, ip, profile); err != nil {
					fmt.Printf("  Warning: failed to apply %s to %s: %v\n", profileName, ip, err)
				}
			}
		}

		fmt.Println("\nlagsim initialized.")
		return nil
	},
}

func detectInterface() error {
	candidates, err := netif.ListCandidates()
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		return fmt.Errorf("no suitable network interfaces found")
	}

	var selected netif.Interface

	if len(candidates) == 1 {
		selected = candidates[0]
		fmt.Printf("Detected LAN interface: %s (%s)\n", selected.Name, selected.Addrs[0])
	} else {
		fmt.Println("Select the LAN interface to condition:")
		fmt.Println()
		for i, iface := range candidates {
			status := "DOWN"
			if iface.IsUp {
				status = "UP"
			}
			fmt.Printf("  %d) %-20s %s  %s\n", i+1, iface.Name, strings.Join(iface.Addrs, ", "), status)
		}
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Printf("Choice [1-%d]: ", len(candidates))
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			n, err := strconv.Atoi(input)
			if err == nil && n >= 1 && n <= len(candidates) {
				selected = candidates[n-1]
				break
			}
			fmt.Println("Invalid choice, try again.")
		}
	}

	// Use the first IPv4 address to derive the subnet
	subnet := netif.SubnetFromCIDR(selected.Addrs[0])

	cfg.Interfaces.LAN = selected.Name
	cfg.Interfaces.Subnet = subnet
	fmt.Printf("Using interface %s, subnet %s\n", selected.Name, subnet)

	// Save so we don't ask again
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}
