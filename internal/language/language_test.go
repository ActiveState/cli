package language

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestLanguage(t *testing.T) {
	var l Language
	assert.Equal(t, Unset, l)

	assert.True(t, Bash.Executable().CanUseThirdParty())

	assert.NotEmpty(t, Python3.Executable().Name())
	assert.False(t, Python3.Executable().CanUseThirdParty())

	assert.Equal(t, "#!/usr/bin/env perl\n", Perl.Header())
}

func TestMakeLanguage(t *testing.T) {
	assert.Equal(t, Python3, MakeByName("python3"), "python3")
	assert.Equal(t, Unknown, MakeByName("python4"), "unknown language")
	assert.Equal(t, Unset, MakeByName(""), "unset language")
}

func TestUnmarshal(t *testing.T) {
	var l Language

	err := yaml.Unmarshal([]byte("junk"), &l)
	assert.Error(t, err, "fail due to bad yaml input")
	assert.Equal(t, Unset, l)

	err = yaml.Unmarshal([]byte("python3"), &l)
	assert.NoError(t, err, "successfully unmarshal 'python3'")
	assert.Equal(t, Python3, l)

	err = yaml.Unmarshal([]byte("bash"), &l)
	assert.NoError(t, err, "successfully unmarshal 'bash'")
	assert.Equal(t, Bash, l)

	err = yaml.Unmarshal([]byte("unknown"), &l)
	assert.Error(t, err, "not successfully unmarshal 'unknown'")
}

func TestMarshal(t *testing.T) {
	l := Python3
	bs, err := yaml.Marshal(l)
	require.NoError(t, err)
	assert.Contains(t, string(bs), "python")

	l = Batch
	bs, err = yaml.Marshal(&l)
	require.NoError(t, err)
	assert.Contains(t, string(bs), "batch")
	assert.Empty(t, l.Header())

}

func TestMakeLanguageByShell(t *testing.T) {
	assert.Equal(t, Batch, MakeByShell("cmd.exe"), "strings with 'cmd' return batch")
	assert.Equal(t, Bash, MakeByShell("anything_else"), "anything else returns bash")
}

func TestRecognized(t *testing.T) {
	langs := Recognized()
	for _, l := range langs {
		assert.NotEqual(t, l, Unset, "not unset")
		assert.NotEqual(t, l, Unknown, "not unknown")
	}
}

func TestSupported(t *testing.T) {
	var l Supported
	assert.Equal(t, Unset, l.Language)
}

func TestSupportedUnmarshal(t *testing.T) {
	var l Supported

	err := yaml.Unmarshal([]byte("junk"), &l)
	assert.Error(t, err, "fail due to bad yaml input")
	assert.Equal(t, Unset, l.Language)

	err = yaml.Unmarshal([]byte("python3"), &l)
	assert.NoError(t, err, "successfully unmarshal 'python3'")
	assert.Equal(t, Python3, l.Language)

	err = yaml.Unmarshal([]byte("bash"), &l)
	assert.Error(t, err, "not successfully unmarshal 'bash'")
}

func TestSupportedMarshal(t *testing.T) {
	l := Supported{Python3}
	bs, err := yaml.Marshal(l)
	require.NoError(t, err)
	assert.Contains(t, string(bs), "python")

	l = Supported{Batch}
	bs, err = yaml.Marshal(&l)
	require.NoError(t, err)
	assert.Contains(t, string(bs), "batch")
	assert.Empty(t, l.Header())

}

func TestRecognizedSupporteds(t *testing.T) {
	langs := RecognizedSupporteds()
	for _, l := range langs {
		assert.NotEqual(t, l.Language, Unset, "not unset")
		assert.NotEqual(t, l.Language, Unknown, "not unknown")
		assert.False(t, l.Executable().CanUseThirdParty())
		assert.NotEmpty(t, l.Executable().Name())
	}
}

func TestMakeByNameAndVersion(t *testing.T) {
	type args struct {
		name    string
		version string
	}
	tests := []struct {
		name    string
		args    args
		want    Language
		wantErr bool
	}{
		{
			"Valid Python3 version",
			args{"python", "3.6.6"},
			Python3,
			false,
		},
		{
			"Valid Python2 version",
			args{"python", "2.7.18"},
			Python2,
			false,
		},
		{
			"Valid Python2 invalid patch",
			args{"python", "2.7.18.1"},
			Python2,
			false,
		},
		{
			"Valid Python3 invalid patch",
			args{"python", "3.9"},
			Python3,
			false,
		},
		{
			"Missing version",
			args{"python", ""},
			Python3,
			false,
		},
		{
			"Valid Perl version",
			args{"perl", "5.28.1"},
			Perl,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MakeByNameAndVersion(tt.args.name, tt.args.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeByNameAndVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MakeByNameAndVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
