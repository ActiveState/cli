package projectfile

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestYamlMerge(t *testing.T) {
	rootpath, err := environment.GetRootPath()
	if err != nil {
		t.Fatal(err)
	}

	project, err := Parse(filepath.Join(rootpath, "pkg", "projectfile", "testdata", "activestate.yaml"))
	assert.NoError(t, err, "Should not throw an error")

	assert.Equal(t, "dev,qa,prod", project.Environment) // should not be overridden by macOS's environment

	assert.True(t, len(project.Constants) > 2) // should not be overridden by macOS's constants
	for _, constant := range project.Constants {
		switch constant.Name {
		case "DEBUG":
			assert.Equal(t, "true", constant.Value) // should not be overridden by macOS's 'false' value
		case "PYTHONPATH":
			assert.NotEmpty(t, constant.Value)
		case "macOS":
			assert.Fail(t, "macOS constant should not have been merged from activestate.macos.yaml")
		}
	}
}
