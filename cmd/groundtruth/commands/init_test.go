package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RajarshiParmar/groundtruth/internal/config"
)

// initDir moves into a fresh directory whose global git config holds email (none when empty)
func initDir(t *testing.T, email string) string {
	t.Helper()
	tmp := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	if email != "" {
		if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[user]\n\temail = "+email+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(tmp)
	return tmp
}

// runInit runs init and returns its stdout, stderr and error
func runInit(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	stderr = captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			err = execute(t, RootCmd(), append([]string{"init"}, args...)...)
		})
	})
	return stdout, stderr, err
}

func loadWritten(t *testing.T, path string) *config.Config {
	t.Helper()
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	return cfg
}

func TestInit_InRepo(t *testing.T) {
	tmp := initDir(t, "me@example.com")
	makeRepo(t, tmp)

	stdout, stderr, err := runInit(t)
	if err != nil {
		t.Fatal(err)
	}
	cfg := loadWritten(t, "groundtruth.yaml")
	if cfg.Repository.Path != tmp || cfg.Repository.BaseDir {
		t.Errorf("repository = %+v, want path %s, base_dir false", cfg.Repository, tmp)
	}
	if got := cfg.Identity.Authors[0].Email; got != "me@example.com" {
		t.Errorf("author email = %q, want me@example.com", got)
	}
	want := "wrote groundtruth.yaml\n" +
		"next: edit it, then run: groundtruth run\n" +
		"note: groundtruth.yaml holds your email addresses; keep it out of commits (add it to .gitignore)\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestInit_FolderOfRepos(t *testing.T) {
	tmp := initDir(t, "me@example.com")
	for _, name := range []string{"a", "b"} {
		dir := filepath.Join(tmp, name)
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		makeRepo(t, dir)
	}

	stdout, _, err := runInit(t)
	if err != nil {
		t.Fatal(err)
	}
	cfg := loadWritten(t, "groundtruth.yaml")
	if cfg.Repository.Path != tmp || !cfg.Repository.BaseDir {
		t.Errorf("repository = %+v, want path %s, base_dir true", cfg.Repository, tmp)
	}
	if strings.Contains(stdout, "note:") {
		t.Errorf("stdout has a note outside a single repo:\n%s", stdout)
	}
}

func TestInit_EmptyFolder(t *testing.T) {
	tmp := initDir(t, "me@example.com")

	_, stderr, err := runInit(t)
	if err != nil {
		t.Fatal(err)
	}
	if want := "warning: no git repository found in " + tmp + "; edit repository.path"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
	loadWritten(t, "groundtruth.yaml")
}

func TestInit_NoEmail(t *testing.T) {
	tmp := initDir(t, "")
	makeRepo(t, tmp)

	_, stderr, err := runInit(t)
	if err != nil {
		t.Fatal(err)
	}
	if got := loadWritten(t, "groundtruth.yaml").Identity.Authors[0].Email; got != "you@company.com" {
		t.Errorf("author email = %q, want the placeholder you@company.com", got)
	}
	if want := "warning: git config user.email is not set; edit identity.authors in groundtruth.yaml"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}

func TestInit_Exists(t *testing.T) {
	initDir(t, "me@example.com")
	if err := os.WriteFile("groundtruth.yaml", []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runInit(t)
	if err == nil || err.Error() != "groundtruth.yaml already exists (use --force to overwrite)" {
		t.Errorf("err = %v, want the already exists error", err)
	}
	if data, _ := os.ReadFile("groundtruth.yaml"); string(data) != "mine\n" {
		t.Errorf("file changed without --force: %q", data)
	}

	stdout, _, err := runInit(t, "--force")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stdout, "overwrote groundtruth.yaml\n") {
		t.Errorf("stdout = %q, want it to start with overwrote groundtruth.yaml", stdout)
	}
	loadWritten(t, "groundtruth.yaml")
}

func TestInit_CustomPath(t *testing.T) {
	initDir(t, "me@example.com")
	for _, c := range []struct{ path, next string }{
		{"my.yaml", "next: edit it, then run: groundtruth run --config my.yaml\n"},
		{"my file.yaml", "next: edit it, then run: groundtruth run --config 'my file.yaml'\n"},
	} {
		stdout, _, err := runInit(t, c.path)
		if err != nil {
			t.Fatalf("%s: %v", c.path, err)
		}
		if !strings.Contains(stdout, c.next) {
			t.Errorf("%s: stdout = %q, want %q", c.path, stdout, c.next)
		}
		loadWritten(t, c.path)
	}
}

func TestInit_DirectoryTarget(t *testing.T) {
	initDir(t, "me@example.com")
	if err := os.Mkdir("cfg", 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runInit(t, "cfg"); err == nil || err.Error() != "cfg is a directory" {
		t.Errorf("err = %v, want cfg is a directory", err)
	}
}

func TestInit_MissingDir(t *testing.T) {
	initDir(t, "me@example.com")
	if _, _, err := runInit(t, "nope/x.yaml"); err == nil {
		t.Error("init into a missing directory succeeded")
	}
	if _, err := os.Stat("nope"); !os.IsNotExist(err) {
		t.Errorf("nope was created (stat err %v)", err)
	}
}

// defaultConfigDir moves into a directory holding repo/ and a groundtruth.yaml that points at it
func defaultConfigDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	makeRepo(t, repo)
	data, err := os.ReadFile(writeConfig(t, repo, false, "analyze", "  commit_count: true"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, defaultConfig), data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmp)
	return tmp
}

func TestRun_DefaultConfig(t *testing.T) {
	tmp := defaultConfigDir(t)
	captureStdout(t, func() {
		if err := execute(t, RunCmd(), "--dry-run"); err != nil {
			t.Errorf("run --dry-run: %v", err)
		}
		if err := execute(t, RunCmd(), "--output", "out"); err != nil {
			t.Errorf("run --output out: %v", err)
		}
	})
	if entries, err := os.ReadDir(filepath.Join(tmp, "out")); err != nil || len(entries) == 0 {
		t.Errorf("no reports in out/ (err %v)", err)
	}
}

func TestValidate_DefaultConfig(t *testing.T) {
	defaultConfigDir(t)
	captureStdout(t, func() {
		if err := execute(t, ValidateCmd()); err != nil {
			t.Errorf("validate: %v", err)
		}
	})
}

func TestRun_MissingDefaultConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	want := `no config file: groundtruth.yaml not found (pass --config, or create one with "groundtruth init")`
	if err := execute(t, RunCmd()); err == nil || err.Error() != want {
		t.Errorf("run: err = %v, want %q", err, want)
	}
	if err := execute(t, ValidateCmd()); err == nil || err.Error() != want {
		t.Errorf("validate: err = %v, want %q", err, want)
	}
}

func TestRun_ExplicitMissingConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := execute(t, RunCmd(), "--config", "nope.yaml"); err == nil || !strings.HasPrefix(err.Error(), "Failed to read config file:") {
		t.Errorf("err = %v, want it to start with Failed to read config file:", err)
	}
}

// The next: line is meant to be pasted into a shell, so anything the shell would read specially is quoted
func TestShellQuote(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"my.yaml", "my.yaml"},
		{"cfg/my-file_1.yaml", "cfg/my-file_1.yaml"},
		{"my file.yaml", "'my file.yaml'"},
		{"it's.yaml", `'it'\''s.yaml'`},
		{"a$b.yaml", "'a$b.yaml'"},
		{"*.yaml", "'*.yaml'"},
		{"zoë.yaml", "'zoë.yaml'"},
	} {
		if got := shellQuote(c.in); got != c.want {
			t.Errorf("shellQuote(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}
