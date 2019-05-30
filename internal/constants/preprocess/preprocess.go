package preprocess

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
)

// Constants holds constants that will be preprocessed, meaning the key value parts here will be built into the constants
// package as actual constants, using the build-time interpretations
var Constants = map[string]func() string{}

func init() {
	branchName, branchNameFull := branchName()
	buildNumber := buildNumber()

	Constants["BranchName"] = func() string { return branchName }
	Constants["BuildNumber"] = func() string { return buildNumber }
	Constants["RevisionHash"] = func() string { return getCmdOutput("git rev-parse --verify " + branchNameFull) }
	Constants["Version"] = func() string { return fmt.Sprintf("%s-%s", constants.VersionNumber, buildNumber) }
	Constants["Date"] = func() string { return time.Now().Format("Mon Jan 2 2006 15:04:05 -0700 MST") }
	Constants["APIEnv"] = func() string { return strings.TrimSpace(os.Getenv("APIENV")) }
	Constants["UserAgent"] = func() string {
		return fmt.Sprintf("%s/%s; %s; %s", constants.CommandName, Constants["Version"](), branchName, Constants["APIEnv"]())
	}
	Constants["APITokenName"] = func() string { return fmt.Sprintf("%s-%s", constants.APITokenNamePrefix, branchName) }
}

func branchName() (string, string) {
	if branch, isset := os.LookupEnv("BRANCH_OVERRIDE"); isset {
		if strings.Contains(branch, "/") {
			return strings.Split(branch, "/")[1], branch
		}
		return branch, branch
	}
	if branch, isset := os.LookupEnv("SYSTEM_PULLREQUEST_SOURCEBRANCH"); isset {
		return branch, "origin/" + branch
	}
	if branch, isset := os.LookupEnv("BUILD_SOURCEBRANCHNAME"); isset {
		return branch, "origin/" + branch
	}
	branch := getCmdOutput("git rev-parse --abbrev-ref HEAD")
	return branch, branch
}

func buildNumber() string {
	out := getCmdOutput("git rev-list --all --count")
	return strings.TrimSpace(out)
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
