//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
)

func getDiskSpaceInfo(path string) []uint64 {
	lpFreeBytesAvailable := uint64(0)
	lpTotalNumberOfBytes := uint64(0)
	lpTotalNumberOfFreeBytes := uint64(0)

	pathPtr, _ := syscall.UTF16PtrFromString(path)
	ret, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&lpFreeBytesAvailable)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfBytes)),
		uintptr(unsafe.Pointer(&lpTotalNumberOfFreeBytes)),
	)

	if ret == 0 {
		return []uint64{0, 0, 0}
	}

	return []uint64{lpTotalNumberOfBytes, lpTotalNumberOfFreeBytes, lpFreeBytesAvailable}
}
