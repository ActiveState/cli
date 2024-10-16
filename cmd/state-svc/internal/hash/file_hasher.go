package hash

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

type hashedFile struct {
	Pattern string
	Path    string
	Hash    string
}

func NewFileHasher() *FileHasher {
	return &FileHasher{
		cache: cache.New(24*time.Hour, 24*time.Hour),
	}
}

func (fh *FileHasher) HashFiles(wd string, globs []string) (_ string, _ []hashedFile, rerr error) {
	sort.Strings(globs) // ensure consistent ordering
	hashedFiles := []hashedFile{}
	hasher := xxhash.New()
	for _, glob := range globs {
		files, err := filepath.Glob(glob)
		if err != nil {
			return "", nil, errs.Wrap(err, "Could not match glob: %s", glob)
		}
		sort.Strings(files) // ensure consistent ordering
		for _, f := range files {
			if !filepath.IsAbs(f) {
				af, err := filepath.Abs(filepath.Join(wd, f))
				if err != nil {
					return "", nil, errs.Wrap(err, "Could not get absolute path for file: %s", f)
				}
				f = af
			}
			file, err := os.Open(f)
			if err != nil {
				return "", nil, errs.Wrap(err, "Could not open file: %s", file.Name())
			}
			defer rtutils.Closer(file.Close, &rerr)

			fileInfo, err := file.Stat()
			if err != nil {
				return "", nil, errs.Wrap(err, "Could not stat file: %s", file.Name())
			}

			var hash string
			cachedHash, ok := fh.cache.Get(cacheKey(file.Name(), fileInfo.ModTime()))
			if ok {
				hash, ok = cachedHash.(string)
				if !ok {
					return "", nil, errs.New("Could not convert cache value to string")
				}
			} else {
				fileHasher := xxhash.New()
				if _, err := io.Copy(fileHasher, file); err != nil {
					return "", nil, errs.Wrap(err, "Could not hash file: %s", file.Name())
				}

				hash = fmt.Sprintf("%016x", fileHasher.Sum64())
			}

			fh.cache.Set(cacheKey(file.Name(), fileInfo.ModTime()), hash, cache.NoExpiration)

			hashedFiles = append(hashedFiles, hashedFile{
				Pattern: glob,
				Path:    file.Name(),
				Hash:    hash,
			})

			// Incorporate the individual file hash into the overall hash in hex format
			fmt.Fprintf(hasher, "%016x", hash)
		}
	}

	return fmt.Sprintf("%016x", hasher.Sum64()), hashedFiles, nil
}

func cacheKey(file string, modTime time.Time) string {
	return fmt.Sprintf("%s-%d", file, modTime.UTC().UnixNano())
}
