package commands

import (
	"fmt"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// version is set at build time:
// -ldflags "-X github.com/RajarshiParmar/groundtruth/cmd/groundtruth/commands.version=v1.2.3"
var version string

// pseudoVersion matches the timestamp and revision of a Go pseudo-version
var pseudoVersion = regexp.MustCompile(`\d{14}-[0-9a-f]{12}`)

// Version returns the build flag version, else the module version recorded
// by `go install`, else the VCS revision of a local build, else "dev"
func Version() string {
	info, ok := debug.ReadBuildInfo()
	return resolveVersion(version, info, ok)
}

func resolveVersion(ldflag string, info *debug.BuildInfo, ok bool) string {
	if ldflag != "" {
		return ldflag
	}
	if !ok || info == nil {
		return "dev"
	}
	v := info.Main.Version
	if v == "(devel)" {
		v = ""
	}
	// A tagged build is shown as is; Go 1.24+ also stamps local builds with
	// pseudo-versions and "+dirty", where the revision reads better
	if v != "" && !pseudoVersion.MatchString(v) && !strings.HasSuffix(v, "+dirty") {
		return v
	}

	var revision string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if revision == "" {
		if v != "" {
			return v
		}
		return "dev"
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	if modified {
		return "dev (" + revision + ", modified)"
	}
	return "dev (" + revision + ")"
}

func VersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("groundtruth version:", Version())
		},
	}
}
