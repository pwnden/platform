//go:build development

package playerweb

import "io/fs"

// Development uses Vite; source commands require no embedded production build.
func Read(string) ([]byte, error) { return nil, fs.ErrNotExist }
