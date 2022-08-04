package workflow_helpers

import (
	"os"
	"regexp"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/testhelpers/secrethelper"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
)

var jiraIssueRx = regexp.MustCompile(`(?i)(DX-\d+)`)

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
	return matches[1], nil
}

func JqlUnpaged(client *jira.Client, jql string) ([]jira.Issue, error) {
	result := []jira.Issue{}
	page := 0
	perPage := 100

	for x := 0; x < 100; x++ { // hard limit of 100,000 commits
		issues, _, err := client.Issue.Search(jql, &jira.SearchOptions{
			StartAt:    page * x,
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
	return semver.Parse(strings.TrimPrefix(version, "v"))
}

func FetchJiraIssue(jiraClient *jira.Client, jiraIssueID string) (*jira.Issue, error) {
	jiraIssue, _, err := jiraClient.Issue.Get(jiraIssueID, nil)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get Jira issue")
	}

	return jiraIssue, nil
}

var ErrVersionIsAny = errs.New("Version is '%s'", VersionAny)

func ParseTargetFixVersion(issue *jira.Issue, verifyActive bool) (semver.Version, *jira.FixVersion, error) {
	if len(issue.Fields.FixVersions) < 1 {
		return semver.Version{}, nil, errs.New("Jira issue does not have a fixVersion assigned: %s\n", issue.Key)
	}

	if len(issue.Fields.FixVersions) > 1 {
		return semver.Version{}, nil, errs.New("Jira issue has multiple fixVersions assigned: %s. This is incompatible with our workflow.", issue.Key)
	}

	fixVersion := issue.Fields.FixVersions[0]
	if verifyActive && (fixVersion.Archived != nil && *fixVersion.Archived) || (fixVersion.Released != nil && *fixVersion.Released) {
		return semver.Version{}, nil, errs.New("fixVersion '%s' has either been archived or released\n", fixVersion.Name)
	}

	if fixVersion.Name == VersionAny {
		return semver.Version{}, fixVersion, ErrVersionIsAny
	}

	v, err := ParseJiraVersion(fixVersion.Name)
	return v, fixVersion, err
}

func IsMergedStatus(status string) bool {
	if strings.HasPrefix(status, "Ready for") || status == "Done" || strings.Contains(status, "Testing") {
		return true
	}
	return false
}
