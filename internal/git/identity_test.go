package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5"
)

// isolateHome points the global git config at an empty temp home; global, if set, becomes ~/.gitconfig
func isolateHome(t *testing.T, global string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	if global != "" {
		if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(global), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

const globalIdentity = "[user]\n\tname = G\n\temail = g@example.com\n"

func TestReadIdentity_Global(t *testing.T) {
	isolateHome(t, globalIdentity)
	name, email, warnings := ReadIdentity("")
	if name != "G" || email != "g@example.com" || len(warnings) > 0 {
		t.Errorf("ReadIdentity = %q, %q, %v; want G, g@example.com, no warnings", name, email, warnings)
	}
}

func TestReadIdentity_RepoOverridesGlobal(t *testing.T) {
	isolateHome(t, globalIdentity)
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatal(err)
	}
	local := "[core]\n\tbare = false\n[user]\n\temail = l@example.com\n"
	if err := os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}
	name, email, warnings := ReadIdentity(dir)
	if name != "G" || email != "l@example.com" || len(warnings) > 0 {
		t.Errorf("ReadIdentity = %q, %q, %v; want G, l@example.com, no warnings", name, email, warnings)
	}
}

func TestReadIdentity_NotSet(t *testing.T) {
	isolateHome(t, "")
	name, email, warnings := ReadIdentity("")
	if name != "" || email != "" || len(warnings) > 0 {
		t.Errorf("ReadIdentity = %q, %q, %v; want empty, no warnings", name, email, warnings)
	}
}

func TestReadIdentity_NoHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	name, email, _ := ReadIdentity("")
	if name != "" || email != "" {
		t.Errorf("ReadIdentity = %q, %q; want empty", name, email)
	}
}

func TestReadIdentity_BadRepo(t *testing.T) {
	isolateHome(t, globalIdentity)
	dir := t.TempDir()
	name, email, warnings := ReadIdentity(dir)
	if name != "G" || email != "g@example.com" {
		t.Errorf("ReadIdentity = %q, %q; want the global G, g@example.com", name, email)
	}
	if len(warnings) != 1 || !strings.HasPrefix(warnings[0], "could not read git config in "+dir+": ") {
		t.Errorf("warnings = %v, want one starting with %q", warnings, "could not read git config in "+dir)
	}
}

// go-git does not follow include or includeIf, so a work email set that way would be missed without a word
func TestReadIdentity_IncludeWarns(t *testing.T) {
	const want = "git config uses include or includeIf, which groundtruth does not read; check the name and email"
	home := isolateHome(t, globalIdentity+"[includeIf \"gitdir:~/work/\"]\n\tpath = ~/.gitconfig-work\n")
	_, _, warnings := ReadIdentity("")
	if len(warnings) != 1 || warnings[0] != want {
		t.Errorf("global includeIf: warnings = %v, want [%q]", warnings, want)
	}

	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(globalIdentity), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatal(err)
	}
	local := "[core]\n\tbare = false\n[include]\n\tpath = ../extra.gitconfig\n"
	if err := os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, warnings = ReadIdentity(dir)
	if len(warnings) != 1 || warnings[0] != want {
		t.Errorf("local include: warnings = %v, want [%q]", warnings, want)
	}
}
