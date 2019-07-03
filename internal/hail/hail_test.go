package hail

import (
	"io/ioutil"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func directory() string {
	dir := "/"
	if runtime.GOOS == "windows" {
		dir = `C:\`
	}
	return dir
}

func TestSend(t *testing.T) {
	fail := Send(directory(), []byte{})
	assert.Error(t, fail.ToError())

	file := "garbage"
	f, err := os.Create(file)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, f.Close())
		assert.NoError(t, os.Remove(file))
	}()

	want := []byte("some data")
	fail = Send(file, want)
	require.NoError(t, fail.ToError())

	got, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, got, want)
}

func TestOpen(t *testing.T) {
	start := time.Now()
	done := make(chan struct{})
	defer close(done)

	_, fail := Open(done, directory())
	assert.Error(t, fail.ToError())

	file := "garbage"
	rcvs, fail := Open(done, file)
	require.NoError(t, fail.ToError())
	defer func() {
		assert.NoError(t, os.Remove(file))
	}()
	postOpen := time.Now()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	data := []byte("some data")

	go func() {
		defer wg.Done()

		r := <-rcvs
		assert.True(t, r.Open.After(start) && postOpen.After(r.Open))
		assert.True(t, r.Time.After(postOpen))
	}()

	f, err := os.OpenFile(file, os.O_TRUNC|os.O_WRONLY, 0660)
	require.NoError(t, err)
	_, err = f.Write(data)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, f.Close())
	}()

	wg.Wait()
}
