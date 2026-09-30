// Package installation manages the platform's bundled problem distribution.
package installation

// These values are supplied by the platform package builder. Development builds
// can continue to use an explicit repository without a bundled distribution.
var (
	catalogRevision string
	catalogSHA256   string
)

const ArchiveName = "catalog.tar.gz"
