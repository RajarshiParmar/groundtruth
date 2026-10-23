package commands

import (
	"os"

	"github.com/spf13/cobra"
)

// RootCmd builds the groundtruth command tree
func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "groundtruth",
		Short:   "Developer-owned appraisal reports from Git commit history",
		Version: Version(),
	}
	root.SetVersionTemplate("groundtruth version: {{.Version}}\n")
	root.SetOut(os.Stdout)

	root.AddCommand(RunCmd())
	root.AddCommand(ValidateCmd())
	root.AddCommand(VersionCmd())

	return root
}
