package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/svcctl/svcmsg"
)

func newAttempt() (*svcmsg.Attempt, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot get executable info: %w", err)
	}
	attempt := svcmsg.NewAttempt(execPath)

	return attempt, nil
}
