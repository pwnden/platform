//go:build !development

// Package playerweb contains the browser build embedded in the executable.
// Build the assets with Dockerfile.web before compiling Go from source.
package playerweb

import "embed"

//go:embed dist/index.html dist/assets
var files embed.FS

// Read returns a public asset. Paths are exact; no filesystem traversal,
// directory listings or SPA route fallbacks are performed.
func Read(path string) ([]byte, error) {
	if path == "/" {
		return files.ReadFile("dist/index.html")
	}
	return files.ReadFile("dist" + path)
}
