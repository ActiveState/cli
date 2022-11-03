package main

import (
	"fmt"
	"os"
	"os/exec"
)

func runCmd(meta *executorMeta) error {
	userArgs := os.Args[1:]
	cmd := exec.Command(meta.MatchingBin, userArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = meta.TransformedEnv

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %q failed: %w", meta.MatchingBin, err)
	}

	return nil
}
