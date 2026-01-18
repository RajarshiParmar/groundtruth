package main

import (
	"fmt"
	"os"

	"github.com/RajarshiParmar/groundtruth/cmd/groundtruth/commands"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "groundtruth",
		Short: "Developer-owned appraisal reports from Git commit history",
	}

	rootCmd.AddCommand(commands.RunCmd())
	rootCmd.AddCommand(commands.ValidateCmd())
	rootCmd.AddCommand(commands.VersionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
