package osutils

import (
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/logging"

	"github.com/ActiveState/cli/internal/testhelpers/osutil"

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

func TestExecuteAndPipeStd(t *testing.T) {
	out, err := osutil.CaptureStdout(func() {
		logging.SetLevel(logging.NOTHING)
		defer logging.SetLevel(logging.NORMAL)
		ExecuteAndPipeStd("printenv", []string{"FOO"}, []string{"FOO=--out--"})
	})
	require.NoError(t, err)
	assert.Equal(t, "--out--\n", out, "captures output")
}
