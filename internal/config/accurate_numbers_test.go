package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/RajarshiParmar/groundtruth/internal/paths"
)

func TestLoad_ExcludePaths(t *testing.T) {
	cfg, err := Load(writeConfig(t, validBase+"exclude_paths:\n  - go.sum\n  - \"vendor/**\"\n"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := []string{"go.sum", "vendor/**"}; !reflect.DeepEqual(cfg.ExcludePaths, want) {
		t.Errorf("ExcludePaths = %q, want %q", cfg.ExcludePaths, want)
	}

	_, err = Load(writeConfig(t, validBase+"exclude_paths: [\"a/[b\"]\n"))
	if err == nil || err.Error() != `exclude_paths: invalid pattern "a/[b"` {
		t.Errorf("bad pattern: error %v", err)
	}
}

func withIdentity(extra string) string {
	return strings.Replace(validBase, "identity:\n", "identity:\n"+extra, 1)
}

func TestLoad_IdentityMailmap(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "x")
	if err := os.WriteFile(file, []byte("<a@x.com> <b@x.com>\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(writeConfig(t, withIdentity("  mailmap: "+file+"\n")))
	if err != nil || cfg.Identity.Mailmap != file {
		t.Fatalf("Load: %v, Mailmap %q", err, cfg.Identity.Mailmap)
	}

	t.Setenv("HOME", dir)
	if _, err := Load(writeConfig(t, withIdentity("  mailmap: ~/x\n"))); err != nil {
		t.Errorf("~/ path: %v", err)
	}
	if got, err := ExpandHome("~/x"); err != nil || got != file {
		t.Errorf("ExpandHome = %q, %v; want %q", got, err, file)
	}
	if got, _ := ExpandHome("rel/x"); got != "rel/x" {
		t.Errorf("ExpandHome(rel/x) = %q", got)
	}

	_, err = Load(writeConfig(t, withIdentity("  mailmap: "+filepath.Join(dir, "missing")+"\n")))
	if err == nil || !strings.HasPrefix(err.Error(), "identity.mailmap: ") || !strings.Contains(err.Error(), "no such file") {
		t.Errorf("missing file: error %v", err)
	}
}

func TestLoad_CoAuthoredWarning(t *testing.T) {
	const warning = "identity.co_authored has no effect without identity.authors: every author already counts."

	noAuthors := strings.Replace(validBase, "  authors:\n    - name: x\n      email: x@y.z\n", "  co_authored: true\n", 1)
	cfg, err := Load(writeConfig(t, noAuthors))
	if err != nil || !slices.Contains(cfg.Warnings, warning) {
		t.Errorf("no authors: err %v, Warnings %q", err, cfg.Warnings)
	}

	cfg, err = Load(writeConfig(t, withIdentity("  co_authored: true\n")))
	if err != nil || !cfg.Identity.CoAuthored || len(cfg.Warnings) != 0 {
		t.Errorf("with authors: err %v, CoAuthored %v, Warnings %q", err, cfg.Identity.CoAuthored, cfg.Warnings)
	}
}

func TestLoad_IdentityMailmapNoHome(t *testing.T) {
	t.Setenv("HOME", "")
	_, err := Load(writeConfig(t, withIdentity("  mailmap: ~/x\n")))
	if err == nil || !strings.HasPrefix(err.Error(), "identity.mailmap: ") {
		t.Errorf("no home: error %v", err)
	}
}

// Patterns written like directories match no file path, so they would leave
// the counts unchanged without a word
func TestLoad_ExcludePathsDeadPatternWarns(t *testing.T) {
	cfg, err := Load(writeConfig(t, validBase+"exclude_paths: [\"vendor/\", \"/vendor/**\", 'vendor\\**', \"**/vendor/**\"]\n"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var got []string
	for _, w := range cfg.Warnings {
		if strings.HasPrefix(w, "exclude_paths: ") {
			got = append(got, w)
		}
	}
	const hint = ` never matches a file: paths are relative to the repo root, use "/" between directories and no "/" at either end (for example "**/vendor/**")`
	want := []string{`exclude_paths: "vendor/"` + hint, `exclude_paths: "/vendor/**"` + hint, `exclude_paths: "vendor\\**"` + hint}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("warnings =\n%q\nwant\n%q", got, want)
	}
}

// The suggested defaults must also cover nested vendor and node_modules
// directories (monorepos), and must not warn
func TestExampleConfig_ExcludePaths(t *testing.T) {
	data, err := os.ReadFile("../../config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var example struct {
		ExcludePaths []string `yaml:"exclude_paths"`
	}
	if err := yaml.Unmarshal(data, &example); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(writeConfig(t, validBase+"exclude_paths: ["+`"`+strings.Join(example.ExcludePaths, `", "`)+`"`+"]\n"))
	if err != nil || len(cfg.Warnings) != 0 {
		t.Fatalf("Load: %v, warnings %q", err, cfg.Warnings)
	}
	e, err := paths.NewExcluder(example.ExcludePaths)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"go.sum", "web/package-lock.json", "vendor/a.go", "svc/vendor/a.go", "node_modules/x.js", "web/node_modules/x/y.js", "api/v1.pb.go"} {
		if !e.Match(p) {
			t.Errorf("example exclude_paths do not match %s", p)
		}
	}
	if e.Match("svc/main.go") {
		t.Error("example exclude_paths match svc/main.go")
	}
}
