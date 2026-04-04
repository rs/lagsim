package cmd

import (
	"lagsim/pkg/config"
	"lagsim/pkg/tc"
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <ip>",
	Short: "Remove network conditioning from a client IP",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		exitIfNotRoot()
		ip := args[0]

		if err := cfg.ValidateIP(ip); err != nil {
			return err
		}

		r := &tc.Runner{DryRun: dryRun, Verbose: verbose}

		fmt.Printf("Removing profile from %s\n", ip)
		if err := tc.RemoveFromAllDevices(r, cfg, ip); err != nil {
			return err
		}

		delete(cfg.Assignments, ip)
		if err := config.Save(cfg, cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Done. %s is no longer conditioned.\n", ip)
		return nil
	},
}
