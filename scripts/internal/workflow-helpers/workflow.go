package workflow_helpers

import (
	"log"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
)

const (
	MasterBranch  = "master"
	StagingBranch = "beta"
	ReleaseBranch = "release"
)

const VersionAny = "Any"

const VersionedPRPrefix = "Version "

func VersionedPRTitle(version semver.Version) string {
	return VersionedPRPrefix + version.String()
}

func VersionFromPRTitle(title string) (semver.Version, error) {
	if !strings.HasPrefix(title, VersionedPRPrefix) {
		return semver.Version{}, errs.New("Title: '%s' does not start with '%s'", title, VersionedPRPrefix)
	}
	v, err := semver.Parse(strings.TrimPrefix(title, VersionedPRPrefix))
	if err != nil {
		return semver.Version{}, errs.Wrap(err, "Failed to parse version from PR title")
	}
	return v, nil
}

const versionBranchPrefix = "version/"

func ValidVersionBranch(branchName string) error {
	if strings.HasPrefix(branchName, versionBranchPrefix) {
		return nil
	}
	return errs.New("Branch name: '%s' does not start with '%s'", branchName, versionBranchPrefix)
}

func VersionedBranchName(version semver.Version) string {
	return versionBranchPrefix + strings.Replace(version.String(), ".", "-", -1)
}

func issuesWithVersionAssert(issues []*github.Issue, assert Assertion, versionToCompare semver.Version) []*github.Issue {
	result := []*github.Issue{}
	for _, issue := range issues {
		if !strings.HasPrefix(issue.GetTitle(), VersionedPRPrefix) {
			// Search doesn't ensure that it's a prefix
			continue
		}
		version, err := VersionFromPRTitle(issue.GetTitle())
		if err != nil {
			// Not a version PR
			continue
		}
		switch assert {
		case AssertLT:
			if versionToCompare.LT(version) || versionToCompare.EQ(version) {
				continue
			}
			result = append(result, issue)
		case AssertGT:
			if versionToCompare.GT(version) || versionToCompare.EQ(version) {
				continue
			}
			result = append(result, issue)
		default:
			log.Printf("Unknown assertion: %v\n", assert)
			continue
		}
	}

	sort.Slice(result, func(i, j int) bool {
		v1, _ := VersionFromPRTitle(result[i].GetTitle())
		v2, _ := VersionFromPRTitle(result[j].GetTitle())
		switch assert {
		case AssertLT:
			return v2.LT(v1)
		case AssertGT:
			return v2.GT(v1)
		}

		return false
	})

	return result
}
