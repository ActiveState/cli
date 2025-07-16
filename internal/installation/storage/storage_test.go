package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AppDataPath(t *testing.T) {
	path1 := AppDataPath()
	path2 := AppDataPath()
	assert.Equal(t, path1, path2)
}
