package assets

import (
	"embed"
	"path/filepath"
)

//go:embed contents
var fs embed.FS

// ReadFileBytes reads and returns bytes from the given file in this package's embedded assets.
func ReadFileBytes(filename string) ([]byte, error) {
	return fs.ReadFile(filepath.Join("contents", filename))
}
