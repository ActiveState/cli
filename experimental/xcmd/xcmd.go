package main

import (
	"os"
	"os/exec"
	"strings"
)

type xcmd struct {
	*exec.Cmd
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
		Cmd: cmd,
	}

	return c, nil
}

func (c *xcmd) close() error {
	return c.Cmd.Process.Kill()
}

func (c *xcmd) wait() error {
	if err := c.Wait(); err != nil {
		if !strings.Contains(err.Error(), "killed") {
			return err
		}
	}
	return nil
}
