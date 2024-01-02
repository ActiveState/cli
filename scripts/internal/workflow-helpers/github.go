package workflow_helpers

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/testhelpers/secrethelper"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"github.com/thoas/go-funk"
	"golang.org/x/net/context"
)

func InitGHClient() *github.Client {
	token := secrethelper.GetSecretIfEmpty(os.Getenv("GITHUB_TOKEN"), "user.GITHUB_TOKEN")

	return github.NewClient(&http.Client{
		Transport: NewRateLimitTransport(http.DefaultTransport, token),
	})
}

// ExtractJiraIssueID tries to extract the jira issue ID from the branch name
func ExtractJiraIssueID(pr *github.PullRequest) (string, error) {
	if pr.Head == nil || pr.Head.Ref == nil {
		panic(fmt.Sprintf("Head or head ref is nil: %#v", pr))
	}

	v, err := ParseJiraKey(*pr.Head.Ref)
	if err != nil {
		return "", errs.New("Please ensure your branch name is valid")
	}
	return v, nil
}

// FetchPRs fetches all PRs and iterates over all available pages
func FetchPRs(ghClient *github.Client, cutoff time.Time, opts *github.PullRequestListOptions) ([]*github.PullRequest, error) {
	result := []*github.PullRequest{}

	if opts == nil {
		opts = &github.PullRequestListOptions{
			State: "closed",
			Base:  "master",
		}
	}

	opts.Sort = "updated"
	opts.Direction = "desc"

	nextPage := 1

	for x := 0; x < 10; x++ { // Hard limit of 1000 most recent PRs
		opts.ListOptions = github.ListOptions{
			Page:    nextPage,
			PerPage: 100,
		}
		// Grab github PRs to compare against jira stories, cause Jira's API does not tell us what the linker PR is
		prs, v, err := ghClient.PullRequests.List(context.Background(), "ActiveState", "cli", opts)
		if err != nil {
			return nil, errs.Wrap(err, "Could not find PRs")
		}
		nextPage = v.NextPage
		if len(prs) > 0 && prs[0].UpdatedAt.Before(cutoff) {
			break // The rest of the PRs are too old to care about
		}
		result = append(result, prs...)
		if nextPage == 0 {
			break // Last page
		}
	}

	return result, nil
}

func FetchCommitsByShaRange(ghClient *github.Client, startSha string, stopSha string) ([]*github.RepositoryCommit, error) {
	return FetchCommitsByRef(ghClient, startSha, func(commit *github.RepositoryCommit) bool {
		return commit.GetSHA() == stopSha
	})
}

func FetchCommitsByRef(ghClient *github.Client, ref string, stop func(commit *github.RepositoryCommit) bool) ([]*github.RepositoryCommit, error) {
	result := []*github.RepositoryCommit{}
	perPage := 100
	nextPage := 1

	for x := 0; x < 100; x++ { // hard limit of 100,000 commits
		commits, v, err := ghClient.Repositories.ListCommits(context.Background(), "ActiveState", "cli", &github.CommitsListOptions{
			SHA: ref,
			ListOptions: github.ListOptions{
				Page:    nextPage,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, errs.Wrap(err, "ListCommits failed")
		}
		nextPage = v.NextPage

		for _, commit := range commits {
			if stop != nil && stop(commit) {
				return result, nil
			}
			result = append(result, commit)
		}

		if nextPage == 0 {
			break // Last page
		}

		if x == 99 {
			fmt.Println("WARNING: Hard limit of 100,000 commits reached")
		}
	}

	return result, nil
}

func SearchGithubIssues(client *github.Client, term string) ([]*github.Issue, error) {
	issues := []*github.Issue{}
	perPage := 100
	nextPage := 1

	for x := 0; x < 10; x++ { // hard limit of 1,000 issues
		result, v, err := client.Search.Issues(context.Background(), "repo:ActiveState/cli  "+term, &github.SearchOptions{
			ListOptions: github.ListOptions{
				Page:    nextPage,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, errs.Wrap(err, "Search.Issues failed")
		}
		nextPage = v.NextPage
		issues = append(issues, result.Issues...)
		if nextPage == 0 {
			break // Last page
		}
	}

	return issues, nil
}

func FetchPRByTitle(ghClient *github.Client, title string) (*github.PullRequest, error) {
	var targetIssue *github.Issue
	issues, _, err := ghClient.Search.Issues(context.Background(), fmt.Sprintf("repo:ActiveState/cli in:title is:pr %s", title), nil)
	if err != nil {
		return nil, errs.Wrap(err, "failed to search for issues")
	}

	for _, issue := range issues.Issues {
		if strings.TrimSpace(issue.GetTitle()) == strings.TrimSpace(title) {
			targetIssue = issue
			break
		}
	}

	if targetIssue != nil {
		targetPR, err := FetchPR(ghClient, *targetIssue.Number)
		if err != nil {
			return nil, errs.Wrap(err, "failed to get PR")
		}
		return targetPR, nil
	}

	return nil, nil
}

func FetchPR(ghClient *github.Client, number int) (*github.PullRequest, error) {
	pr, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", number)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get PR")
	}
	return pr, nil
}

func CreatePR(ghClient *github.Client, prName, branchName, baseBranch, body string) (*github.PullRequest, error) {
	payload := &github.NewPullRequest{
		Title: &prName,
		Head:  &branchName,
		Base:  ptr.To(baseBranch),
		Body:  ptr.To(body),
	}

	pr, _, err := ghClient.PullRequests.Create(context.Background(), "ActiveState", "cli", payload)
	if err != nil {
		return nil, errs.Wrap(err, "failed to create PR")
	}

	return pr, nil
}

func LabelPR(ghClient *github.Client, prnumber int, labels []string) error {
	if _, _, err := ghClient.Issues.AddLabelsToIssue(
		context.Background(), "ActiveState", "cli", prnumber, labels,
	); err != nil {
		return errs.Wrap(err, "failed to add label")
	}
	return nil
}

type Assertion string

const (
	AssertLT Assertion = "less than"
	AssertGT           = "greater than"
)

func FetchVersionPRs(ghClient *github.Client, assert Assertion, versionToCompare semver.Version, limit int) ([]*github.PullRequest, error) {
	issues, err := SearchGithubIssues(ghClient, "is:pr in:title "+VersionedPRPrefix)
	if err != nil {
		return nil, errs.Wrap(err, "failed to search for PRs")
	}

	filtered := issuesWithVersionAssert(issues, assert, versionToCompare)
	result := []*github.PullRequest{}
	for n, issue := range filtered {
		if !strings.HasPrefix(issue.GetTitle(), VersionedPRPrefix) {
			// The search above matches the whole title, and is very forgiving, which we don't want to be
			continue
		}
		pr, err := FetchPR(ghClient, issue.GetNumber())
		if err != nil {
			return nil, errs.Wrap(err, "failed to get PR")
		}
		result = append(result, pr)
		if limit != -1 && n+1 == limit {
			break
		}
	}

	return result, nil
}

func FetchVersionPR(ghClient *github.Client, assert Assertion, versionToCompare semver.Version) (*github.PullRequest, error) {
	prs, err := FetchVersionPRs(ghClient, assert, versionToCompare, 1)
	if err != nil {
		return nil, err
	}
	if len(prs) == 0 {
		return nil, errs.New("Could not find issue with version %s %s", assert, versionToCompare.String())
	}
	return prs[0], nil
}

func BranchHasVersionsGT(client *github.Client, jiraClient *jira.Client, branchName string, version semver.Version) (bool, error) {
	versions, err := ActiveVersionsOnBranch(client, jiraClient, branchName, time.Now().AddDate(0, -6, 0))
	if err != nil {
		return false, errs.Wrap(err, "failed to get versions on master")
	}

	for _, v := range versions {
		if v.GT(version) {
			// Master has commits on it intended for versions greater than the one being targeted
			return true, nil
		}
	}

	return false, nil
}

func ActiveVersionsOnBranch(ghClient *github.Client, jiraClient *jira.Client, branchName string, dateCutoff time.Time) ([]semver.Version, error) {
	commits, err := FetchCommitsByRef(ghClient, branchName, func(commit *github.RepositoryCommit) bool {
		return commit.Commit.Committer.Date.Before(dateCutoff)
	})
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch commits")
	}
	jiraIDs := []string{}
	for _, commit := range commits {
		jiraID, err := ParseJiraKey(commit.Commit.GetMessage())
		if err != nil {
			// no match
			continue
		}
		jiraIDs = append(jiraIDs, jiraID)
	}

	jiraIDs = funk.Uniq(jiraIDs).([]string)
	issues, err := JqlUnpaged(jiraClient, fmt.Sprintf(`project = "DX" AND id IN(%s)`, strings.Join(jiraIDs, ",")))
	if err != nil {
		return nil, errs.Wrap(err, "failed to fetch issues")
	}

	seen := map[string]struct{}{}
	result := []semver.Version{}
	for _, issue := range issues {
		if issue.Fields.FixVersions == nil || len(issue.Fields.FixVersions) == 0 {
			continue
		}
		versionValue := issue.Fields.FixVersions[0].Name
		if _, ok := seen[versionValue]; ok {
			continue
		}
		seen[versionValue] = struct{}{}
		version, err := ParseJiraVersion(versionValue)
		if err != nil {
			logging.Debug("Failed to parse version %s: %v", versionValue, err)
			continue
		}
		result = append(result, version)
	}

	return result, nil
}

func UpdatePRTargetBranch(client *github.Client, prnumber int, targetBranch string) error {
	_, _, err := client.PullRequests.Edit(context.Background(), "ActiveState", "cli", prnumber, &github.PullRequest{
		Base: &github.PullRequestBranch{
			Ref: github.String(targetBranch),
		},
	})
	if err != nil {
		return errs.Wrap(err, "failed to update PR target branch")
	}
	return nil
}

func GetCommitsBehind(client *github.Client, base, head string) ([]*github.RepositoryCommit, error) {
	// Note we're swapping base and head when doing this because github responds with the commits that are ahead, rather than behind.
	commits, _, err := client.Repositories.CompareCommits(context.Background(), "ActiveState", "cli", head, base, nil)
	if err != nil {
		return nil, errs.Wrap(err, "failed to compare commits")
	}
	result := []*github.RepositoryCommit{}
	for _, commit := range commits.Commits {
		msg := strings.Split(commit.GetCommit().GetMessage(), "\n")[0] // first line only
		msgWords := strings.Split(msg, " ")
		if msg == UpdateVersionCommitMessage {
			// Updates to version.txt are not meant to be inherited
			continue
		}
		suffix := strings.TrimPrefix(msgWords[len(msgWords)-1], "ActiveState/")
		if (strings.HasPrefix(msg, "Merge pull request") && IsVersionBranch(suffix)) ||
			(strings.HasPrefix(msg, "Merge branch '"+constants.BetaChannel+"'") && IsVersionBranch(suffix)) {
			// Git's compare commits is not smart enough to consider merge commits from other version branches equal
			// This matches the following types of messages:
			// Merge pull request #2531 from ActiveState/version/0-38-1-RC1
			// Merge branch 'beta' into version/0-40-0-RC1
			continue
		}
		result = append(result, commit)
	}
	return result, nil
}

func SetPRBody(client *github.Client, prnumber int, body string) error {
	_, _, err := client.PullRequests.Edit(context.Background(), "ActiveState", "cli", prnumber, &github.PullRequest{
		Body: &body,
	})
	if err != nil {
		return errs.Wrap(err, "failed to set PR body")
	}
	return nil
}

func CreateBranch(ghClient *github.Client, branchName string, SHA string) error {
	_, _, err := ghClient.Git.CreateRef(context.Background(), "ActiveState", "cli", &github.Reference{
		Ref: github.String(fmt.Sprintf("refs/heads/%s", branchName)),
		Object: &github.GitObject{
			SHA: ptr.To(SHA),
		},
	})
	if err != nil {
		return errs.Wrap(err, "failed to create ref")
	}
	return nil
}

func CreateFileUpdateCommit(ghClient *github.Client, branchName string, path string, contents string, message string) (string, error) {
	fileContents, _, _, err := ghClient.Repositories.GetContents(context.Background(), "ActiveState", "cli", path, &github.RepositoryContentGetOptions{
		Ref: branchName,
	})
	if err != nil {
		return "", errs.Wrap(err, "failed to get file contents for %s on branch %s", path, branchName)
	}

	resp, _, err := ghClient.Repositories.UpdateFile(context.Background(), "ActiveState", "cli", path, &github.RepositoryContentFileOptions{
		Author: &github.CommitAuthor{
			Name:  ptr.To("ActiveState CLI Automation"),
			Email: ptr.To("support@activestate.com"),
		},
		Branch:  &branchName,
		Message: ptr.To(message),
		Content: []byte(contents),
		SHA:     fileContents.SHA,
	})
	if err != nil {
		return "", errs.Wrap(err, "failed to update file")
	}

	return resp.GetSHA(), nil
}
