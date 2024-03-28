package workflow_helpers

import (
	"io"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/testhelpers/secrethelper"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
)

var jiraIssueRx = regexp.MustCompile(`(?i)(DX-\d+)`)

const JiraStatusTodo = "To Do"
const JiraStatusInProgress = "In Progress"
const JiraStatusPending = "Pending"

func InitJiraClient() (*jira.Client, error) {
	username := secrethelper.GetSecretIfEmpty(os.Getenv("JIRA_USERNAME"), "user.JIRA_USERNAME")
	password := secrethelper.GetSecretIfEmpty(os.Getenv("JIRA_TOKEN"), "user.JIRA_TOKEN")

	tp := &jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}
	jiraClient, err := jira.NewClient(tp.Client(), "https://activestatef.atlassian.net/")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to create JIRA client")
	}
	return jiraClient, nil
}

func ParseJiraKey(v string) (string, error) {
	matches := jiraIssueRx.FindStringSubmatch(v)
	if len(matches) < 1 {
		return "", errs.New("Could not extract jira key from %s, please ensure it matches the regex: %s", v, jiraIssueRx.String())
	}
	return strings.ToUpper(matches[1]), nil
}

func JqlUnpaged(client *jira.Client, jql string) ([]jira.Issue, error) {
	result := []jira.Issue{}
	perPage := 100

	for x := 0; x < 100; x++ { // hard limit of 100,000 commits
		issues, _, err := client.Issue.Search(jql, &jira.SearchOptions{
			StartAt:    x * perPage,
			MaxResults: perPage,
		})
		if err != nil {
			return nil, errs.Wrap(err, "Failed to search JIRA")
		}
		result = append(result, issues...)
		if len(issues) < perPage {
			break
		}
	}

	return result, nil
}

func ParseJiraVersion(version string) (semver.Version, error) {
	return semver.Parse(ParseJiraVersionRaw(version))
}

func ParseJiraVersionRaw(version string) string {
	return strings.TrimPrefix(version, "v")
}

func FetchJiraIssue(jiraClient *jira.Client, jiraIssueID string) (*jira.Issue, error) {
	jiraIssue, _, err := jiraClient.Issue.Get(jiraIssueID, nil)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get Jira issue")
	}

	return jiraIssue, nil
}

type Version struct {
	semver.Version
	JiraID string
}

func FetchAvailableVersions(jiraClient *jira.Client) ([]Version, error) {
	pj, _, err := jiraClient.Project.Get("DX")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get JIRA project")
	}

	emptySemver := semver.Version{}
	result := []Version{}
	for _, version := range pj.Versions {
		if version.Archived != nil && *version.Archived {
			continue
		}
		if version.Released != nil && *version.Released {
			continue
		}
		semversion, err := ParseJiraVersion(version.Name)
		if err != nil || semversion.EQ(emptySemver) {
			logging.Debug("Not a semver version %s; skipping", version.Name)
			continue
		}
		result = append(result, Version{semversion, version.ID})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].LT(result[j].Version)
	})

	return result, nil
}

var VersionMaster = semver.MustParse("0.0.0")

func ParseTargetFixVersion(issue *jira.Issue, availableVersions []Version) (target Version, original *jira.FixVersion, err error) {
	if len(issue.Fields.FixVersions) < 1 {
		return Version{}, nil, errs.New("Jira issue does not have a fixVersion assigned: %s\n", issue.Key)
	}

	if len(issue.Fields.FixVersions) > 1 {
		return Version{}, nil, errs.New("Jira issue has multiple fixVersions assigned: %s. This is incompatible with our workflow.", issue.Key)
	}

	fixVersion := issue.Fields.FixVersions[0]
	if fixVersion.Archived != nil && *fixVersion.Archived || fixVersion.Released != nil && *fixVersion.Released {
		return Version{}, nil, errs.New("fixVersion '%s' has either been archived or released\n", fixVersion.Name)
	}

	switch fixVersion.Name {
	case VersionNextFeasible:
		target, err := ParseJiraVersion(strings.Split(fixVersion.Description, " ")[0])
		if err != nil {
			return Version{}, nil, errs.Wrap(err, "Failed to parse Jira version from description: %s", fixVersion.Description)
		}
		if len(availableVersions) < 1 {
			return Version{}, nil, errs.New("No feasible versions available")
		}
		for _, version := range availableVersions {
			if version.EQ(target) {
				return version, fixVersion, nil
			}
		}
		return Version{}, nil, errs.New("Next feasible version does not exist: %s", target.String())
	case VersionNextUnscheduled:
		return Version{VersionMaster, ""}, fixVersion, nil
	}

	v, err := ParseJiraVersion(fixVersion.Name)
	return Version{v, fixVersion.ID}, fixVersion, err
}

func IsMergedStatus(status string) bool {
	if strings.HasPrefix(status, "Ready for") || status == "Done" || strings.Contains(status, "Testing") {
		return true
	}
	return false
}

func FetchJiraIDsInCommits(commits []*github.RepositoryCommit) []string {
	found := []string{}
	for _, commit := range commits {
		key, err := ParseJiraKey(commit.GetCommit().GetMessage())
		if err != nil {
			continue
		}
		found = append(found, strings.ToUpper(key))
	}
	return found
}

func UpdateJiraFixVersion(client *jira.Client, issue *jira.Issue, versionID string) error {
	issueUpdate := &jira.Issue{
		ID:  issue.ID,
		Key: issue.Key,
		Fields: &jira.IssueFields{
			FixVersions: []*jira.FixVersion{{ID: versionID}},
		},
	}

	if len(issue.Fields.FixVersions) > 0 {
		fixVersion := issue.Fields.FixVersions[0]
		switch fixVersion.Name {
		case VersionNextFeasible:
			issueUpdate.Fields.Labels = append(issueUpdate.Fields.Labels, "WasNextFeasible")
		case VersionNextUnscheduled:
			issueUpdate.Fields.Labels = append(issueUpdate.Fields.Labels, "WasNextUnscheduled")
		}
	}

	issueUpdate.Fields.FixVersions = []*jira.FixVersion{{ID: versionID}}
	_, response, err := client.Issue.Update(issueUpdate)
	res, err2 := io.ReadAll(response.Body)
	if err2 != nil {
		res = []byte(err2.Error())
	}
	if err != nil {
		return errs.Wrap(err, string(res))
	}
	return nil
}

func UpdateJiraStatus(client *jira.Client, issue *jira.Issue, statusName string) error {
	transitions, _, err := client.Issue.GetTransitions(issue.ID)
	if err != nil {
		return errs.Wrap(err, "failed to get Jira transitions")
	}

	var transition *jira.Transition
	for _, t := range transitions {
		if t.To.Name == statusName {
			transition = &t
			break
		}
	}
	if transition == nil {
		return errs.New("failed to find a Jira transition that changes the status to %s for issue %s", statusName, issue.Key)
	}

	response, err := client.Issue.DoTransition(issue.ID, transition.ID)
	if err != nil && response == nil {
		return errs.Wrap(err, "failed to perform Jira transition")
	}

	// Include response body in error
	res, err2 := io.ReadAll(response.Body)
	if err2 != nil {
		res = []byte(err2.Error())
	}
	if err != nil {
		return errs.Wrap(err, string(res))
	}
	return nil
}
