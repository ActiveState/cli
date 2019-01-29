package osutils

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmdExitCode(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "TestCmdExitCode")
	assert.NoError(t, err)

	tmpfile.WriteString("#!/usr/bin/env bash\n")
	tmpfile.WriteString("exit 255")
	tmpfile.Close()
	os.Chmod(tmpfile.Name(), 0755)

	cmd := exec.Command(tmpfile.Name())
	err = cmd.Run()
	assert.Error(t, err)
	assert.Equal(t, 255, CmdExitCode(cmd), "Exits with code 255")
}
