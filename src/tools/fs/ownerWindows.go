//go:build windows

package fs

import (
	"os"
)

func getOwnerInfo(info os.FileInfo) (uint32, uint32) {
	// Windows doesn't have simple UID/GID concepts like POSIX
	return 0, 0
}
