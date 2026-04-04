package cmd

import (
	"lagsim/pkg/config"
	"lagsim/pkg/tc"
	"fmt"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply <ip> <profile>",
	Short: "Apply a network profile to a client IP",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		exitIfNotRoot()
		ip := args[0]
		profileName := args[1]

		if err := cfg.ValidateIP(ip); err != nil {
			return err
		}
		if err := cfg.ValidateProfile(profileName); err != nil {
			return err
		}

		profile := cfg.Profiles[profileName]
		r := &tc.Runner{DryRun: dryRun, Verbose: verbose}

		// Auto-initialize if infrastructure isn't set up
		if cfg.Interfaces.LAN == "" || cfg.Interfaces.Subnet == "" {
			if err := detectInterface(); err != nil {
				return err
			}
		}
		if !r.QdiscExists(cfg.Interfaces.LAN, "htb 1:") {
			fmt.Println("TC infrastructure not initialized, running init...")
			if err := tc.Setup(cfg, r); err != nil {
				return err
			}
		}

		// If client already has a profile, remove it first to ensure clean state
		if existing, ok := cfg.Assignments[ip]; ok {
			fmt.Printf("Replacing profile %s on %s\n", existing, ip)
			_ = tc.RemoveFromAllDevices(r, cfg, ip)
		}

		fmt.Printf("Applying %s to %s\n", profileName, ip)
		if err := tc.ApplyToAllDevices(r, cfg, ip, profile); err != nil {
			return err
		}

		// Save assignment
		cfg.Assignments[ip] = profileName
		if err := config.Save(cfg, cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Done. %s now has profile %s.\n", ip, profileName)
		return nil
	},
}
