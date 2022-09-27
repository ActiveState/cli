package main

import (
	"os"
	"os/exec"
)

func runCmd(meta *executorMeta, userArgs []string) error {
	cmd := exec.Command(meta.MatchingBin, userArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = meta.TransformedEnv

	return cmd.Run()
}
