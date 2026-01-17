package commands

import (
	"fmt"

	"github.com/RajarshiParmar/groundtruth/internal/config"
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
			_ = cfg

			if verbose {
				fmt.Println("config loaded successfully")
				if outputDir != "" {
					fmt.Println("output override:", outputDir)
				}
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
