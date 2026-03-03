package cmd

import (
	"fmt"
	"os"

	"github.com/christopherluey/sc/internal/config"
	"github.com/christopherluey/sc/internal/tui"
	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

var Version = "dev"

var (
	cfgFile string
	host    string
	user    string
)

var rootCmd = &cobra.Command{
	Use:   "sc",
	Short: "Stanford Compute Cluster TUI — manage Slurm jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, firstRun, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if host != "" {
			cfg.SSH.Host = host
		}
		if user != "" {
			cfg.SSH.User = user
		}

		app := tui.NewApp(cfg, firstRun)
		p := tea.NewProgram(app)
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("sc", Version)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/sc/config.toml)")
	rootCmd.PersistentFlags().StringVar(&host, "host", "", "SSH host override")
	rootCmd.PersistentFlags().StringVar(&user, "user", "", "SSH user override")
	rootCmd.AddCommand(versionCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
