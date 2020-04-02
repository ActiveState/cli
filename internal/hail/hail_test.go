package hail

import (
	"context"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSend(t *testing.T) {
	fail := Send("/", []byte{})
	assert.Error(t, fail.ToError())

	tempFile, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	file := tempFile.Name()
	defer func() {
		assert.NoError(t, tempFile.Close())
		assert.NoError(t, os.Remove(file))
	}()

	want := []byte("some data")
	fail = Send(file, want)
	require.NoError(t, fail.ToError())

	got, err := ioutil.ReadAll(tempFile)
	require.NoError(t, err)
	assert.Equal(t, got, want)
}

func TestOpen(t *testing.T) {
	file := `/`
	if runtime.GOOS == "windows" {
		file = `xx:\`
	}

	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, fail := Open(ctx, file)
	assert.Error(t, fail.ToError())

	tempFile, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	file = tempFile.Name()
	rcvs, fail := Open(ctx, file)
	defer func() {
		_ = tempFile.Close()
		assert.NoError(t, os.Remove(file))
	}()
	require.NoError(t, fail.ToError())

	postOpen := time.Now()
	data := []byte("some data")

	ready := make(chan struct{})
	go func() {
		f, err := os.OpenFile(file, os.O_TRUNC|os.O_WRONLY, 0660)
		require.NoError(t, err)
		_, err = f.Write(data)
		require.NoError(t, err)
		assert.NoError(t, f.Close())
		close(ready)
	}()
	<-ready

	var r *Received
	select {
	case r = <-rcvs:
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "should not block")
	}

	// windows test env has poor time resolution
	if runtime.GOOS != "windows" {
		assert.True(t, r.Open.After(start))
		assert.True(t, postOpen.After(r.Open))
		assert.True(t, r.Time.After(postOpen))
	}
	require.NoError(t, r.Fail.ToError())
}

func TestOpen_ReceivesClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempFile, err := ioutil.TempFile("", t.Name())
	require.NoError(t, err)

	file := tempFile.Name()
	rcvs, fail := Open(ctx, file)
	require.NoError(t, fail.ToError())
	defer func() {
		_ = tempFile.Close()
		assert.NoError(t, os.Remove(file))
	}()

	cancel()

	var malfunc bool
	select {
	case _, malfunc = <-rcvs:
	case <-time.After(time.Second * 2):
		malfunc = true
	}

	assert.False(t, malfunc, "rcvs should be closed")
}
