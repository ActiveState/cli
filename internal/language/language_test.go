package language

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestLanguage(t *testing.T) {
	assert.Empty(t, Bash.Executable().Name())
	assert.True(t, Bash.Executable().Builtin())
	assert.False(t, Python3.Executable().Builtin())

	var l Language
	err := yaml.Unmarshal([]byte("junk"), &l)
	assert.Error(t, err, "fail due to bad yaml input")

	err = yaml.Unmarshal([]byte("perl"), &l)
	assert.NoError(t, err, "successfully unmarshal 'perl'")
	assert.Equal(t, l, Perl)

	l = Batch
	bs, err := yaml.Marshal(&l)
	assert.NoError(t, err, "successfully marshal 'batch'")
	assert.Equal(t, "batch\n", string(bs))
}

func TestMakeLanguage(t *testing.T) {
	assert.Equal(t, Python3, makeByName("python3"), "python3")
	assert.Equal(t, Unknown, makeByName("python4"), "unknown language")
}

func TestMakeLanguageByShell(t *testing.T) {
	assert.Equal(t, Batch, MakeByShell("cmd.exe"), "strings with 'cmd' return batch")
	assert.Equal(t, Bash, MakeByShell("anything_else"), "anything else returns bash")
}
