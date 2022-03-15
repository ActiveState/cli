package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingBuffer(t *testing.T) {
	buf := newRingBuffer(10)
	assert.Equal(t, buf.size, 10)

	p := make([]byte, 10)
	read, err := buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 0, read)
	assert.Equal(t, "", string(p[:read]))

	wrote, err := buf.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, wrote)
	assert.False(t, buf.full)
	read, err = buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 5, read)
	assert.Equal(t, "hello", string(p[:read]))

	wrote, err = buf.Write([]byte("world"))
	assert.NoError(t, err)
	assert.Equal(t, 5, wrote)
	assert.True(t, buf.full)
	read, err = buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 10, read)
	assert.Equal(t, "helloworld", string(p[:read]))

	wrote, err = buf.Write([]byte("!")) // will wrap
	assert.NoError(t, err)
	assert.Equal(t, 1, wrote)
	read, err = buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 10, read)
	assert.Equal(t, "elloworld!", string(p[:read]))

	p = make([]byte, 15) // p is larger than buf.size
	read, err = buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 10, read)
	assert.Equal(t, "elloworld!", string(p[:read]))

	p = make([]byte, 5) // p is smaller than buf.size
	read, err = buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 5, read)
	assert.Equal(t, "orld!", string(p[:read]))

	buf = newRingBuffer(10)
	buf.Write([]byte("hello!"))
	read, err = buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 5, read)
	assert.Equal(t, "ello!", string(p[:read]))

	p = make([]byte, 10)
	msg := "this is longer than buf.size"
	wrote, err = buf.Write([]byte(msg))
	assert.NoError(t, err)
	assert.Equal(t, len(msg), wrote)
	read, err = buf.Read(p)
	assert.NoError(t, err)
	assert.Equal(t, 10, read)
	assert.Equal(t, msg[len(msg)-10:], string(p[:read]))
}
