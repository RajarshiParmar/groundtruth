package git

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/config"
)

// ReadIdentity returns user.name and user.email from git config: the repo's own
// settings over the global ones, or only the global ones when repoPath is "".
// A missing or unreadable global config counts as not set.
func ReadIdentity(repoPath string) (name, email string, warnings []string) {
	global, err := config.LoadConfig(config.GlobalScope)
	if err != nil {
		global = nil
	}
	cfg, local := global, (*config.Config)(nil)
	if repoPath != "" {
		repo, err := openRepo(repoPath)
		if err == nil {
			local, err = repo.Config()
		}
		if err == nil {
			cfg, err = repo.ConfigScoped(config.GlobalScope)
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("could not read git config in %s: %v", repoPath, err))
			cfg, local = global, nil
		}
	}
	// go-git does not follow includes, so an identity set in an included file is missed
	if usesInclude(global) || usesInclude(local) {
		warnings = append(warnings, "git config uses include or includeIf, which groundtruth does not read; check the name and email")
	}
	if cfg == nil {
		return "", "", warnings
	}
	return strings.TrimSpace(cfg.User.Name), strings.TrimSpace(cfg.User.Email), warnings
}

func usesInclude(cfg *config.Config) bool {
	if cfg == nil || cfg.Raw == nil {
		return false
	}
	for _, s := range cfg.Raw.Sections {
		if s.IsName("include") || s.IsName("includeIf") {
			return true
		}
	}
	return false
}
