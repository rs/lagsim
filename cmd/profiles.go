package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List available network profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PROFILE\tDELAY\tJITTER\tLOSS\tDUP\tREORDER\tRATE")
		fmt.Fprintln(w, "-------\t-----\t------\t----\t---\t-------\t----")

		for _, name := range cfg.ProfileNames() {
			p := cfg.Profiles[name]
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				name,
				dash(p.Delay),
				dash(p.Jitter),
				dash(p.Loss),
				dash(p.Duplicate),
				dash(p.Reorder),
				dash(p.Rate),
			)
		}
		w.Flush()
		return nil
	},
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
