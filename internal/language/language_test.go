package language

import (
	"testing"

	"github.com/stretchr/testify/require"

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
	assert.Empty(t, l.Header())

	l = Perl
	assert.Equal(t, "#!/usr/bin/env perl\n", l.Header())
}

func TestMakeLanguage(t *testing.T) {
	assert.Equal(t, Python3, MakeByName("python3"), "python3")
	assert.Equal(t, Unknown, MakeByName("python4"), "unknown language")
	assert.Equal(t, Unset, MakeByName(""), "unset language")
}

func TestUnmarshal(t *testing.T) {
	var unmarshal Language
	yaml.Unmarshal([]byte(`python3`), &unmarshal)
	assert.Equal(t, Python3, unmarshal)
}

func TestMarshal(t *testing.T) {
	var marshal = Python3
	out, err := yaml.Marshal(marshal)
	require.NoError(t, err)
	assert.Contains(t, string(out), Python3.data().name)
}

func TestMakeLanguageByShell(t *testing.T) {
	assert.Equal(t, Batch, MakeByShell("cmd.exe"), "strings with 'cmd' return batch")
	assert.Equal(t, Bash, MakeByShell("anything_else"), "anything else returns bash")
}

func TestAvailable(t *testing.T) {
	langs := Available()
	for _, l := range langs {
		assert.False(t, l.Executable().Builtin())
		assert.NotEmpty(t, l.Executable().Name())
	}
}
