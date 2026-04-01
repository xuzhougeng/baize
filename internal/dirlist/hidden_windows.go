//go:build windows

package dirlist

import (
	"os"
	"strings"
	"syscall"
)

func isHiddenPath(path string, entry os.DirEntry) bool {
	if strings.HasPrefix(entry.Name(), ".") {
		return true
	}
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attrs, err := syscall.GetFileAttributes(ptr)
	if err != nil {
		return false
	}
	return attrs&syscall.FILE_ATTRIBUTE_HIDDEN != 0
}
