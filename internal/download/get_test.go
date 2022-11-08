package download

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	_, err := Get(filepath.Join("download", "file1"))
  assert.Error(t, err)
	//assert.NoError(t, err, "Should download file")
}
