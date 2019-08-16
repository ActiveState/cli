package hail

import (
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

	_, fail := Open(done, "/")
	assert.Error(t, fail.ToError())

	file := "garbage"
	rcvs, fail := Open(done, file)
	require.NoError(t, fail.ToError())
	defer func() {
		assert.NoError(t, os.Remove(file))
	}()
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
	case <-time.After(time.Second):
		assert.FailNow(t, "should not block")
	}

	// windows test env has poor time resolution
	if runtime.GOOS != "windows" {
		assert.True(t, r.Open.After(start))
		assert.True(t, postOpen.After(r.Open))
		assert.True(t, r.Time.After(postOpen))
	}
	require.NoError(t, r.Fail.ToError())
	assert.Equal(t, data, r.Data)
}
