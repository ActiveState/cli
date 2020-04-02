package clean

import (
	"os"
)

func removeCache(cachePath string) error {
	return os.RemoveAll(cachePath)
}
