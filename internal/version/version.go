package version

var (
	// Version is the current version of MailBus
	Version = "0.1.0"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildDate is the date when the binary was built
	BuildDate = "unknown"

	// GoVersion is the version of Go used to build
	GoVersion = "unknown"
)

// Info returns version information
func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"commit":     GitCommit,
		"build_date": BuildDate,
		"go_version": GoVersion,
	}
}
