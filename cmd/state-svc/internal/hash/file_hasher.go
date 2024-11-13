package hash

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/bmatcuk/doublestar/v4"
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
	fs := os.DirFS(wd)
	hashedFiles := []hashedFile{}
	hashes := []string{}
	for _, glob := range globs {
		files, err := doublestar.Glob(fs, glob)
		if err != nil {
			return "", nil, errs.Wrap(err, "Could not match glob: %s", glob)
		}
		sort.Strings(files) // ensure consistent ordering
		for _, relativePath := range files {
			absolutePath, err := filepath.Abs(filepath.Join(wd, relativePath))
			if err != nil {
				return "", nil, errs.Wrap(err, "Could not get absolute path for file: %s", relativePath)
			}
			fileInfo, err := os.Stat(absolutePath)
			if err != nil {
				return "", nil, errs.Wrap(err, "Could not stat file: %s", absolutePath)
			}

			if fileInfo.IsDir() {
				continue
			}

			var hash string
			cachedHash, ok := fh.cache.Get(cacheKey(fileInfo.Name(), fileInfo.ModTime()))
			if ok {
				hash, ok = cachedHash.(string)
				if !ok {
					return "", nil, errs.New("Could not convert cache value to string")
				}
			} else {
				fileHasher := xxhash.New()
				// include filepath in hash, because moving files should affect the hash
				fmt.Fprintf(fileHasher, "%016x", relativePath)
				file, err := os.Open(absolutePath)
				if err != nil {
					return "", nil, errs.Wrap(err, "Could not open file: %s", absolutePath)
				}
				defer file.Close()
				if _, err := io.Copy(fileHasher, file); err != nil {
					return "", nil, errs.Wrap(err, "Could not hash file: %s", fileInfo.Name())
				}

				hash = fmt.Sprintf("%016x", fileHasher.Sum64())
			}

			fh.cache.Set(cacheKey(fileInfo.Name(), fileInfo.ModTime()), hash, cache.NoExpiration)

			hashes = append(hashes, hash)
			hashedFiles = append(hashedFiles, hashedFile{
				Pattern: glob,
				Path:    relativePath,
				Hash:    hash,
			})
		}
	}

	if hashedFiles == nil {
		return "", nil, nil
	}

	// Ensure the overall hash is consistently calculated
	sort.Slice(hashedFiles, func(i, j int) bool { return hashedFiles[i].Path < hashedFiles[j].Path })
	h := xxhash.New()
	for _, f := range hashedFiles {
		fmt.Fprintf(h, "%016x", f.Hash)
	}

	return fmt.Sprintf("%016x", h.Sum64()), hashedFiles, nil
}

func cacheKey(file string, modTime time.Time) string {
	return fmt.Sprintf("%s-%d", file, modTime.UTC().UnixNano())
}
