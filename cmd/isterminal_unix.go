//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package cmd

import (
	"syscall"
	"unsafe"
)

// isTerminal reports whether the given file descriptor is connected to a
// terminal (TTY). It issues the terminal-attributes ioctl and treats success as
// "is a terminal". A pipe or /dev/null (as used by the test harness and in
// non-interactive invocations) fails the ioctl and is correctly reported as not
// a terminal — this is what distinguishes the first-run prompt path from the
// hint path.
func isTerminal(fd uintptr) bool {
	var termios syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, fd, ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return errno == 0
}
