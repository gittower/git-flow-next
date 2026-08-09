//go:build !windows && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package cmd

// isTerminal is a conservative fallback for Go targets that neither define a
// termios read ioctl (linux + the BSDs) nor use the Windows console API — e.g.
// aix, solaris, illumos, plan9. Without a portable TTY probe we assume stdin is
// not a terminal, so the first-run activation takes the non-interactive hint
// path rather than blocking on a prompt.
func isTerminal(fd uintptr) bool {
	return false
}
