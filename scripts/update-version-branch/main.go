package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/scripts/internal/github-helpers"
	jira_helpers "github.com/ActiveState/cli/scripts/internal/jira-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/codemodus/relay"
	"github.com/google/go-github/v45/github"
	"github.com/thoas/go-funk"
	"golang.org/x/net/context"
	"gopkg.in/src-d/go-git.v4"
)

var commitMessagePrefix = "Merge pull request #"
var commitMessageRx = regexp.MustCompile(commitMessagePrefix + `(\d+)`)

var r = relay.New()

func main() {
	defer relay.Handle()

	// Validate Input
	{
		// Verify input args
		if len(os.Args) != 2 {
			r.Check(errs.New("Usage: update-version-branch <sha-of-desired-pr>"))
			return
		}

		// Only run on push to master, unless we're not on CI
		if condition.OnCI() && (os.Getenv("GITHUB_EVENT") != "push" || os.Getenv("GITHUB_REF_NAME") != "master") {
			fmt.Println("Not a push event targeting master")
			return
		}

		// Ensure our worktree is clean so we don't accidentally clobber anything
		if v := diffFiles(); v != "" {
			r.Check(errs.New("Working tree is not clean:\n%s", v))
			return
		}
	}

	path := environment.GetRootPathUnsafe()
	shaOfMergedPR := os.Args[1]

	// Initialize Clients
	ghClient := github_helpers.InitClient()
	jiraClient := jira_helpers.InitClient()

	repo, err := git.PlainOpen(path)
	r.Check(err)

	// Ensure that whatever we do, we end up back where we started
	repoHead, err := repo.Head()
	r.Check(err)
	r = relay.New(func(error) {
		curHead, err := repo.Head()
		r.Check(err)

		if curHead.Hash().String() != repoHead.Hash().String() {
			checkout(repoHead.Name().String())
		}
		relay.DefaultHandler()(err)
	})

	// Retrieve Relevant Pr
	mergedPR := getMergedPR(ghClient, shaOfMergedPR)
	if mergedPR == nil {
		return
	}

	// Retrieve Relevant Jira Issue
	jiraIssue := getJiraIssueFromPR(jiraClient, mergedPR)
	if jiraIssue == nil {
		return
	}

	// Retrieve Relevant Fixversion
	fixVersion := getTargetFixVersion(jiraIssue, true)
	if fixVersion == nil {
		return
	}

	// Retrieve Relevant Fixversion Pr
	var prName string
	var branchName string
	targetPR := getTargetPR(ghClient, fixVersion.Name)
	if targetPR != nil {
		// Check If Target Pr Already Contains Our Commit
		if !targetPRMissingMergedPR(ghClient, targetPR, mergedPR.GetNumber()) {
			return
		}

		prName = targetPR.GetTitle()
		branchName = targetPR.GetHead().GetRef()
	} else {
		// Set PR and branch names for PR that we'll be creating
		prName = fixVersion.Name
		branchName = strings.Replace(fixVersion.Name, ".", "_", -1)

		createBranch(branchName)
	}

	// Ensure
	defer checkout(repoHead.Hash().String())

	// Check Out Rc Branch So We Can Cherry Pick
	checkout(branchName)

	// Cherry Pick the merge commit to the RC branch
	cherryPick(shaOfMergedPR)

	// Push changes to RC branch
	r.Check(repo.Push(nil))

	// Check Out Original Commit
	checkout(shaOfMergedPR)

	// Create Relevant Fixversion Pr If None Exists
	if targetPR == nil {
		targetPR = createTargetPR(ghClient, fixVersion.Name, prName, branchName)
	}

	fmt.Println("Done")
}

func getMergedPR(gh *github.Client, sha string) *github.PullRequest {
	commit, _, err := gh.Git.GetCommit(context.Background(), "ActiveState", "cli", sha)
	r.Check(err)

	match := commitMessageRx.FindStringSubmatch(*commit.Message)
	if len(match) != 2 {
		fmt.Printf("Commit message '%s' does not match regexp '%s' -- skipping\n", *commit.Message, commitMessageRx)
		return nil
	}
	mergedPRID, err := strconv.Atoi(match[1])
	r.Check(err)

	mergedPR, _, err := gh.PullRequests.Get(context.Background(), "ActiveState", "cli", mergedPRID)
	r.Check(err)

	fmt.Printf("Extracted PR %d from provided sha\n", mergedPRID)

	return mergedPR
}

func getJiraIssueFromPR(jiraClient *jira.Client, pr *github.PullRequest) *jira.Issue {
	jiraIssueID := github_helpers.ExtractJiraIssueID(pr)
	if jiraIssueID == nil {
		fmt.Printf("PR does not have Jira issue ID associated with it: %s\n", pr.Links.GetHTML().GetHRef())
		return nil
	}

	jiraIssue, _, err := jiraClient.Issue.Get(*jiraIssueID, nil)
	r.Check(err)

	fmt.Printf("Extracted Jira issue %s from PR %d\n", jiraIssue.Key, pr.ID)

	return jiraIssue
}

func getTargetFixVersion(issue *jira.Issue, verifyActive bool) *jira.FixVersion {
	if len(issue.Fields.FixVersions) < 1 {
		fmt.Printf("Jira issue does not have a fixVersion assigned: %s\n", issue.Key)
		return nil
	}

	fixVersion := issue.Fields.FixVersions[0]
	if verifyActive && (fixVersion.Archived != nil && *fixVersion.Archived) || (fixVersion.Released != nil && *fixVersion.Released) {
		fmt.Printf("Skipping because fixVersion '%s' has either been archived or released\n", fixVersion.Name)
		return nil
	}

	fmt.Printf("Extracted fixVersion %s from Jira issue %s\n", fixVersion.Name, issue.Key)

	return fixVersion
}

func getTargetPR(ghClient *github.Client, version string) *github.PullRequest {
	var targetIssue *github.Issue
	searchTerm := strings.Split(version, "-")[0] // GitHub search doesn't support Dashes. I'm not joking.. This is real..
	issues, _, err := ghClient.Search.Issues(context.Background(), fmt.Sprintf("repo:ActiveState/cli is:pr %s", searchTerm), nil)
	r.Check(err)

	for _, issue := range issues.Issues {
		if issue.Title == nil || !strings.HasPrefix(*issue.Title, version) ||
			issue.State != nil && *issue.State == "closed" {
			continue
		}
		if targetIssue != nil {
			r.Check(errs.New("Multiple open PRs found for fixVersion '%s'", version))
			return nil
		}
		targetIssue = issue
	}

	if targetIssue != nil {
		targetPR, _, err := ghClient.PullRequests.Get(context.Background(), "ActiveState", "cli", *targetIssue.Number)
		r.Check(err)

		fmt.Printf("Found PR %s for fixVersion '%s'\n", *targetIssue.Number, version)
		return targetPR
	}

	return nil
}

func createBranch(name string) {
	fmt.Printf("Creating branch '%s' from beta\n", name)

	checkout("beta")

	// Technically the go-git lib is supposed to support this, but it's so low level it's not immediately evident how to work with it
	code, _, err := exeutils.ExecuteAndPipeStd("git", []string{"branch", name}, []string{})
	r.Check(err)
	if code != 0 {
		r.Check(errs.New("git checkout returned code %d", code))
	}
}

func createTargetPR(ghClient *github.Client, fixVersion string, prName string, branchName string) *github.PullRequest {
	payload := &github.NewPullRequest{
		Title: &prName,
		Head:  &branchName,
		Base:  p.StrP("beta"),
	}

	fmt.Printf("Creating PR for fixVersion: %s, with name: %s\n", fixVersion, prName)

	targetPR, _, err := ghClient.PullRequests.Create(context.Background(), "ActiveState", "cli", payload)
	r.Check(err)
	_, _, err2 := ghClient.Issues.AddLabelsToIssue(context.Background(), "ActiveState", "cli", *targetPR.Number, []string{"Test: all"})
	r.Check(err2)

	return targetPR
}

func targetPRMissingMergedPR(ghClient *github.Client, targetPR *github.PullRequest, seekPR int) bool {
	commits, _, err := ghClient.PullRequests.ListCommits(context.Background(), "ActiveState", "cli", *targetPR.Number, nil)
	commits = funk.Reverse(commits).([]*github.RepositoryCommit)
	r.Check(err)
	seek := commitMessagePrefix + strconv.Itoa(seekPR)
	found := false
	for _, commit := range commits {
		if !strings.HasPrefix(*commit.Commit.Message, seek) {
			continue
		}

		found = true
		fmt.Println("Release candidate PR already contains merge commit for merged PR")
	}

	return !found
}

func checkout(target string) {
	fmt.Printf("Checking out %s\n", target)
	// Technically the go-git lib is supposed to support this, but it's non-evident where this functionality is hidden and not worth my time
	code, _, err := exeutils.ExecuteAndPipeStd("git", []string{"checkout", target}, []string{})
	r.Check(err)
	if code != 0 {
		r.Check(errs.New("git checkout returned code %d", code))
	}
}

func cherryPick(sha string) {
	fmt.Println("Cherry Picking merged PR to RC branch")
	code, _, err := exeutils.ExecuteAndPipeStd("git", []string{"cherry-pick", "-m", "1", sha}, []string{})
	r.Check(err)
	if code != 0 {
		r.Check(errs.New("git cherry-pick returned code %d", code))
	}
}

func diffFiles() string {
	stdout, _, err := exeutils.ExecSimple("git", []string{"diff-files", "-p"}, []string{})
	r.Check(err)
	return strings.TrimSpace(stdout)
}
