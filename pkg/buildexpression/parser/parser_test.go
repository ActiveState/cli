package parser

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/assert"
)

func TestPaser_Parse(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err)

	testData, err := fileutils.ReadFile(filepath.Join(root, "pkg", "buildexpression", "testdata", "buildexpression.json"))
	assert.NoError(t, err)

	p, err := New(testData)
	assert.NoError(t, err)
	_, err = p.Parse()
	assert.NoError(t, err)
}

func TestPaser_Parse_Complex(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err)

	testData, err := fileutils.ReadFile(filepath.Join(root, "pkg", "buildexpression", "testdata", "buildexpression-complex.json"))
	assert.NoError(t, err)

	p, err := New(testData)
	assert.NoError(t, err)
	_, err = p.Parse()
	assert.NoError(t, err)
}
