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
	branchName, branchNameFull := branchName()
	buildNumber := buildNumber(branchNameFull)

	Constants["BranchName"] = func() string { return branchName }
	Constants["BuildNumber"] = func() string { return buildNumber }
	Constants["RevisionHash"] = func() string { return getCmdOutput("git rev-parse --verify " + branchNameFull) }
	Constants["Version"] = func() string { return fmt.Sprintf("%s-%s", constants.VersionNumber, buildNumber) }
	Constants["Date"] = func() string { return time.Now().Format("Mon Jan 2 2006 15:04:05 -0700 MST") }
	Constants["APIEnv"] = func() string { return strings.TrimSpace(os.Getenv("APIENV")) }
	Constants["UserAgent"] = func() string {
		return fmt.Sprintf("%s/%s; %s; %s", constants.CommandName, Constants["Version"](), branchNameFull, Constants["APIEnv"]())
	}
}

func branchName() (string, string) {
	if branch, isset := os.LookupEnv("BRANCH_OVERRIDE"); isset {
		return branch, branch
	}
	if branch, isset := os.LookupEnv("SYSTEM_PULLREQUEST_SOURCEBRANCH"); isset {
		return branch, "origin/" + branch
	}
	branch := getCmdOutput("git rev-parse --abbrev-ref HEAD")
	return branch, branch
}

func buildNumber(branchName string) string {
	out := getCmdOutput("git rev-list --abbrev-commit " + branchName)
	return strconv.Itoa(len(strings.Split(out, "\n")))
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
