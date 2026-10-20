package commands

import (
	"fmt"

	"github.com/RajarshiParmar/groundtruth/internal/config"
	"github.com/RajarshiParmar/groundtruth/internal/git"
	"github.com/spf13/cobra"
)

func ValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			outputDir, _ := cmd.Flags().GetString("output")

			cfg, err := config.Load(cfgPath, func(c *config.Config) {
				if outputDir != "" {
					c.Output.Directory = outputDir
				}
			})
			if err != nil {
				return err
			}

			if cfg.Repository.BaseDir {
				if err := git.ValidateBaseDir(cfg.Repository.Path); err != nil {
					return fmt.Errorf("git: %w", err)
				}
			} else {
				if err := git.ValidateRepo(cfg.Repository.Path); err != nil {
					return fmt.Errorf("git: %w", err)
				}
			}
			fmt.Println("configuration and repository are valid")

			return nil
		},
	}

	cmd.Flags().String("config", "", "Path to config file (required)")
	cmd.Flags().String("output", "", "Override output directory")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}
