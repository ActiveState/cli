package hash

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/cespare/xxhash"
	"github.com/patrickmn/go-cache"
)

type fileCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, expiration time.Duration)
}

type FileHasher struct {
	cache fileCache
}

func NewFileHasher() *FileHasher {
	return &FileHasher{
		cache: cache.New(24*time.Hour, 24*time.Hour),
	}
}

func (fh *FileHasher) HashFiles(files []string) (string, error) {
	sort.Strings(files)

	hasher := xxhash.New()
	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			return "", errs.Wrap(err, "Could not open file: %s", file.Name())
		}
		defer rtutils.Closer(file.Close, &rerr)

		fileInfo, err := file.Stat()
		if err != nil {
			return "", errs.Wrap(err, "Could not stat file: %s", file.Name())
		}

		var hash string
		cachedHash, ok := fh.cache.Get(cacheKey(file.Name(), fileInfo.ModTime()))
		if ok {
			hash, ok = cachedHash.(string)
			if !ok {
				return "", errs.New("Could not convert cache value to string")
			}
		} else {
			fileHasher := xxhash.New()
			if _, err := io.Copy(fileHasher, file); err != nil {
				return "", errs.Wrap(err, "Could not hash file: %s", file.Name())
			}

			hash = fmt.Sprintf("%x", fileHasher.Sum(nil))
		}

		fh.cache.Set(cacheKey(file.Name(), fileInfo.ModTime()), hash, cache.NoExpiration)

		// Incorporate the individual file hash into the overall hash in hex format
		fmt.Fprintf(hasher, "%x", hash)
	}

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil)), nil
}

func cacheKey(file string, modTime time.Time) string {
	return fmt.Sprintf("%s-%d", file, modTime.UTC().UnixNano())
}
