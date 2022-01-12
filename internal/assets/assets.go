// Holds embeddable assets for the state tool.
package assets

import "embed"

//go:embed *
var fs embed.FS

// Reads and returns bytes from the given file in this package's embedded assets.
func ReadFileBytes(filename string) ([]byte, error) {
	return fs.ReadFile(filename)
}
