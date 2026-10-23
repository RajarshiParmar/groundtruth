package commands

import (
	"fmt"
	"os"

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
			cfgPath, err := configPathOrDefault(cfgPath)
			if err != nil {
				return err
			}
			outputDir, _ := cmd.Flags().GetString("output")

			cfg, err := config.Load(cfgPath, func(c *config.Config) {
				if outputDir != "" {
					c.Output.Directory = outputDir
				}
			})
			if err != nil {
				return err
			}
			for _, w := range cfg.Warnings {
				fmt.Fprintln(os.Stderr, "warning:", w)
			}

			repos, warnings, err := git.ResolveRepos(cfg.Repository.Path, cfg.Repository.BaseDir)
			for _, w := range warnings {
				fmt.Fprintln(os.Stderr, "warning:", w)
			}
			if err != nil {
				return fmt.Errorf("git: %w", err)
			}

			if cfg.Repository.BaseDir {
				fmt.Printf("found %d usable repositories\n", len(repos))
			}
			fmt.Println("configuration and repository are valid")

			return nil
		},
	}

	cmd.Flags().String("config", "", "Path to config file (default groundtruth.yaml)")
	cmd.Flags().String("output", "", "Override output directory")

	return cmd
}
