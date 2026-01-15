package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

func VersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("groundtruth version:", version)
		},
	}
}
