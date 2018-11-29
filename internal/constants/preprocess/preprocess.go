package preprocess

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
)

// Constants holds constants that will be preprocessed, meaning the key value parts here will be built into the constants
// package as actual constants, using the build-time interpretations
var Constants = map[string]func() string{}

func init() {
	Constants["BranchName"] = func() string { return getCmdOutput("git rev-parse --abbrev-ref HEAD") }
	Constants["BuildNumber"] = func() string {
		out := getCmdOutput("git rev-list --abbrev-commit HEAD")
		return strconv.Itoa(len(strings.Split(out, "\n")))
	}
	Constants["RevisionHash"] = func() string { return getCmdOutput("git rev-parse --verify HEAD") }
	Constants["Version"] = func() string { return fmt.Sprintf("%s-%s", constants.VersionNumber, Constants["BuildNumber"]()) }
	Constants["Date"] = func() string { return time.Now().Format("Mon Jan 2 2006 15:04:05 -0700 MST") }
	Constants["EnvName"] = func() string { return strings.TrimSpace(os.Getenv("CLIENV")) }
}

func getCmdOutput(cmdString string) string {
	cmdArgs := strings.Split(cmdString, " ")

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		log.Fatalf("Command failed, command: %s, args: %v, output: %s, error: %s, code: %s", cmdArgs[0], cmdArgs[1:], out.String(), stderr.String(), err)
	}
	return strings.Trim(out.String(), "\n")
}
