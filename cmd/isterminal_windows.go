//go:build windows

package cmd

import (
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
)

// isTerminal reports whether the given file descriptor is attached to a Windows
// console. GetConsoleMode succeeds only for a real console handle, so a pipe or
// the null device is correctly reported as not a terminal.
func isTerminal(fd uintptr) bool {
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(fd, uintptr(unsafe.Pointer(&mode)))
	return r != 0
}
