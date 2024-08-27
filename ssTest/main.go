package main

import (
	"fmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/subshell"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	fmt.Println("Setting up configuration")
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not create configuration")
	}

	fmt.Println("Detecting shell")
	shell, path := subshell.DetectShell(cfg)
	fmt.Println("Found shell:", shell)
	fmt.Println("Path:", path)

	return nil
}
