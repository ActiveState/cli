package hash

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/ActiveState/cli/internal/errs"
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
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}
}

func (f *FileHasher) HashFiles(files []string) (string, error) {
	sort.Strings(files)

	hasher := xxhash.New()
	for _, file := range files {
		openFile, err := os.Open(file)
		if err != nil {
			return "", errs.Wrap(err, "Could not open file: %s", file)
		}

		fileInfo, err := openFile.Stat()
		if err != nil {
			return "", errs.Wrap(err, "Could not stat file: %s", file)
		}

		var hash string
		fh, ok := f.cache.Get(cacheKey(file, fileInfo.ModTime().String()))
		if ok {
			hash, ok = fh.(string)
			if !ok {
				return "", errs.New("Could not convert cache value to string")
			}
		} else {
			fileHasher := xxhash.New()
			if _, err := io.Copy(fileHasher, openFile); err != nil {
				return "", errs.Wrap(err, "Could not hash file: %s", file)
			}

			if err := openFile.Close(); err != nil {
				return "", errs.Wrap(err, "Could not close file: %s", file)
			}

			hash = fmt.Sprintf("%x", fileHasher.Sum(nil))
		}

		f.cache.Set(cacheKey(file, fileInfo.ModTime().String()), hash, cache.NoExpiration)
		fmt.Fprintf(hasher, "%x", hash)
	}

	return base64.StdEncoding.EncodeToString(hasher.Sum(nil)), nil
}

func cacheKey(file string, modTime string) string {
	return fmt.Sprintf("%s-%s", file, modTime)
}
