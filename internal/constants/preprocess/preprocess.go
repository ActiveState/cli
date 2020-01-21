package preprocess

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
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
	Constants["Version"] = func() string { return getVersion(branchName, true) }
	Constants["VersionNumber"] = func() string { return getVersion(branchName, false) }
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

func getVersion(branchName string, preRelease bool) string {
	output := getCmdOutput("state --version")
	versionString := strings.Split(strings.TrimSpace(output), "\n")[0]
	versionNumber := strings.Split(strings.TrimSpace(versionString), " ")
	masterVersion := versionNumber[len(versionNumber)-1]

	currentSemver, err := semver.Parse(masterVersion)
	if err != nil {
		log.Fatalf("Failed to create semver from version string: %s", err)
	}

	if preRelease {
		prVersion, err := semver.NewPRVersion((Constants["RevisionHashShort"]()))
		if err != nil {
			log.Fatalf("Could not create pre-release version number: %v", err)
		}
		currentSemver.Pre = []semver.PRVersion{prVersion}
	} else {
		currentSemver.Pre = nil
	}

	if !onCI() {
		return currentSemver.String()
	}

	label := getVersionLabel(branchName)
	switch label {
	case "version: patch":
		currentSemver.Patch++
	case "version: minor":
		currentSemver.Minor++
		currentSemver.Patch = 0
	case "version: major":
		currentSemver.Major++
		currentSemver.Minor = 0
		currentSemver.Patch = 0
	case Constants["BranchName"]():
		// We are on a branch and there is no PR associated. Use current version.
	default:
		log.Fatalf("Encountered an unexepected Github PR label: %s", label)
	}
	if err != nil {
		log.Fatalf("Could not increment version number: %s", err)
	}

	return currentSemver.String()
}

func onCI() bool {
	if os.Getenv("CI") != "" {
		return true
	}
	return false
}

func getVersionLabel(branchName string) string {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_REPO_TOKEN")},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	client := github.NewClient(tc)

	if branchName == "master" {
		return getVersionLabelMaster(client)
	}

	return getVersionLabelPR(client)
}

func getVersionLabelMaster(client *github.Client) string {
	pullReqests, _, err := client.PullRequests.List(context.Background(), "ActiveState", constants.LibraryName, &github.PullRequestListOptions{State: "closed", Sort: "updated", Direction: "desc"})
	if err != nil {
		log.Fatalf("Could not list pull requests: %v", err)
	}

	var versionLabel string
	for _, pullRequest := range pullReqests {
		if isMerged(*pullRequest.Number, client) {
			if len(pullRequest.Labels) != 1 {
				log.Fatalf("Pull reqests must have one label")
			}

			versionLabel = *pullRequest.Labels[0].Name
			break
		}
	}
	if versionLabel == "" {
		log.Fatal("No version label from latest merged pull request")
	}

	return versionLabel
}

func isMerged(number int, client *github.Client) bool {
	merged, _, err := client.PullRequests.IsMerged(context.Background(), "ActiveState", constants.LibraryName, number)
	if err != nil {
		log.Fatalf("Could not confirm pull request #%d has been merged: %v", number, err)
	}
	return merged
}

func getVersionLabelPR(client *github.Client) string {
	prNumber := getPRNumber()
	if prNumber == -1 {
		return Constants["BranchName"]()
	}

	pullRequest, _, err := client.PullRequests.Get(context.Background(), "ActiveState", constants.LibraryName, prNumber)
	if err != nil {
		log.Fatal(err)
	}
	if len(pullRequest.Labels) != 1 {
		log.Fatalf("Pull requests must have one label")
	}

	versionLabel := *pullRequest.Labels[0].Name
	if versionLabel == "" {
		log.Fatal("No version label associated with this branch")
	}

	return versionLabel
}

func getPRNumber() int {
	// CircleCI
	prInfo := os.Getenv("CI_PULL_REQUEST")
	if prInfo != "" {
		return getPRNumberCircle(prInfo)
	}

	// Azure
	prInfo = os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTNUMBER")
	if prInfo != "" {
		return getPRNumberAzure(prInfo)
	}

	// Pull request info not set, we are on a branch but no PR has been created
	return -1
}

func getPRNumberCircle(info string) int {
	info = strings.TrimPrefix(info, "https://github.com/ActiveState/cli/pull/")
	prNumber, err := strconv.Atoi(info)
	if err != nil {
		log.Fatalf("Could not convert pull request number: %v", err)
	}
	return prNumber
}

func getPRNumberAzure(info string) int {
	prNumber, err := strconv.Atoi(info)
	if err != nil {
		log.Fatalf("Could not convert pull request number: %v", err)
	}
	return prNumber
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
