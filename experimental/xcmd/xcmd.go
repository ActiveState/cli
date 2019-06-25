package main

import (
	"os"
	"os/exec"
	"syscall"
)

type xcmd struct {
	cmd *exec.Cmd
}

func newXCmd() (*xcmd, error) {
	cmd := exec.Command("/bin/bash")

	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	c := &xcmd{
		cmd: cmd,
	}

	return c, nil
}

func (c *xcmd) close() error {
	return c.cmd.Process.Signal(syscall.SIGTERM)
}

func (c *xcmd) wait() error {
	if err := c.cmd.Wait(); err != nil {
		if eerr, ok := err.(*exec.ExitError); ok {
			if eerr.Exited() && eerr.ExitCode() == -1 {
				return nil
			}
			return eerr
		}
	}
	return nil
}
