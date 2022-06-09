package main

import (
	"strings"
	"testing"

	github_helpers "github.com/ActiveState/cli/scripts/internal/github-helpers"
	"github.com/codemodus/relay"
	"github.com/stretchr/testify/suite"
	"golang.org/x/net/context"
)

type MainTestSuite struct {
	suite.Suite
}

func (suite *MainTestSuite) TestVerifyRC() {
	var rcPRWithMissingCommits = 1885

	ghClient := github_helpers.InitClient()
	pr, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", rcPRWithMissingCommits)
	suite.Require().NoError(err)

	var relayed = false
	r = relay.New(func(err error) {
		relayed = true
		if err == nil {
			suite.Fail("Expected error but got nil")
		}
		if !strings.Contains(err.Error(), "Missing JIRA issues") {
			suite.Fail("Should have received Missing jira issue error, but got: %s", err.Error())
		}
	})
	defer relay.Handle()

	verifyRC(ghClient, pr)
	suite.Require().True(relayed, "Should have triggered relay")
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
