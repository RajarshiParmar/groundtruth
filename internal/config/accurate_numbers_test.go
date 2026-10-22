package config

import (
	"reflect"
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
