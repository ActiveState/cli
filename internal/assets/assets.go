package assets

import "embed"

//go:embed contents
var fs embed.FS

// ReadFileBytes reads and returns bytes from the given file in this package's embedded assets.
// Filenames should use forward slashes, not `filepath.Join()`, because go:embed requires '/'.
func ReadFileBytes(filename string) ([]byte, error) {
	return fs.ReadFile("contents/" + filename)
}
