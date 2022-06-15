package main

import (
	"testing"

	"github.com/ActiveState/cli/scripts/internal/github-helpers"
	"github.com/ActiveState/cli/scripts/internal/jira-helpers"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestGetMergePR() {
	var commitWithMerge = "c4d8519113ac3eb0ac565179787619bae96b7498"
	var commitWithoutMerge = "d18992f370ada0fd179fb4cdced811e77c7b1136"

	ghClient := github_helpers.InitClient()

	suite.Require().NotNil(getMergedPR(ghClient, commitWithMerge))
	suite.Require().Nil(getMergedPR(ghClient, commitWithoutMerge))
}

func (suite *MainTestSuite) TestGetJiraIssueFromPR() {
	var prWithJiraIssue = 1872
	var prWithoutJiraIssue = 1717

	jiraClient := jira_helpers.InitClient()
	ghClient := github_helpers.InitClient()

	{
		pr, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prWithJiraIssue)
		suite.Require().NoError(err)
		suite.Require().NotNil(getJiraIssueFromPR(jiraClient, pr))
	}

	{
		pr, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prWithoutJiraIssue)
		suite.Require().NoError(err)
		suite.Require().Nil(getJiraIssueFromPR(jiraClient, pr))
	}
}

func (suite *MainTestSuite) TestGetTargetFixVersion() {
	var jiraIssueWithFixVersion = "DX-965"
	var jiraIssueWithArchivedFixVersion = "DX-497"
	var jiraIssueWithoutFixVersion = "DX-968"

	jiraClient := jira_helpers.InitClient()

	{
		issue, _, err := jiraClient.Issue.Get(jiraIssueWithFixVersion, nil)
		suite.Require().NoError(err)
		suite.Require().NotNil(getTargetFixVersion(issue, false))
	}

	{
		issue, _, err := jiraClient.Issue.Get(jiraIssueWithArchivedFixVersion, nil)
		suite.Require().NoError(err)
		suite.Require().Nil(getTargetFixVersion(issue, true))
	}

	{
		issue, _, err := jiraClient.Issue.Get(jiraIssueWithoutFixVersion, nil)
		suite.Require().NoError(err)
		suite.Require().Nil(getTargetFixVersion(issue, true))
	}
}

func (suite *MainTestSuite) TestTargetPR() {
	var prefixForOpenVersionPR = "1.2.3-RC"
	var prefixForClosedVersionPR = "v0.32.0-RC"

	ghClient := github_helpers.InitClient()

	{
		pr := getTargetPR(ghClient, prefixForOpenVersionPR)
		suite.Require().NotNil(pr)
	}

	{
		pr := getTargetPR(ghClient, prefixForClosedVersionPR)
		suite.Require().Nil(pr)
	}
}

func (suite *MainTestSuite) TestTargetPRMissingMergedPR() {
	var prWithMergeAssertion = 1868
	var prWithFailedMergeAssertion = 1867
	var prMergeAssertion = 1824

	ghClient := github_helpers.InitClient()

	{
		pr, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prWithMergeAssertion)
		suite.Require().NoError(err)
		suite.Require().False(targetPRMissingMergedPR(ghClient, pr, prMergeAssertion)) // Already has merge
	}

	{
		pr, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", prWithFailedMergeAssertion)
		suite.Require().NoError(err)
		suite.Require().True(targetPRMissingMergedPR(ghClient, pr, prMergeAssertion)) // Doesn't have merge
	}
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
