package github_helpers

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/testhelpers/secrethelper"
	"github.com/google/go-github/v45/github"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var issueKeyRx = regexp.MustCompile(`(?i)(DX-\d+)`)

// ExtractJiraIssueID tries to extract the jira issue ID from either the PR title or the branch name
func ExtractJiraIssueID(pr *github.PullRequest) *string {
	if pr.Title == nil {
		panic(fmt.Sprintf("PR title is nil: %#v", pr))
	}
	if pr.Head == nil || pr.Head.Ref == nil {
		panic(fmt.Sprintf("Head or head ref is nil: %#v", pr))
	}

	// Extract from title
	matches := issueKeyRx.FindStringSubmatch(*pr.Title)
	if len(matches) == 2 {
		return p.StrP(strings.ToUpper(matches[1]))
	}

	// Extract from branch
	matches = issueKeyRx.FindStringSubmatch(*pr.Head.Ref)
	if len(matches) == 2 {
		return p.StrP(strings.ToUpper(matches[1]))
	}

	return nil
}

func InitClient() *github.Client {
	token := secrethelper.GetSecretIfEmpty(os.Getenv("GITHUB_TOKEN"), "user.GITHUB_TOKEN")

	// Init github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}
