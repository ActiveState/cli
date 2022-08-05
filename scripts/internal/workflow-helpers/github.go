package workflow_helpers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/testhelpers/secrethelper"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"github.com/thoas/go-funk"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

func InitGHClient() *github.Client {
	token := secrethelper.GetSecretIfEmpty(os.Getenv("GITHUB_TOKEN"), "user.GITHUB_TOKEN")

	// Init github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
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

	for x := 0; x < 10; x++ { // Hard limit of 1000 most recent PRs
		opts.ListOptions = github.ListOptions{
			Page:    x,
			PerPage: 100,
		}
		// Grab github PRs to compare against jira stories, cause Jira's API does not tell us what the linker PR is
		prs, _, err := ghClient.PullRequests.List(context.Background(), "ActiveState", "cli", opts)
		if err != nil {
			return nil, errs.Wrap(err, "Could not find PRs")
		}
		if len(prs) > 0 && prs[0].UpdatedAt.Before(cutoff) {
			break // The rest of the PRs are too old to care about
		}
		result = append(result, prs...)
		if len(prs) < opts.ListOptions.PerPage {
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
	page := 0
	perPage := 100

	for x := 0; x < 100; x++ { // hard limit of 100,000 commits
		commits, _, err := ghClient.Repositories.ListCommits(context.Background(), "ActiveState", "cli", &github.CommitsListOptions{
			SHA: ref,
			ListOptions: github.ListOptions{
				Page:    page * x,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, errs.Wrap(err, "ListCommits failed")
		}

		for _, commit := range commits {
			if stop != nil && stop(commit) {
				return result, nil
			}
			result = append(result, commit)
		}

		if len(commits) < perPage {
			break // Last page
		}

		page++
	}

	return result, nil
}

func SearchGithubIssues(client *github.Client, term string) ([]*github.Issue, error) {
	issues := []*github.Issue{}
	page := 0
	perPage := 100
	for x := 0; x < 10; x++ { // hard limit of 1,000 issues
		result, _, err := client.Search.Issues(context.Background(), "repo:ActiveState/cli  "+term, &github.SearchOptions{
			ListOptions: github.ListOptions{
				Page:    page * x,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, errs.Wrap(err, "Search.Issues failed")
		}
		issues = append(issues, result.Issues...)
		if result.GetTotal() == len(issues) || len(result.Issues) < perPage {
			break // Last page
		}
	}

	return issues, nil
}

func FetchPRByTitle(ghClient *github.Client, title string) (*github.PullRequest, error) {
	// Strip out words containing illegal characters. This'll mean we get more results than we want, but it's still
	// faster than iterating over ALL prs
	searchTerm := sanitizeSearchTerm(title)
	if searchTerm == "" {
		return nil, errs.New("All title words contain illegal characters, so no search could be performed")
	}

	var targetIssue *github.Issue
	issues, _, err := ghClient.Search.Issues(context.Background(), fmt.Sprintf("repo:ActiveState/cli in:title is:pr %s", searchTerm), nil)
	if err != nil {
		return nil, errs.Wrap(err, "failed to search for issues")
	}

	for _, issue := range issues.Issues {
		if issue.GetTitle() == title {
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
		Base:  p.StrP(baseBranch),
		Body:  p.StrP(body),
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
		if versionValue == VersionAny {
			continue
		}
		seen[versionValue] = struct{}{}
		version, err := ParseJiraVersion(versionValue)
		if err != nil {
			return nil, errs.Wrap(err, "failed to parse version: %s", versionValue)
		}
		result = append(result, version)
	}

	return result, nil
}

func UpdatePRTargetBranch(client *github.Client, prnumber int, targetBranch string) error {
	_, _, err := client.PullRequests.Edit(context.Background(), "ActiveState", "cli", prnumber, &github.PullRequest{
		Base: &github.PullRequestBranch{
			Ref: github.String(fmt.Sprintf("refs/heads/%s", targetBranch)),
		},
	})
	if err != nil {
		return errs.Wrap(err, "failed to update PR target branch")
	}
	return nil
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
			SHA: p.StrP(SHA),
		},
	})
	if err != nil {
		return errs.Wrap(err, "failed to create ref")
	}
	return nil
}

func CreateFileUpdateCommit(ghClient *github.Client, branchName string, path string, contents string) (string, error) {
	fileContents, _, _, err := ghClient.Repositories.GetContents(context.Background(), "ActiveState", "cli", path, &github.RepositoryContentGetOptions{
		Ref: branchName,
	})
	if err != nil {
		return "", errs.Wrap(err, "failed to get file contents for %s on branch %s", path, branchName)
	}

	resp, _, err := ghClient.Repositories.UpdateFile(context.Background(), "ActiveState", "cli", path, &github.RepositoryContentFileOptions{
		Author: &github.CommitAuthor{
			Name:  p.StrP("ActiveState CLI Automation"),
			Email: p.StrP("support@activestate.com"),
		},
		Branch:  &branchName,
		Message: p.StrP(fmt.Sprintf("Update %s", path)),
		Content: []byte(contents),
		SHA:     fileContents.SHA,
	})
	if err != nil {
		return "", errs.Wrap(err, "failed to update file")
	}

	return resp.GetSHA(), nil
}

// sannitizeSearchTerm strips words containing illegal characters from the search term
// https://docs.github.com/en/search-github/searching-on-github/searching-code#considerations-for-code-search
func sanitizeSearchTerm(term string) string {
	illegal := strings.Split(". , : ; / \\ ` ' \" = * ! ? # $ & + ^ | ~ < > ( ) { } [ ] @", " ")
	var result string
	skip := false
	lastSpace := 0
	for x := 0; x < len(term); x++ {
		char := string(term[x])
		if char == " " {
			lastSpace = len(result)
			skip = false
		} else if funk.Contains(illegal, char) {
			skip = true
			result = result[0:lastSpace]
		}
		if skip {
			continue
		}
		result += char
	}
	return strings.TrimSpace(result)
}
