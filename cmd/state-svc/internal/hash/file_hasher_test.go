package hash

import (
	"os"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

type testCache struct {
	cache  *cache.Cache
	hits   []string
	misses []string
}

func (tc *testCache) Get(key string) (interface{}, bool) {
	val, ok := tc.cache.Get(key)
	if ok {
		tc.hits = append(tc.hits, key)
	} else {
		tc.misses = append(tc.misses, key)
	}

	return val, ok
}

func (tc *testCache) Set(key string, value interface{}, expiration time.Duration) {
	tc.cache.Set(key, value, cache.DefaultExpiration)
}

func TestFileHasher_HashFiles(t *testing.T) {
	file1 := createTempFile(t, "file1")
	file2 := createTempFile(t, "file2")

	hasher := NewFileHasher()

	hash1, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	hash2, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	assert.Equal(t, hash1, hash2)
}

func TestFileHasher_CacheHit(t *testing.T) {
	file1 := createTempFile(t, "file1")
	file2 := createTempFile(t, "file2")

	tc := &testCache{
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	hasher := &FileHasher{
		cache: tc,
	}

	hash1, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	hash2, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	assert.Equal(t, hash1, hash2)
	assert.Len(t, tc.hits, 2)
	assert.Len(t, tc.misses, 2)
}

func TestFileHasher_CacheMiss(t *testing.T) {
	file1 := createTempFile(t, "file1")
	file2 := createTempFile(t, "file2")

	tc := &testCache{
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	hasher := &FileHasher{
		cache: tc,
	}

	hash1, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	if err := os.Chtimes(file1, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(file1)
	assert.NoError(t, err)
	err = file.Sync()
	assert.NoError(t, err)

	hash2, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	assert.Equal(t, hash1, hash2)
	assert.Len(t, tc.hits, 1)
	assert.Len(t, tc.misses, 3)
}

func TestFileHasher_ContentAgnostic(t *testing.T) {
	// Files have same content but different names and modification times
	file1 := createTempFile(t, "file1")

	// Ensure mod times are different
	time.Sleep(1 * time.Millisecond)
	file2 := createTempFile(t, "file1")

	tc := &testCache{
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	hasher := &FileHasher{
		cache: tc,
	}

	hash1, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	hash2, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	assert.Equal(t, hash1, hash2)
	assert.Len(t, tc.hits, 2)
	assert.Len(t, tc.misses, 2)
}

func TestFileHasher_NotEqualFileAdded(t *testing.T) {
	file1 := createTempFile(t, "file1")
	file2 := createTempFile(t, "file2")
	file3 := createTempFile(t, "file3")

	tc := &testCache{
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	hasher := &FileHasher{
		cache: tc,
	}

	hash1, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	hash2, err := hasher.HashFiles([]string{file1, file2, file3})
	assert.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
	assert.Len(t, tc.hits, 2)
	assert.Len(t, tc.misses, 3)
}

func TestFileHasher_NotEqualFileRemoved(t *testing.T) {
	file1 := createTempFile(t, "file1")
	file2 := createTempFile(t, "file2")
	file3 := createTempFile(t, "file3")

	tc := &testCache{
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	hasher := &FileHasher{
		cache: tc,
	}

	hash1, err := hasher.HashFiles([]string{file1, file2, file3})
	assert.NoError(t, err)

	hash2, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	assert.NotEqual(t, hash1, hash2)
	assert.Len(t, tc.hits, 2)
	assert.Len(t, tc.misses, 3)
}

func TestFileHasher_NotEqualContentChanged(t *testing.T) {
	file1 := createTempFile(t, "file1")
	file2 := createTempFile(t, "file2")

	tc := &testCache{
		cache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	hasher := &FileHasher{
		cache: tc,
	}

	hash1, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	hash2, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	assert.Equal(t, hash1, hash2)

	// Change content of file1 and ensure mod time is different to avoid a cache hit.
	// The time these tests take as well as the accuracy of the file system's mod time
	// resolution may cause the mod time to be the same.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(file1, []byte("file1_changed"), 0644); err != nil {
		t.Fatal(err)
	}

	hash2Modified, err := hasher.HashFiles([]string{file1, file2})
	assert.NoError(t, err)

	assert.NotEqual(t, hash1, hash2Modified)
	assert.Len(t, tc.hits, 3)
	assert.Len(t, tc.misses, 3)
}

func createTempFile(t *testing.T, content string) string {
	tmpfile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	return tmpfile.Name()
}
