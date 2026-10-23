package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RajarshiParmar/groundtruth/internal/config"
	"github.com/RajarshiParmar/groundtruth/internal/git"
	"github.com/spf13/cobra"
)

// defaultConfig is the file init writes and run and validate read without --config
const defaultConfig = "groundtruth.yaml"

// placeholderEmail stands in for an unset git user.email, the same as in config.example.yaml
const placeholderEmail = "you@company.com"

func InitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Write a starter config for the current directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := defaultConfig
			if len(args) == 1 {
				target = args[0]
			}
			force, _ := cmd.Flags().GetBool("force")

			info, err := os.Stat(target)
			exists := err == nil
			if exists && info.IsDir() {
				return fmt.Errorf("%s is a directory", target)
			}
			if exists && !force {
				return fmt.Errorf("%s already exists (use --force to overwrite)", target)
			}
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}

			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			if wd, err = filepath.Abs(wd); err != nil {
				return err
			}

			// The working directory as one repo, else as a folder of repos; parents are not searched, as in run
			v := config.StarterValues{Path: wd}
			singleRepo := git.ValidateRepo(wd) == nil
			if !singleRepo {
				repos, warnings, err := git.ResolveRepos(wd, true)
				for _, w := range warnings {
					fmt.Fprintln(os.Stderr, "warning:", w)
				}
				if err == nil && len(repos) > 0 {
					v.BaseDir = true
				} else {
					fmt.Fprintf(os.Stderr, "warning: no git repository found in %s; edit repository.path\n", wd)
				}
			}

			identityRepo := ""
			if singleRepo {
				identityRepo = wd
			}
			name, email, warnings := git.ReadIdentity(identityRepo)
			for _, w := range warnings {
				fmt.Fprintln(os.Stderr, "warning:", w)
			}
			v.Name, v.Email = name, email
			if v.Email == "" {
				v.Email = placeholderEmail
				fmt.Fprintf(os.Stderr, "warning: git config user.email is not set; edit identity.authors in %s\n", target)
			}

			if err := os.WriteFile(target, config.Starter(v), 0o644); err != nil {
				return err
			}

			verb := "wrote"
			if exists {
				verb = "overwrote"
			}
			fmt.Println(verb, target)
			next := "groundtruth run"
			if target != defaultConfig {
				next += " --config " + shellQuote(target)
			}
			fmt.Println("next: edit it, then run:", next)
			if singleRepo && inside(wd, target) {
				fmt.Printf("note: %s holds your email addresses; keep it out of commits (add it to .gitignore)\n", target)
			}
			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Overwrite an existing config file")

	return cmd
}

// configPathOrDefault returns the --config value, or groundtruth.yaml when the flag is empty and that file exists
func configPathOrDefault(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if _, err := os.Stat(defaultConfig); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf(`no config file: %s not found (pass --config, or create one with "groundtruth init")`, defaultConfig)
	}
	return defaultConfig, nil
}

// shellQuote single-quotes a path the shell would read specially, so the printed command can be pasted as is
func shellQuote(path string) string {
	if path != "" && !strings.ContainsFunc(path, func(r rune) bool { return !strings.ContainsRune(shellSafe, r) }) {
		return path
	}
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}

// shellSafe holds the characters no POSIX shell treats specially
const shellSafe = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@%+=:,./-_"

// inside reports whether path, relative to the working directory, lies under dir
func inside(dir, path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dir, abs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
