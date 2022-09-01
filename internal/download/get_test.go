package download

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	req, err := NewRequest(filepath.Join("download", "file1"))
	assert.NoError(t, err)
	_, err = Get(req)
	assert.NoError(t, err, "Should download file")
}
