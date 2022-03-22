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

	assert.Equal(t, "dev", project.Environments) // should be overridden

	debugConstantsSeen := 0
	macOSConstantSeen := true
	assert.True(t, len(project.Constants) > 2) // these constants should not override the others
	for _, constant := range project.Constants {
		switch constant.Name {
		case "DEBUG":
			// We are merging with mergo's WithAppendSlice, so there will be two variables with this name.
			// However, we expect the one from activestate.macos.yaml to show up first.
			if debugConstantsSeen == 0 {
				assert.Equal(t, "false", constant.Value) // from activestate.macos.yaml
			} else {
				assert.Equal(t, "true", constant.Value) // from activestate.yaml
			}
			debugConstantsSeen++
		case "macOS":
			assert.Equal(t, "true", constant.Value) // should be added
			macOSConstantSeen = true
		}
	}
	assert.Equal(t, 2, debugConstantsSeen)
	assert.True(t, macOSConstantSeen)
}
