package lockdown

import (
	"os"
	"path/filepath"
)

func IsLocked(sourceDir string) bool {
	_, err := os.Stat(filepath.Join(sourceDir, "state", "lockdown.flag"))
	return err == nil
}
