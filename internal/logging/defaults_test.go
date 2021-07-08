package logging_test

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/stretchr/testify/assert"
)

func TestFilePathForCmd(t *testing.T) {
	filename := logging.FileNameForCmd("cmd-name", 123)
	path := logging.FilePathForCmd("cmd-name", 123)
	assert.NotEqual(t, filename, path)
	assert.True(t, strings.HasSuffix(path, logging.FileNameForCmd("cmd-name", 123)))
}
