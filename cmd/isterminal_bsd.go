//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package cmd

import "syscall"

// ioctlReadTermios is the ioctl request that reads terminal attributes on the
// BSD family (including macOS/Darwin).
const ioctlReadTermios = syscall.TIOCGETA
