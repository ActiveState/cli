package artifactcache

import "github.com/go-openapi/strfmt"

// This file exists solely to export private data from ArtifactCache in order to run integration
// tests in an outside package.

type testArtifactCache struct {
	cache *ArtifactCache
}

// NewTestArtifactCache is only meant to be called from tests. Use New() instead.
func NewTestArtifactCache(dir string, maxSize int64) (*testArtifactCache, error) {
	cache, err := newWithDirAndSize(dir, maxSize)
	if err != nil {
		return nil, err
	}
	return &testArtifactCache{cache}, nil
}

func (ac *testArtifactCache) Dir() string {
	return ac.cache.dir
}

func (ac *testArtifactCache) InfoJson() string {
	return ac.cache.infoJson
}

func (ac *testArtifactCache) MaxSize() int64 {
	return ac.cache.maxSize
}

func (ac *testArtifactCache) CurrentSize() int64 {
	return ac.cache.currentSize
}

func (ac *testArtifactCache) Artifacts() map[strfmt.UUID]*cachedArtifact {
	return ac.cache.artifacts
}

func (ac *testArtifactCache) Get(a strfmt.UUID) (string, bool) {
	return ac.cache.Get(a)
}

func (ac *testArtifactCache) Store(a strfmt.UUID, s string) error {
	return ac.cache.Store(a, s)
}

func (ac *testArtifactCache) Save() error {
	return ac.cache.Save()
}