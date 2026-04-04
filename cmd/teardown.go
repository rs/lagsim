package cmd

import (
	"lagsim/pkg/tc"

	"github.com/spf13/cobra"
)

var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Remove all tc rules and tear down ifb",
	RunE: func(cmd *cobra.Command, args []string) error {
		exitIfNotRoot()
		r := &tc.Runner{DryRun: dryRun, Verbose: verbose}
		tc.Teardown(cfg, r)
		return nil
	},
}
