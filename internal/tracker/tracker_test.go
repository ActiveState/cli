package tracker

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTracker_GetFiles(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)

	tracker, err := newCustom(dir)
	assert.NoError(t, err)
	defer tracker.Close()

	expected := []string{
		"first-file",
		"second-file",
		"third-file",
	}

	var files []File
	for _, e := range expected {
		f := File{Path: e}
		files = append(files)
		err = tracker.Track(f)
		assert.NoError(t, err)
	}

	actual, err := tracker.GetFiles()
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, expected, actual)
}

func TestTracker_GetEnv(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)

	tracker, err := newCustom(dir)
	assert.NoError(t, err)
	defer tracker.Close()

	expected := map[string]string{
		"first-key":  "first-value",
		"second-key": "second-value",
		"third-key":  "third-value",
	}

	for k, v := range expected {
		ev := EnvironmentVariable{Key: k, Value: v}
		err = tracker.Track(ev)
		assert.NoError(t, err)
	}

	actual, err := tracker.GetEnvironmentVariables()
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, actual["first-key"], expected["first-key"])
	assert.Equal(t, actual["second-key"], expected["second-key"])
	assert.Equal(t, actual["third-key"], expected["third-key"])
}
