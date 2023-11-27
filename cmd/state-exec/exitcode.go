package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ActiveState/cli/internal/svcctl/svcmsg"
)

func newExitCodeMessage(exitCode int) (*svcmsg.ExitCode, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot get executable info: %w", err)
	}
	return &svcmsg.ExitCode{execPath, strconv.Itoa(exitCode)}, nil
}
