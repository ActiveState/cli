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
	"github.com/ActiveState/cli/internal/constants/version"
	"github.com/blang/semver"
)

// Constants holds constants that will be preprocessed, meaning the key value parts here will be built into the constants
// package as actual constants, using the build-time interpretations
var Constants = map[string]func() string{}

func init() {
	branchName, commitRef := branchName()
	buildNumber := buildNumber()

	if sha, exists := os.LookupEnv("GITHUB_SHA"); exists {
		commitRef = sha
	}

	newVersion, err := version.ParseVersion(buildEnvironment())
	if err != nil {
		log.Fatalf("Could not parse new version: %s", err)
	}

	Constants["BranchName"] = func() string { return branchName }
	Constants["BuildNumber"] = func() string { return buildNumber }
	Constants["RevisionHash"] = func() string { return getCmdOutput("git rev-parse --verify " + commitRef) }
	Constants["RevisionHashShort"] = func() string { return getCmdOutput("git rev-parse --short " + commitRef) }
	Constants["Version"] = func() string { return mustVersionWithRevision(newVersion, Constants["RevisionHashShort"]()) }
	Constants["VersionNumber"] = func() string { return newVersion.String() }
	Constants["Date"] = func() string { return time.Now().Format(constants.DateTimeFormatRecord) }
	Constants["UserAgent"] = func() string {
		return fmt.Sprintf("%s/%s; %s", constants.CommandName, Constants["Version"](), branchName)
	}
	Constants["APITokenName"] = func() string { return fmt.Sprintf("%s-%s", constants.APITokenNamePrefix, branchName) }
}

// gitBranchName returns the branch name of the current git commit / PR
func gitBranchName() string {
	// branch name variable set by Github Actions
	if branch, isset := os.LookupEnv("GITHUB_HEAD_REF"); isset && branch != "" {
		return "origin/" + branch
	}
	if branch, isset := os.LookupEnv("GITHUB_REF"); isset && branch != "" {
		return "origin/" + strings.TrimPrefix(branch, "refs/heads/")
	}
	branch := getCmdOutput("git rev-parse --abbrev-ref HEAD")
	return branch
}

// branchName returns the release name and the branch name it is generated from
// Usually the release name is identical to the branch name, unless environment variable
// `BRANCH_OVERRIDE` is set
func branchName() (string, string) {
	branch := gitBranchName()
	releaseName := strings.TrimPrefix(branch, "origin/")

	return releaseName, branch
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
		os.Exit(1)
	}
	return strings.Trim(out.String(), "\n")
}

func buildEnvironment() version.Env {
	if !onCI() {
		return version.LocalEnv
	}

	return version.RemoteEnv
}

func onCI() bool {
	if os.Getenv("CI") != "" {
		return true
	}
	return false
}

func mustVersionWithRevision(ver *semver.Version, revision string) string {
	v, err := version.VersionWithRevision(ver, revision)
	if err != nil {
		log.Fatalf("failed to add")
	}

	return v.String()
}
