package commands

import (
	"fmt"

	"github.com/RajarshiParmar/groundtruth/internal/config"
	"github.com/spf13/cobra"
)

func ValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")

			if _, err := config.Load(cfgPath); err != nil {
				return err
			}

			fmt.Println("configuration is valid")
			return nil
		},
	}

	cmd.Flags().String("config", "", "Path to config file (required)")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}
