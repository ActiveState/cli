package main

import (
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

	return cmd.Run()
}
