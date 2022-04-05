package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRingBuffer(t *testing.T) {
	buf := newRingBuffer(10)
	assert.Equal(t, buf.size, 10)

	contents := buf.Read()
	assert.Equal(t, "", contents)

	wrote, err := buf.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, wrote)
	assert.False(t, buf.full)
	contents = buf.Read()
	assert.NoError(t, err)
	assert.Equal(t, "hello", contents)

	wrote, err = buf.Write([]byte("world"))
	assert.NoError(t, err)
	assert.Equal(t, 5, wrote)
	assert.True(t, buf.full)
	contents = buf.Read()
	assert.NoError(t, err)
	assert.Equal(t, "helloworld", contents)

	wrote, err = buf.Write([]byte("!")) // will wrap
	assert.NoError(t, err)
	assert.Equal(t, 1, wrote)
	contents = buf.Read()
	assert.NoError(t, err)
	assert.Equal(t, "elloworld!", contents)

	msg := "this is longer than buf.size"
	wrote, err = buf.Write([]byte(msg))
	assert.NoError(t, err)
	assert.Equal(t, len(msg), wrote)
	contents = buf.Read()
	assert.NoError(t, err)
	assert.Equal(t, msg[len(msg)-10:], contents)
}
