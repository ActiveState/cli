package tracker

import (
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

	expected := []string{
		"first-file",
		"second-file",
		"third-file",
	}

	var files []Trackable
	for _, e := range expected {
		f := File{Path: e}
		files = append(files, f)
	}

	err := tracker.Track(files...)
	assert.NoError(t, err)

	actual, err := tracker.GetFiles()
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, expected, actual)
}

func TestTracker_GetEnv(t *testing.T) {
	tracker := initTracker(t)
	defer tracker.Close()

	expected := map[string]string{
		"first-key":  "first-value",
		"second-key": "second-value",
		"third-key":  "third-value",
	}

	var env []Trackable
	for k, v := range expected {
		ev := EnvironmentVariable{Key: k, Value: v}
		env = append(env, ev)
	}

	err := tracker.Track(env...)
	assert.NoError(t, err)

	actual, err := tracker.GetEnvironmentVariables()
	assert.NoError(t, err)
	assert.Len(t, actual, 3)
	assert.Equal(t, actual["first-key"], expected["first-key"])
	assert.Equal(t, actual["second-key"], expected["second-key"])
	assert.Equal(t, actual["third-key"], expected["third-key"])
}
