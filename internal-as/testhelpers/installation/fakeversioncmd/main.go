package main

import (
	"fmt"

	"github.com/ActiveState/cli/internal/constants"
)

// This command is used in our test-updates as the State Tool executable.
// It simply returns a faked version and channel name on every invocation, so
// we can ensure that the installation/update was indeed successful.

var version string = "99.99.9999"
var channel string // can be set through linker flag -ldflags "-X main.channel=test-channel"

func main() {
	if channel == "" {
		channel = constants.BranchName
	}
	fmt.Printf(`{"version": "%s", "branch": "%s"}`, version, channel)
}
