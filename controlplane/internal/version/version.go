// Package version provides version information for KECS
package version

// These variables are set via ldflags during build time
var (
	// Version is the semantic version of KECS
	Version = "dev"

	// GitCommit is the git commit SHA
	GitCommit = "unknown"

	// BuildDate is the date when the binary was built
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = "unknown"
)

// GetVersion returns the version string
func GetVersion() string {
	if Version == "" {
		return "dev"
	}
	return Version
}

// GetFullVersion returns the full version information
func GetFullVersion() string {
	v := GetVersion()
	if GitCommit != "unknown" && GitCommit != "" {
		v += "-" + GitCommit
	}
	return v
}

// Info contains all version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

// GetInfo returns all version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
	}
}
