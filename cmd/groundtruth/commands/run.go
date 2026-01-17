package commands

import (
	"fmt"

	"github.com/RajarshiParmar/groundtruth/internal/config"
	"github.com/RajarshiParmar/groundtruth/internal/git"
	"github.com/spf13/cobra"
)

func RunCmd() *cobra.Command {
	var configPath string
	var outputDir string
	var dryRun bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Generate appraisal report from git history",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			scanner := git.NewScanner(cfg.Repository.Path)

			// Convert time range
			from, to, err := config.TimeRangeToBounds(cfg.TimeRange)
			if err != nil {
				return err
			}

			// Build allowed email set
			emails := make(map[string]bool)
			for _, a := range cfg.Identity.Authors {
				if a.Email != "" {
					emails[a.Email] = true
				}
			}

			commits, err := scanner.Scan(from, to, emails)
			if err != nil {
				return err
			}

			if verbose {
				fmt.Printf("scanned %d commits\n", len(commits))
			}

			if dryRun {
				fmt.Println("dry run complete")
				return nil
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "Path to config file (required)")
	cmd.Flags().StringVar(&outputDir, "output", "", "Override output directory")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate config and repo without running analysis")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	if err := cmd.MarkFlagRequired("config"); err != nil {
		panic(err)
	}

	return cmd
}
