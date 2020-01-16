package preprocess

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/blang/semver"
	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
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
	Constants["RevisionHashShort"] = func() string { return getCmdOutput("git rev-parse --short " + branchNameFull) }
	Constants["Version"] = func() string { return getVersion(branchName) }
	Constants["Date"] = func() string { return time.Now().Format("Mon Jan 2 2006 15:04:05 -0700 MST") }
	Constants["UserAgent"] = func() string {
		return fmt.Sprintf("%s/%s; %s", constants.CommandName, Constants["Version"](), branchName)
	}
	Constants["APITokenName"] = func() string { return fmt.Sprintf("%s-%s", constants.APITokenNamePrefix, branchName) }
}

// gitBranchName returns the branch name of the current git commit / PR
func gitBranchName() string {
	// branch name variable set by Azure CI during pull request
	if branch, isset := os.LookupEnv("SYSTEM_PULLREQUEST_SOURCEBRANCH"); isset {
		return "origin/" + branch
	}
	// branch name variable set by Azure CI
	if branch, isset := os.LookupEnv("BUILD_SOURCEBRANCHNAME"); isset {
		return "origin/" + branch
	}
	branch := getCmdOutput("git rev-parse --abbrev-ref HEAD")
	return branch
}

// branchName returns the release name and the branch name it is generated from
// Usually the release name is identical to the branch name, unless environment variable
// `BRANCH_OVERRIDE` is set
func branchName() (string, string) {
	branch := gitBranchName()
	releaseName := branch

	if releaseOverride, isset := os.LookupEnv("BRANCH_OVERRIDE"); isset {
		releaseName = releaseOverride
	}
	if strings.Contains(releaseName, "/") {
		releaseName = strings.Split(releaseName, "/")[1]
	}

	return releaseName, branch

}

func getVersion(branchName string) string {
	output := getCmdOutput("state --version")
	versionString := strings.Split(strings.TrimSpace(output), "\n")[0]
	versionNumber := strings.Split(strings.TrimSpace(versionString), " ")
	masterVersion := versionNumber[len(versionNumber)-1]

	currentSemver, err := semver.Parse(masterVersion)
	if err != nil {
		log.Fatalf("Failed to create semver from version string: %s", err)
	}

	prVersion, err := semver.NewPRVersion((Constants["RevisionHashShort"]()))
	if err != nil {
		log.Fatalf("Could not create pre-release version number: %v", err)
	}
	currentSemver.Pre = []semver.PRVersion{prVersion}

	// TODO: Maybe update this to be a verification step
	// Deals with if we are on CI, if there is a pull request,
	// and if there are labels
	if !onCI() {
		return currentSemver.String()
	}

	label := getVersionLabel(branchName)
	switch {
	case label == "version: patch":
		currentSemver.Patch++
	case label == "version: minor":
		currentSemver.Minor++
		currentSemver.Patch = 0
	case label == "version: major":
		currentSemver.Major++
		currentSemver.Minor = 0
		currentSemver.Patch = 0
	default:
		log.Fatalf("Encountered an unexepected Github PR label: %s", label)
	}
	if err != nil {
		log.Fatalf("Could not increment version number: %s", err)
	}

	return currentSemver.String()
}

func onCI() bool {
	// Check if we are running on CircleCI
	if os.Getenv("CIRCLE_JOB") != "" {
		return true
	}
	// Check if we are running on Azure
	if os.Getenv("CI") == "azure" {
		return true
	}
	return false
}

func getVersionLabel(branchName string) string {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		// TODO: Get access token from environment on CI. This will not be called
		// if we are not on CI so it shouldn't fail in that case
		&oauth2.Token{AccessToken: "36ed5dfe82d00e0b874dec5ee434f03e407d73ca"},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// TODO: What if we are on CI and there isn't a PR yet? Or the case for the master branch
	pullRequests, _, err := client.PullRequests.List(ctx, "MDrakos", "gcd-calculator-gui", &github.PullRequestListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	// TODO: What if we are in a PR but missing a label
	var versionLabel string
	for _, pr := range pullRequests {
		branchLabel := strings.Split(pr.GetHead().GetLabel(), ":")
		if len(branchLabel) < 2 {
			continue
		}
		prBranchName := branchLabel[1]

		if prBranchName == branchName {
			if len(pr.Labels) > 1 {
				log.Fatalf("More than one PR label found %v", pr.Labels)
			}
			versionLabel = *pr.Labels[0].Name
			break
		}
	}

	return versionLabel
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
