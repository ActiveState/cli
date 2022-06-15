package main

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/stacktrace"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/scripts/internal/github-helpers"
	"github.com/ActiveState/cli/scripts/internal/jira-helpers"
	"github.com/andygrunwald/go-jira"
	"github.com/codemodus/relay"
	"github.com/google/go-github/v45/github"
	"github.com/thoas/go-funk"
	"golang.org/x/net/context"
	"gopkg.in/AlecAivazis/survey.v1"
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
			r.Check(errs.New("Usage: update-rc <sha-of-merged-pr>"))
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
	r = relay.New(func(err error) {
		curHead, err2 := repo.Head()
		r.Check(err2)

		fmt.Printf("Failed with error: %s (%v)\n%s\n", errs.JoinMessage(err), err, stacktrace.Get().String())
		if curHead.Hash().String() != repoHead.Hash().String() {
			fmt.Println("Reverting checkout")
			execute("git", "checkout", repoHead.Name().Short())
		}

		os.Exit(1)
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

		execute("git", "checkout", "beta")
		execute("git", "branch", branchName)
		execute("git", "checkout", branchName)
		execute("git", "push", "--set-upstream", "origin", branchName)
	}

	remoteBranchName := "origin/" + branchName

	// Check Out Rc Branch So We Can Cherry Pick
	execute("git", "checkout", branchName)

	// Cherry Pick the merge commit to the RC branch
	if !cherryPick(shaOfMergedPR) {
		execute("git", "checkout", repoHead.Name().Short())
		return
	}

	// Push changes to RC branch
	fmt.Printf("Pushing %s to %s\n", branchName, remoteBranchName)
	execute("git", "push")

	// Check Out Original Commit
	execute("git", "checkout", repoHead.Name().Short())

	// Create Relevant Fixversion Pr If None Exists
	if targetPR == nil {
		targetPR = createTargetPR(ghClient, fixVersion, prName, branchName)
	}

	updateTargetPR(ghClient, targetPR, mergedPR, jiraIssue)

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

	if len(issue.Fields.FixVersions) > 1 {
		r.Check(errs.New("Jira issue has multiple fixVersions assigned: %s. This is incompatible with our workflow.", issue.Key))
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

		fmt.Printf("Found PR %d for fixVersion '%s'\n", *targetIssue.Number, version)
		return targetPR
	}

	return nil
}

func createTargetPR(ghClient *github.Client, fixVersion *jira.FixVersion, prName string, branchName string) *github.PullRequest {
	u, err := url.Parse("https://activestatef.atlassian.net/jira/software/c/projects/DX/issues/")
	r.Check(err)

	q := u.Query()
	q.Set("jql", fmt.Sprintf(`project = "DX" AND fixVersion=%s ORDER BY created DESC`, fixVersion.Name))
	u.RawQuery = q.Encode()

	payload := &github.NewPullRequest{
		Title: &prName,
		Head:  &branchName,
		Base:  p.StrP("beta"),
		Body:  p.StrP(fmt.Sprintf(`[View %s tickets on Jira](%s)`, fixVersion.Name, u.String())),
	}

	fmt.Printf("Creating PR for fixVersion: %s, with name: %s\n", fixVersion.Name, prName)

	targetPR, _, err := ghClient.PullRequests.Create(context.Background(), "ActiveState", "cli", payload)
	r.Check(err)
	_, _, err2 := ghClient.Issues.AddLabelsToIssue(context.Background(), "ActiveState", "cli", *targetPR.Number, []string{"Test: all"})
	r.Check(err2)

	return targetPR
}

func updateTargetPR(client *github.Client, targetPR *github.PullRequest, sourcePR *github.PullRequest, issue *jira.Issue) {
	body := fmt.Sprintf("%s\n* %s [%s](https://activestatef.atlassian.net/browse/%s) [#%d](%s)",
		*targetPR.Body, issue.Fields.Summary, issue.Key, issue.Key, *sourcePR.Number, sourcePR.Links.GetHTML().GetHRef())
	_, _, err := client.PullRequests.Edit(context.Background(), "ActiveState", "cli", *targetPR.Number, &github.PullRequest{Body: &body})
	r.Check(err)
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

func diffFiles() string {
	stdout, _, err := exeutils.ExecSimple("git", []string{"diff-files", "-p"}, []string{})
	r.Check(err)
	return strings.TrimSpace(stdout)
}

func cherryPick(shaOfMergedPR string) bool {
	err := executeWithErr("git", "cherry-pick", "-m", "1", shaOfMergedPR)
	if err == nil {
		return true
	}

	if condition.OnCI() {
		r.Check(errs.New(`
Cherry picking failed. Please run this script locally and address the failure:
state run update-version-branch %s
`, shaOfMergedPR))
	}

	var resp bool
	err2 := survey.AskOne(&survey.Confirm{
		Message: "Cherry picking failed. Please manually address failure and then continue. Ready?",
		Default: true,
	}, &resp, nil)
	r.Check(err2)

	if !resp {
		r.Check(executeWithErr("git", "cherry-pick", "--abort"))
	}

	return resp
}

func executeWithErr(args ...string) error {
	fmt.Printf("Executing: %#v\n", args)
	c := exec.Command(args[0], args[1:]...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	if err != nil {
		return errs.Wrap(err, fmt.Sprintf("stdout: %s\nstderr: %s", stdout.String(), stderr.String()))
	}

	code := osutils.CmdExitCode(c)
	if code != 0 {
		return errs.New("%#v returned code %d", args, code)
	}

	return nil
}

func execute(args ...string) {
	r.Check(executeWithErr(args...))
}
