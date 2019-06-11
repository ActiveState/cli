package projectfile

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var exampleYAML = []byte(`
junk: xgarbage
project: https://example.com/xowner/xproject?commitID=12345
12345: xvalue
`)

func TestYAMLReader(t *testing.T) {
	buf := bytes.NewBuffer(exampleYAML)
	yr := yamlReader{buf}

	_, fail := yr.replaceInValue("", "a", "b")
	assert.Error(t, fail.ToError())
	_, fail = yr.replaceInValue(ProjectKey, "", "b")
	assert.Error(t, fail.ToError())
	_, fail = yr.replaceInValue(ProjectKey, "a", "")
	assert.Error(t, fail.ToError())

	outputYAML := bytes.Replace(exampleYAML, []byte("12345"), []byte("987"), 1) // must be 1

	r, fail := yr.replaceInValue(ProjectKey, "12345", "987")
	assert.NoError(t, fail.ToError())

	out := &bytes.Buffer{}
	_, err := out.ReadFrom(r)
	require.NoError(t, err)
	assert.Equal(t, outputYAML, out.Bytes())
}
