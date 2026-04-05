package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"lagsim/pkg/config"

	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List available network profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PROFILE\tDELAY\tJITTER\tDIST\tLOSS\tDUP\tREORDER\tSLOT\tRATE")
		fmt.Fprintln(w, "-------\t-----\t------\t----\t----\t---\t-------\t----\t----")

		for _, name := range cfg.ProfileNames() {
			p := cfg.Profiles[name]
			dl := p.Resolved("download")
			ul := p.Resolved("upload")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				name,
				asym(dl.Delay, ul.Delay),
				asym(dl.Jitter, ul.Jitter),
				asym(dl.Distribution, ul.Distribution),
				asym(lossDisplay(dl), lossDisplay(ul)),
				asym(dl.Duplicate, ul.Duplicate),
				asym(dl.Reorder, ul.Reorder),
				asym(dl.Slot, ul.Slot),
				asym(dl.Rate, ul.Rate),
			)
		}
		w.Flush()
		return nil
	},
}

// asym formats a cell as "value" if symmetric, or "▼ dl ▲ ul" if different.
func asym(dl, ul string) string {
	if dl == "" && ul == "" {
		return "-"
	}
	if dl == ul {
		if dl == "" {
			return "-"
		}
		return dl
	}
	return fmt.Sprintf("▼ %s ▲ %s", dash(dl), dash(ul))
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func lossDisplay(p config.DirectionalProfile) string {
	if p.Loss == "" {
		return ""
	}
	if p.ECN {
		return p.Loss + " ecn"
	}
	return p.Loss
}
