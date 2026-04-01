//go:build !windows

package dirlist

import (
	"os"
	"strings"
)

func isHiddenPath(_ string, entry os.DirEntry) bool {
	return strings.HasPrefix(entry.Name(), ".")
}
