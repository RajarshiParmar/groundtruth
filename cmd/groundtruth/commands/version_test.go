package commands

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	vcs := func(rev, modified string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: rev}, {Key: "vcs.modified", Value: modified}}}
	}
	cases := []struct {
		name, ldflag string
		info         *debug.BuildInfo
		ok           bool
		want         string
	}{
		{"ldflag wins", "v0.2.0", &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}}, true, "v0.2.0"},
		{"go install", "", &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}}, true, "v0.2.0"},
		{"local clean", "", vcs("a1b2c3d4e5f6", "false"), true, "dev (a1b2c3d)"},
		{"local modified", "", vcs("a1b2c3d4e5f6", "true"), true, "dev (a1b2c3d, modified)"},
		{"short revision", "", vcs("abc", "false"), true, "dev (abc)"},
		{"devel no vcs", "", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "dev"},
		// Go 1.24+ stamps local builds with a pseudo-version; the revision reads better
		{"local pseudo dirty", "", withMain(vcs("0478bcc215b8aa", "true"), "v0.1.1-0.20261022181020-0478bcc215b8+dirty"), true, "dev (0478bcc, modified)"},
		{"local pseudo clean", "", withMain(vcs("0478bcc215b8aa", "false"), "v0.1.1-0.20261022181020-0478bcc215b8"), true, "dev (0478bcc)"},
		{"local at tag", "", withMain(vcs("0478bcc215b8aa", "false"), "v0.2.0"), true, "v0.2.0"},
		{"local at tag dirty", "", withMain(vcs("0478bcc215b8aa", "true"), "v0.2.0+dirty"), true, "dev (0478bcc, modified)"},
		{"go install at commit", "", &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20261022181020-0478bcc215b8"}}, true, "v0.0.0-20261022181020-0478bcc215b8"},
		{"empty version", "", &debug.BuildInfo{}, true, "dev"},
		{"no build info", "", nil, false, "dev"},
	}
	for _, c := range cases {
		if got := resolveVersion(c.ldflag, c.info, c.ok); got != c.want {
			t.Errorf("%s: resolveVersion = %q, want %q", c.name, got, c.want)
		}
	}
}

func withMain(info *debug.BuildInfo, v string) *debug.BuildInfo {
	info.Main.Version = v
	return info
}

func TestVersionOutput(t *testing.T) {
	old := version
	version = "v9.9.9"
	t.Cleanup(func() { version = old })

	for _, args := range [][]string{{"version"}, {"--version"}} {
		out := captureStdout(t, func() {
			if err := execute(t, RootCmd(), args...); err != nil {
				t.Errorf("%v: %v", args, err)
			}
		})
		if want := "groundtruth version: v9.9.9\n"; out != want {
			t.Errorf("%v: output = %q, want %q", args, out, want)
		}
	}
}
