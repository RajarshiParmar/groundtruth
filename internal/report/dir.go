package report

import "os"

// ensureDir creates the output directory so each writer can be called on its own
func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
