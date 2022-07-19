package workflow_helpers

import (
	"log"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
)

const (
	MasterBranch  = "master"
	StagingBranch = "beta"
	ReleaseBranch = "release"
)

const VersionedPRPrefix = "Version "

func VersionedPRTitle(version semver.Version) string {
	return VersionedPRPrefix + version.String()
}

func VersionFromPRTitle(title string) *semver.Version {
	if !strings.HasPrefix(title, VersionedPRPrefix) {
		return nil
	}
	v, err := semver.Parse(strings.TrimPrefix(title, VersionedPRPrefix))
	if err != nil {
		log.Printf("Error parsing version from PR title: %s\n", err)
		return nil
	}
	return &v
}

func VersionedBranchName(version semver.Version) string {
	return "version/" + strings.Replace(version.String(), ".", "-", -1)
}

func issueWithVersionLT(issues []*github.Issue, lessThanThisVersion semver.Version) *github.Issue {
	var issueForClosestVersion *github.Issue
	var closestVersion *semver.Version
	for _, issue := range issues {
		if !strings.HasPrefix(issue.GetTitle(), VersionedPRPrefix) {
			// Search doesn't ensure that it's a prefix
			continue
		}
		version := VersionFromPRTitle(issue.GetTitle())
		if version == nil || version.GT(lessThanThisVersion) {
			continue
		}
		if closestVersion == nil || closestVersion.LT(*version) {
			closestVersion = version
			issueForClosestVersion = issue
		}
	}

	return issueForClosestVersion
}
