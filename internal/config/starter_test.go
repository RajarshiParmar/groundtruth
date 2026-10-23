package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The example config is the starter rendered with placeholder values, so the two cannot drift apart
func TestStarter_MatchesExample(t *testing.T) {
	want, err := os.ReadFile("../../config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	got := Starter(StarterValues{Path: ".", Name: "Your Name", Email: "you@company.com"})
	if !bytes.Equal(got, want) {
		t.Errorf("config.example.yaml and internal/config/init.yaml.tmpl differ; update both\n--- got ---\n%s", got)
	}
}

func TestStarter_RoundTrip(t *testing.T) {
	for _, v := range []StarterValues{
		{Path: "/home/me/My Repos #1", Name: "O'Brien: Jr", Email: "ob@example.com", BaseDir: true},
		{Path: "/srv/zoë", Name: "Zoë", Email: "z@example.com"},
		{Path: "/r", Name: "yes", Email: "*star@example.com"},
		{Path: "/r", Name: "", Email: "only@example.com"},
		// git config allows escaped newlines; block scalars would break the template's indentation
		{Path: "/r/a\nb", Name: "A\nB", Email: "e\n@example.com", BaseDir: true},
	} {
		data := Starter(v)
		file := filepath.Join(t.TempDir(), "groundtruth.yaml")
		if err := os.WriteFile(file, data, 0o644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(file)
		if err != nil {
			t.Errorf("%+v: Load: %v\n%s", v, err, data)
			continue
		}
		if len(cfg.Warnings) > 0 {
			t.Errorf("%+v: warnings %v", v, cfg.Warnings)
		}
		if cfg.Repository.Path != v.Path || cfg.Repository.BaseDir != v.BaseDir {
			t.Errorf("%+v: repository = %+v", v, cfg.Repository)
		}
		if len(cfg.Identity.Authors) != 1 {
			t.Errorf("%+v: authors = %+v, want one", v, cfg.Identity.Authors)
			continue
		}
		if a := cfg.Identity.Authors[0]; a.Name != v.Name || a.Email != v.Email {
			t.Errorf("%+v: author = %+v", v, a)
		}
		if v.Name == "" && strings.Contains(string(data), "- name:") {
			t.Errorf("%+v: output has a name line for an empty name\n%s", v, data)
		}
	}
}
