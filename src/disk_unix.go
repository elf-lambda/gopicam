//go:build !windows

package main

import (
	"golang.org/x/sys/unix"
)

func getDiskSpaceInfo(path string) []uint64 {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	if err != nil {
		return []uint64{0, 0, 0}
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	usable := free
	return []uint64{total, free, usable}
}
