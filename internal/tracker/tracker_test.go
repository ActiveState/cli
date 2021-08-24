package tracker

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func initTracker(t *testing.T) *Tracker {
	t.Helper()

	dir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)

	tracker, err := newCustom(dir)
	assert.NoError(t, err)

	return tracker
}

func TestTracker_GetFiles(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	var files []Trackable
	var expected []string
	for i := 0; i < 3; i++ {
		path := fmt.Sprintf("/Some/Path/File%d", i)
		files = append(files, NewFile(fmt.Sprintf("FileKey%d", i), path))
		expected = append(expected, path)
	}

	err := tracker.Track(files...)
	assert.NoError(t, err)

	actual, err := tracker.GetFiles()
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, expected, actual)
}

func TestTracker_GetDirectories(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	var dirs []Trackable
	var expected []string
	for i := 0; i < 3; i++ {
		path := fmt.Sprintf("/Some/Path/Dir%d/", i)
		dirs = append(dirs, NewDirectory(fmt.Sprintf("DirKey%d", i), path))
		expected = append(expected, path)
	}

	err := tracker.Track(dirs...)
	assert.NoError(t, err)

	actual, err := tracker.GetDirectories()
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, expected, actual)
}

func TestTracker_GetTags(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	var tags []Trackable
	var expected []string
	for i := 0; i < 3; i++ {
		tag := fmt.Sprintf("Tag%d", i)
		tags = append(tags, NewRCFileTag(fmt.Sprintf("TagKey%d", i), tag))
		expected = append(expected, tag)
	}

	err := tracker.Track(tags...)
	assert.NoError(t, err)

	actual, err := tracker.GetFileTags()
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, expected, actual)
}

func TestTracker_GetEnv(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	var env []Trackable
	expected := make(map[string]string)
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("EnvVarKey%d", i)
		value := fmt.Sprintf("SomeVar%d", i)
		env = append(env, NewEnvironmentVariable(key, value))
		expected[key] = value
	}

	err := tracker.Track(env...)
	assert.NoError(t, err)

	actual, err := tracker.GetEnvironmentVariables()
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, expected, actual)
}

func TestTracker_GetFile(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	expected := "file"
	key := "key"

	err := tracker.Track(NewFile(key, expected))
	assert.NoError(t, err)

	actual, err := tracker.GetFile(key)
	assert.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestTracker_GetDirectory(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	expected := "dir"
	key := "key"

	err := tracker.Track(NewDirectory(key, expected))
	assert.NoError(t, err)

	actual, err := tracker.GetDirectory(key)
	assert.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestTracker_GetFileTag(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	expected := "file"
	key := "key"

	err := tracker.Track(NewRCFileTag(key, expected))
	assert.NoError(t, err)

	actual, err := tracker.GetFileTag(key)
	assert.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestTracker_GetEnvVar(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	expected := "value"
	key := "key"

	err := tracker.Track(NewEnvironmentVariable(key, expected))
	assert.NoError(t, err)

	actual, err := tracker.GetEnvironmentVariable(key)
	assert.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestTracker_GetMixed(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	var track []Trackable
	var expectedFiles []string
	for i := 0; i < 3; i++ {
		path := fmt.Sprintf("/Some/Path/File%d", i)
		track = append(track, NewFile(fmt.Sprintf("FileKey%d", i), path))
		expectedFiles = append(expectedFiles, path)
	}

	var expectedDirs []string
	for i := 0; i < 3; i++ {
		path := fmt.Sprintf("/Some/Path/Dir%d/", i)
		track = append(track, NewDirectory(fmt.Sprintf("DirKey%d", i), path))
		expectedDirs = append(expectedDirs, path)
	}

	var expectedTags []string
	for i := 0; i < 3; i++ {
		tag := fmt.Sprintf("Tag%d", i)
		track = append(track, NewRCFileTag(fmt.Sprintf("TagKey%d", i), tag))
		expectedTags = append(expectedTags, tag)
	}

	expectedEnv := make(map[string]string)
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("EnvVarKey%d", i)
		value := fmt.Sprintf("SomeVar%d", i)
		track = append(track, NewEnvironmentVariable(key, value))
		expectedEnv[key] = value
	}

	err := tracker.Track(track...)
	assert.NoError(t, err)

	actualFiles, err := tracker.GetFiles()
	assert.NoError(t, err)
	assert.Equal(t, expectedFiles, actualFiles)

	actualDirs, err := tracker.GetDirectories()
	assert.NoError(t, err)
	assert.Equal(t, expectedDirs, actualDirs)

	actualTags, err := tracker.GetFileTags()
	assert.NoError(t, err)
	assert.Equal(t, expectedTags, actualTags)

	actualEnv, err := tracker.GetEnvironmentVariables()
	assert.NoError(t, err)
	assert.Equal(t, expectedEnv, actualEnv)
}
