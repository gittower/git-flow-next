package version

// Version information
const (
	// Version is the current version of git-flow-next
	Version = "0.1.0-alpha.1"

	// BuildTime will be injected during build
	BuildTime = ""

	// GitCommit will be injected during build
	GitCommit = ""
)

// GetVersionInfo returns a formatted version string
func GetVersionInfo() string {
	if BuildTime != "" && GitCommit != "" {
		return Version + " (built " + BuildTime + " from " + GitCommit + ")"
	}
	return Version
}
