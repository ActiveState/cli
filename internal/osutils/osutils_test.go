package osutils

import (
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestCmdExitCode(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "TestCmdExitCode")
	if runtime.GOOS != "windows" {
		assert.NoError(t, err)
		tmpfile.WriteString("#!/usr/bin/env bash\n")
		tmpfile.WriteString("exit 255")
		tmpfile.Close()
	} else {
		tmpfile.WriteString("echo off\n")
		tmpfile.WriteString("exit 255")
		tmpfile.Close()
		err = os.Rename(tmpfile.Name(), tmpfile.Name()+".bat")
		assert.NoError(t, err)
	}
	os.Chmod(tmpfile.Name(), 0755)

	cmd := exec.Command(tmpfile.Name())
	err = cmd.Run()
	assert.Error(t, err)
	assert.Equal(t, 255, CmdExitCode(cmd), "Exits with code 255")
}

func TestBashifyPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skipf("Bashify path only runs on windows")
	}
	bashify := func(value string) string {
		result, err := BashifyPath(value)
		require.NoError(t, err)
		return result
	}
	res := bashify(`C:\temp`)
	assert.True(t, strings.HasSuffix(res, "/c/temp"), "Expected suffix '/c/temp', got %s", res)
	res = bashify(`C:\temp temp`)
	assert.True(t, strings.HasSuffix(res, "/c/temp\\ temp"), "Expected suffix 'c/temp\\ temp', got %s", res)
	assert.Equal(t, "/foo", bashify(`/foo`))

	_, err := BashifyPath("not a valid path")
	require.Error(t, err)
	_, err = BashifyPath("../relative/path")
	require.Error(t, err, "Relative paths should not work")
}

func TestEnvSliceToMap(t *testing.T) {
	tests := []struct {
		name     string
		envSlice []string
		want     map[string]string
	}{
		{
			"Env slice is converted to map",
			[]string{
				"foo=bar",
				"PATH=blah:blah",
				"_=",
			},
			map[string]string{
				"foo":  "bar",
				"PATH": "blah:blah",
				"_":    "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EnvSliceToMap(tt.envSlice); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EnvSliceToMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvMapToSlice(t *testing.T) {
	tests := []struct {
		name   string
		envMap map[string]string
		want   []string
	}{
		{
			"Env map is converted to slice",
			map[string]string{
				"foo":  "bar",
				"PATH": "blah:blah",
				"_":    "",
			},
			[]string{
				"foo=bar",
				"PATH=blah:blah",
				"_=",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnvMapToSlice(tt.envMap)

			sort.Strings(got)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EnvMapToSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutableName(t *testing.T) {
	name := ExecutableName()
	if fileutils.TargetExists(name) {
		t.Fatalf("Executable name should return a filename, not a filepath. Returned: %s", name)
	}
}