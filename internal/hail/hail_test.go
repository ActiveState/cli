package hail

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
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

	wg := &sync.WaitGroup{}
	wg.Add(1)

	data := []byte("some data")

	go func() {
		defer wg.Done()

		r := <-rcvs
		fmt.Println("strt:", start, "\nopen:", r.Open, "\npost:", postOpen, "\nrcvd:", r.Time)
		assert.True(t, r.Open.After(start))
		assert.True(t, postOpen.After(r.Open))
		assert.True(t, r.Time.After(postOpen))
		assert.Equal(t, data, r.Data)
	}()

	//time.Sleep(time.Millisecond * 100) // else windows fails at "r.Time.After(postOpen)"

	f, err := os.OpenFile(file, os.O_TRUNC|os.O_WRONLY, 0660)
	require.NoError(t, err)
	_, err = f.Write(data)
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, f.Close())
	}()

	wg.Wait()
}
