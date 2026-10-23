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
	var cfg *config.Config
	if repoPath != "" {
		repo, err := openRepo(repoPath)
		if err == nil {
			cfg, err = repo.ConfigScoped(config.GlobalScope)
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("could not read git config in %s: %v", repoPath, err))
			cfg = nil
		}
	}
	if cfg == nil {
		global, err := config.LoadConfig(config.GlobalScope)
		if err != nil {
			return "", "", warnings
		}
		cfg = global
	}
	return strings.TrimSpace(cfg.User.Name), strings.TrimSpace(cfg.User.Email), warnings
}
