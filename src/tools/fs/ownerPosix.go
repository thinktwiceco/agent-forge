//go:build !windows

package fs

import (
	"os"
	"syscall"
)

func getOwnerInfo(info os.FileInfo) (uint32, uint32) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return stat.Uid, stat.Gid
	}
	return 0, 0
}
