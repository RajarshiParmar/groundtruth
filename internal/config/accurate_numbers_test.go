package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
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
