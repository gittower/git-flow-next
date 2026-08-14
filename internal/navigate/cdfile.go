// Package navigate implements the channel git-flow uses to hand a destination
// directory back to the calling shell.
//
// git-flow runs as a subprocess and cannot change its parent shell's working
// directory. A command that would move the user therefore writes its destination
// to the file named by GIT_FLOW_CD_FILE, leaving the caller that set the variable
// to read the file and change directory itself. The variable is an input git-flow
// reads, never one it sets: it describes one invocation's calling environment,
// not a repository preference, which is why it is not a git config key.
package navigate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvVar names the environment variable holding the destination file.
const EnvVar = "GIT_FLOW_CD_FILE"

// DestinationFile returns the absolute path of the file to write the destination
// to, or "" when the channel is not in use.
//
// Resolve this BEFORE an operation that may delete the current working directory
// (removing the worktree the user is standing in): a relative destination can
// only be made absolute while that directory still exists.
func DestinationFile() string {
	file := strings.TrimSpace(os.Getenv(EnvVar))
	if file == "" {
		return ""
	}
	if abs, err := filepath.Abs(file); err == nil {
		return abs
	}
	return file
}

// WriteDestination writes path as the destination for the calling shell. It is a
// no-op when the channel is not in use.
//
// Call it only AFTER the operation it follows has succeeded, so a refused or
// failed command leaves the file empty.
func WriteDestination(path string) error {
	return WriteDestinationTo(DestinationFile(), path)
}

// WriteDestinationTo writes path to a destination file resolved earlier by
// DestinationFile. An empty file argument is a no-op, so callers need no
// conditional of their own.
func WriteDestinationTo(file string, path string) error {
	if file == "" {
		return nil
	}

	destination := path
	if abs, err := filepath.Abs(path); err == nil {
		destination = abs
	}

	if err := os.WriteFile(file, []byte(destination+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write navigation destination to %s: %w", file, err)
	}
	return nil
}
