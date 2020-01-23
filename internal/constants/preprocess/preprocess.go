package preprocess

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
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

const (
	unknownEnv = iota
	localEnv
	masterEnv
	branchEnv
	pullRequestEnv
)

const (
	patch = "version: patch"
	minor = "version: minor"
	major = "version: major"
)

func init() {
	branchName, branchNameFull := branchName()
	buildNumber := buildNumber()

	Constants["BranchName"] = func() string { return branchName }
	Constants["BuildNumber"] = func() string { return buildNumber }
	Constants["RevisionHash"] = func() string { return getCmdOutput("git rev-parse --verify " + branchNameFull) }
	Constants["RevisionHashShort"] = func() string { return getCmdOutput("git rev-parse --short " + branchNameFull) }
	Constants["Version"] = func() string { return getVersionPreRelease(branchName, getCurrentVersionPreRelease()) }
	Constants["VersionNumber"] = func() string { return getVersion(branchName, getCurrentVersion()) }
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

func getVersion(branchName string, current *semver.Version) string {
	return updateVersion(current, branchName)
}

func getVersionPreRelease(branchName string, current *semver.Version) string {
	return updateVersion(current, branchName)
}

func getCurrentVersion() *semver.Version {
	output := getCmdOutput(fmt.Sprintf("%s --version", constants.CommandName))
	regex := regexp.MustCompile("\\d+\\.\\d+\\.\\d+-[a-f0-9]+")
	match := regex.FindString(output)
	if match == "" {
		log.Fatal("Could not determine current version")
	}

	currentSemver, err := semver.New(match)
	if err != nil {
		log.Fatalf("Failed to create semver from version string: %s", err)
	}
	currentSemver.Pre = nil

	return currentSemver
}

func getCurrentVersionPreRelease() *semver.Version {
	version := getCurrentVersion()

	prVersion, err := semver.NewPRVersion((Constants["RevisionHashShort"]()))
	if err != nil {
		log.Fatalf("Could not create pre-release version number: %v", err)
	}
	version.Pre = []semver.PRVersion{prVersion}

	return version
}

func updateVersion(current *semver.Version, branchName string) string {
	state := buildEnvironment(branchName)
	switch state {
	case localEnv, branchEnv:
		return current.String()
	case masterEnv, pullRequestEnv:
		return incrementVersion(branchName, current)
	default:
		log.Fatalf("Build state is not local, remote branch, remote master, or pull request")
	}

	return ""
}

func buildEnvironment(branchName string) int {
	if !onCI() {
		return localEnv
	}

	if branchName == "master" {
		return masterEnv
	}

	prNum := getPRNumber()
	if prNum == 0 {
		return branchEnv
	}

	return pullRequestEnv
}

func onCI() bool {
	if os.Getenv("CI") != "" {
		return true
	}
	return false
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
	return 0
}

func incrementVersion(branchName string, current *semver.Version) string {
	label := getVersionLabel(branchName)
	switch label {
	case patch:
		current.Patch++
	case minor:
		current.Minor++
		current.Patch = 0
	case major:
		current.Major++
		current.Minor = 0
		current.Patch = 0
	default:
		log.Fatalf("Encountered an unexepected Github label: %s", label)
	}

	return current.String()
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
	pullReqests, _, err := client.PullRequests.List(context.Background(), constants.LibraryOwner, constants.LibraryName, &github.PullRequestListOptions{State: "closed", Sort: "updated", Direction: "desc"})
	if err != nil {
		log.Fatalf("Could not list pull requests: %v", err)
	}

	var versionLabel string
	for _, pullRequest := range pullReqests {
		if isMerged(*pullRequest.Number, client) {
			label := getLabel(pullRequest.Labels)
			if label == "" {
				log.Fatalf("Pull request does not have version label")
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
	merged, _, err := client.PullRequests.IsMerged(context.Background(), constants.LibraryOwner, constants.LibraryName, number)
	if err != nil {
		log.Fatalf("Could not confirm pull request #%d has been merged: %v", number, err)
	}
	return merged
}

func getVersionLabelPR(client *github.Client) string {
	pullRequest, _, err := client.PullRequests.Get(context.Background(), constants.LibraryOwner, constants.LibraryName, getPRNumber())
	if err != nil {
		log.Fatal(err)
	}

	label := getLabel(pullRequest.Labels)
	target := strings.TrimPrefix(pullRequest.GetBase().GetLabel(), fmt.Sprintf("%s:", constants.LibraryName))
	if target != "master" && label == "" {
		return patch
	}

	if label == "" {
		log.Fatalf("Pull request does not have version label")
	}

	return label
}

func getPRNumberCircle(info string) int {
	regex := regexp.MustCompile("/pull/[0-9]+")
	match := regex.FindString(info)
	if match == "" {
		log.Fatalf("Could not determine pull request number from: %s", info)
	}
	num := strings.TrimPrefix(match, "/pull/")
	prNumber, err := strconv.Atoi(num)
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

func getLabel(labels []*github.Label) string {
	regex := regexp.MustCompile("version: (major|minor|patch)")

	for _, label := range labels {
		if label.Name != nil && regex.MatchString(*label.Name) {
			return *label.Name
		}
	}

	return ""
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
