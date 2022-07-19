package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifactcache"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	// Note: the length in bytes of each artifact is its index.
	testArtifacts := []strfmt.UUID{
		"000000000-0000-0000-0000-000000000000",
		"74D554B3-6B0F-434B-AFE2-9F2F0B5F32BA",
		"87ADD1B0-169D-4C01-8179-191BB9910799",
		"5D8D933F-09FA-45A3-81FF-E6F33E91C9ED",
		"992B8488-C61D-433C-ADF2-D76EBD8DAE59",
		"2C36A315-59ED-471B-8629-2663ECC95476",
		"57E8EAF4-F7EE-4BEF-B437-D9F0A967BA52",
		"E299F10C-7B5D-4B25-B821-90E30193A916",
		"F95C0ECE-9F69-4998-B83F-CE530BACD468",
		"CAC9708D-FAA6-4295-B640-B8AA41A8AABC",
		"009D20C9-0E38-44E8-A095-7B6FEF01D7DA",
	}

	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	// Test cache creation.
	cache, err := artifactcache.NewTestArtifactCache(dir, 10) // bytes
	require.NoError(t, err)
	assert.Equal(t, cache.Dir(), dir)
	assert.False(t, fileutils.FileExists(cache.InfoJson())) // not yet
	assert.Equal(t, cache.MaxSize(), int64(10))
	assert.Equal(t, cache.CurrentSize(), int64(0))
	assert.Empty(t, cache.Artifacts())

	// Test cache.Get() with empty cache.
	path, found := cache.Get(testArtifacts[1])
	assert.Empty(t, path)
	assert.False(t, found)

	// Test cache.Store().
	testArtifactFile := osutil.GetTestFile(string(testArtifacts[1]))
	err = cache.Store(testArtifacts[1], testArtifactFile)
	require.NoError(t, err)
	assert.Equal(t, len(cache.Artifacts()), 1)
	assert.Equal(t, cache.CurrentSize(), int64(1))

	cached := cache.Artifacts()[testArtifacts[1]] // will test cache.Get() later; avoid last access time update
	assert.Equal(t, cached.Id, testArtifacts[1])
	assert.Equal(t, cached.ArchivePath, filepath.Join(cache.Dir(), string(testArtifacts[1])))
	assert.Equal(t, cached.Size, int64(1))
	assert.True(t, cached.LastAccessTime > 0)

	cachedFile := cached.ArchivePath
	assert.True(t, fileutils.FileExists(cachedFile))
	assert.Equal(t, fileutils.ReadFileUnsafe(testArtifactFile), fileutils.ReadFileUnsafe(cachedFile))

	// Test cache.Get() and last access time updating.
	lastAccessTime := cached.LastAccessTime
	time.Sleep(1 * time.Second)
	path, found = cache.Get(testArtifacts[1])
	assert.Equal(t, path, cachedFile)
	assert.True(t, found)
	assert.True(t, cached.LastAccessTime > lastAccessTime)

	// Test cache.Store() and removing least-recently accessed artifacts.
	time.Sleep(1 * time.Second)
	cache.Store(testArtifacts[3], osutil.GetTestFile(string(testArtifacts[3])))
	cache.Store(testArtifacts[5], osutil.GetTestFile(string(testArtifacts[5])))
	assert.Equal(t, cache.CurrentSize(), int64(9))
	assert.Equal(t, len(cache.Artifacts()), 3)

	cache.Store(testArtifacts[2], osutil.GetTestFile(string(testArtifacts[2])))
	assert.Equal(t, cache.CurrentSize(), int64(10))
	assert.Equal(t, len(cache.Artifacts()), 3)
	assert.Nil(t, cache.Artifacts()[testArtifacts[1]])
	assert.NotNil(t, cache.Artifacts()[testArtifacts[2]])
	assert.NotNil(t, cache.Artifacts()[testArtifacts[3]])
	assert.NotNil(t, cache.Artifacts()[testArtifacts[5]])

	// Test cache.Save().
	err = cache.Save()
	require.NoError(t, err)
	assert.True(t, fileutils.FileExists(cache.InfoJson()))

	reloaded, err := artifactcache.NewTestArtifactCache(cache.Dir(), 10)
	require.NoError(t, err)
	assert.Equal(t, reloaded.CurrentSize(), int64(10))
	assert.Equal(t, len(reloaded.Artifacts()), 3)
	assert.NotNil(t, reloaded.Artifacts()[testArtifacts[2]])
	assert.NotNil(t, reloaded.Artifacts()[testArtifacts[3]])
	assert.NotNil(t, reloaded.Artifacts()[testArtifacts[5]])

	// Test too small of a cache max size.
	dir, err = os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	cache, err = artifactcache.NewTestArtifactCache(dir, 1) // bytes
	require.NoError(t, err)
	cache.Store(testArtifacts[1], osutil.GetTestFile(string(testArtifacts[1])))
	cache.Store(testArtifacts[2], osutil.GetTestFile(string(testArtifacts[2]))) // should not store nor erase existing artifacts
	assert.Equal(t, cache.CurrentSize(), int64(1))
	assert.Equal(t, len(cache.Artifacts()), 1)
	assert.NotNil(t, cache.Artifacts()[testArtifacts[1]])
}
