//go:build linux

package cmd

import "syscall"

// ioctlReadTermios is the ioctl request that reads terminal attributes on Linux.
const ioctlReadTermios = syscall.TCGETS
