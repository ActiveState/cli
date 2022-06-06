package artifactcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type cachedArtifact struct {
	Id             artifact.ArtifactID `json:"id"`
	ArchivePath    string              `json:"archivePath"`
	Size           int64               `json:"size"`
	LastAccessTime int64               `json:"lastAccessTime"`
}

// ArtifactCache is a cache of downloaded artifacts from the ActiveState Platform.
// The State Tool prefers to use this cache instead of redownloading artifacts.
type ArtifactCache struct {
	dir              string
	infoJson         string
	maxSize          int64 // bytes
	currentSize      int64 // bytes
	artifacts        map[artifact.ArtifactID]*cachedArtifact
	mutex            sync.Mutex
	timeSpentCopying time.Duration
	sizeCopied       int64 //bytes
}

const MB int64 = 1024 * 1024

// New returns a new artifact cache in the State Tool's cache directory with the default maximum size of 500MB.
func New() (*ArtifactCache, error) {
	var maxSize int64 = 500 * MB
	// TODO: size should be configurable and the user should be warned of an invalid size.
	// https://activestatef.atlassian.net/browse/DX-984
	if sizeOverride, err := strconv.Atoi(os.Getenv(constants.ArtifactCacheSizeEnvVarName)); err != nil && sizeOverride > 0 {
		maxSize = int64(sizeOverride) * MB
	}
	return newWithDirAndSize(storage.ArtifactCacheDir(), maxSize)
}

func newWithDirAndSize(dir string, maxSize int64) (*ArtifactCache, error) {
	err := fileutils.MkdirUnlessExists(dir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create artifact cache directory '%s'", dir)
	}

	if !fileutils.IsDir(dir) {
		return nil, errs.New("'%s' is not a directory; cannot use as artifact cache", dir)
	}

	var artifacts []cachedArtifact
	infoJson := filepath.Join(dir, constants.ArtifactCacheFileName)
	if fileutils.FileExists(infoJson) {
		data, err := fileutils.ReadFile(infoJson)
		if err != nil {
			return nil, errs.Wrap(err, "Could not read artifact cache's "+infoJson)
		}
		err = json.Unmarshal(data, &artifacts)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to read cached artifacts from "+infoJson)
		}
	}

	var currentSize int64 = 0
	artifactMap := map[artifact.ArtifactID]*cachedArtifact{}
	for _, artifact := range artifacts {
		currentSize += artifact.Size
		artifactMap[artifact.Id] = &cachedArtifact{artifact.Id, artifact.ArchivePath, artifact.Size, artifact.LastAccessTime}
	}

	logging.Debug("Opened artifact cache at '%s' containing %d artifacts occupying %.1f/%.1f MB", dir, len(artifactMap), float64(currentSize)/float64(MB), float64(maxSize)/float64(MB))
	return &ArtifactCache{dir, infoJson, maxSize, currentSize, artifactMap, sync.Mutex{}, 0, 0}, nil
}

// Get returns the path to the cached artifact with the given id along with true if it exists.
// Otherwise returns an empty string and false.
// Updates the access timestamp if possible so that this artifact is not removed anytime soon.
func (cache *ArtifactCache) Get(a artifact.ArtifactID) (string, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if artifact, found := cache.artifacts[a]; found {
		logging.Debug("Fetched cached artifact '%s' as '%s'; updating access time", string(a), artifact.ArchivePath)
		artifact.LastAccessTime = time.Now().Unix()
		return artifact.ArchivePath, true
	}
	return "", false
}

// Stores the given artifact in the cache.
// If the cache is too small, removes the least-recently accessed artifacts to make room.
func (cache *ArtifactCache) Store(a artifact.ArtifactID, archivePath string) error {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	stat, err := os.Stat(archivePath)
	if err != nil {
		return errs.Wrap(err, "Unable to stat artifact '%s'. Does it exist?", archivePath)
	}
	size := stat.Size()

	if size > cache.maxSize {
		logging.Debug("Cannot avoid exceeding cache size; not storing artifact")
		rollbar.Error("Artifact '%s' is %.1fMB, which exceeds the cache size of %.1fMB", a, float64(size)/float64(MB), float64(cache.maxSize)/float64(MB))
		return nil
	}

	for cache.currentSize+size > cache.maxSize {
		logging.Debug("Storing artifact in cache would exceed cache size; finding least-recently accessed artifact")
		var lastAccessed *cachedArtifact
		for _, artifact := range cache.artifacts {
			if lastAccessed == nil || artifact.LastAccessTime < lastAccessed.LastAccessTime {
				lastAccessed = artifact
			}
		}

		if lastAccessed == nil {
			rollbar.Error("Cannot avoid exceeding cache size; not storing artifact.")
			return nil // avoid infinite loop, but this really shouldn't happen...
		}

		logging.Debug("Removing cached artifact '%s' last accessed on %s", lastAccessed.ArchivePath, time.Unix(lastAccessed.LastAccessTime, 0).Format(time.UnixDate))
		err := os.Remove(lastAccessed.ArchivePath)
		if err != nil {
			return errs.Wrap(err, "Unable to remove cached artifact '%s'", lastAccessed.ArchivePath)
		}
		delete(cache.artifacts, lastAccessed.Id)
		cache.currentSize -= lastAccessed.Size
	}

	targetPath := filepath.Join(cache.dir, string(a))
	startTime := time.Now()
	err = fileutils.CopyFile(archivePath, targetPath)
	cache.timeSpentCopying += time.Since(startTime)
	cache.sizeCopied += size
	if err != nil {
		return errs.Wrap(err, "Unable to copy artifact '%s' into cache as '%s'", archivePath, targetPath)
	}

	logging.Debug("Storing artifact '%s'", targetPath)
	cached := &cachedArtifact{a, targetPath, size, time.Now().Unix()}
	cache.artifacts[a] = cached
	cache.currentSize += size

	return nil
}

// Saves this cache's information to disk.
// You must call this function when you are done utilizing the cache.
func (cache *ArtifactCache) Save() error {
	artifacts := make([]*cachedArtifact, len(cache.artifacts))
	i := 0
	for _, artifact := range cache.artifacts {
		artifacts[i] = artifact
		i++
	}
	data, err := json.Marshal(artifacts)
	if err != nil {
		return errs.Wrap(err, "Unable to store cached artifacts into JSON")
	}

	logging.Debug("Saving artifact cache at '%s'", cache.infoJson)
	err = fileutils.WriteFile(cache.infoJson, data)
	if err != nil {
		return errs.Wrap(err, "Unable to write artifact cache's "+cache.infoJson)
	}

	if cache.timeSpentCopying > 5*time.Second {
		multilog.Log(logging.Debug, rollbar.Error)("Spent %.1f seconds copying %.1fMB of artifacts to cache", cache.timeSpentCopying.Seconds, float64(cache.sizeCopied)/float64(MB))
	}
	cache.timeSpentCopying = 0 // reset
	cache.sizeCopied = 0       //reset

	return nil
}
