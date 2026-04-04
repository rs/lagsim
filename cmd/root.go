package cmd

import (
	"fmt"
	"lagsim/pkg/config"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	cfgPath string
	cfg     *config.Config
	dryRun  bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "lagsim",
	Short: "Network condition simulator - manage tc/netem per client IP",
	Long: `lagsim manages Linux tc/netem rules to inject network impairments
(latency, jitter, loss, reordering, duplication) per client IP on the LAN.

Run without arguments to launch the interactive TUI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		exitIfNotRoot()
		if cfg.Interfaces.LAN == "" || cfg.Interfaces.Subnet == "" {
			if err := detectInterface(); err != nil {
				return err
			}
		}
		return runTUI()
	},
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", defaultConfigPath(), "config file path")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print commands without executing")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		return nil
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(teardownCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(profilesCmd)
	rootCmd.AddCommand(statusCmd)
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "lagsim.yaml"
	}
	return filepath.Join(home, ".config", "lagsim.yaml")
}

func exitIfNotRoot() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "lagsim requires root privileges")
		os.Exit(1)
	}
}
