package cmd

import (
	"lagsim/pkg/tc"
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show raw tc qdisc/class/filter state for debugging",
	RunE: func(cmd *cobra.Command, args []string) error {
		r := &tc.Runner{Verbose: verbose}

		for _, dev := range []string{cfg.Interfaces.LAN, cfg.Interfaces.IFB} {
			fmt.Printf("=== %s ===\n", dev)

			fmt.Println("\n--- qdiscs ---")
			out, _ := r.Output("qdisc", "show", "dev", dev)
			fmt.Print(out)

			fmt.Println("\n--- classes ---")
			out, _ = r.Output("class", "show", "dev", dev)
			fmt.Print(out)

			fmt.Println("\n--- filters ---")
			out, _ = r.Output("filter", "show", "dev", dev)
			fmt.Print(out)

			fmt.Println()
		}

		// Also show ingress filters on LAN
		fmt.Printf("=== %s ingress ===\n", cfg.Interfaces.LAN)
		out, _ := r.Output("filter", "show", "dev", cfg.Interfaces.LAN, "parent", "ffff:")
		fmt.Print(out)

		return nil
	},
}
