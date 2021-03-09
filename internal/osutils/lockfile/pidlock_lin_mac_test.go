// +build linux darwin

package lockfile

import (
	"os"
	"os/exec"
	"testing"
)

func prepLockCmd(lockCmd *exec.Cmd) *exec.Cmd {
	return lockCmd
}

func interruptProcess(t *testing.T, p *os.Process) {
	err := p.Signal(os.Interrupt)
	if err != nil {
		t.Fatalf("Failed sending interrupt to process: %v", err)
	}
}
