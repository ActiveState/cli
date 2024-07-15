package main

import (
	"fmt"
	"os"
)

func runCmd(meta *executorMeta) (int, error) {
	userArgs := os.Args[1:]
	cmd := Command(meta.MatchingBin, userArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = meta.TransformedEnv

	if err := cmd.Run(); err != nil {
		return -1, fmt.Errorf("command %q failed: %w", meta.MatchingBin, err)
	}

	return cmd.ProcessState.ExitCode(), nil
}
