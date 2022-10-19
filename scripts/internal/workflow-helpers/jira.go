package workflow_helpers

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
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

func FetchAvailableVersions(jiraClient *jira.Client) ([]semver.Version, error) {
	pj, _, err := jiraClient.Project.Get("DX")
	if err != nil {
		return nil, errs.Wrap(err, "Failed to get JIRA project")
	}

	result := []semver.Version{}
	for _, version := range pj.Versions {
		if version.Archived != nil || *version.Archived {
			continue
		}
		if version.Released != nil || *version.Released {
			continue
		}
		semversion, err := ParseJiraVersion(version.Name)
		if err != nil {
			logging.Debug("Not a semver version %s; skipping", version.Name)
		}
		result = append(result, semversion)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].LT(result[j])
	})

	return result, nil
}

var VersionMaster = semver.MustParse("0.0.0")

func ParseTargetFixVersion(issue *jira.Issue, availableVersions []semver.Version) (semver.Version, *jira.FixVersion, error) {
	if len(issue.Fields.FixVersions) < 1 {
		return semver.Version{}, nil, errs.New("Jira issue does not have a fixVersion assigned: %s\n", issue.Key)
	}

	if len(issue.Fields.FixVersions) > 1 {
		return semver.Version{}, nil, errs.New("Jira issue has multiple fixVersions assigned: %s. This is incompatible with our workflow.", issue.Key)
	}

	fixVersion := issue.Fields.FixVersions[0]
	if fixVersion.Archived != nil && *fixVersion.Archived || fixVersion.Released != nil && *fixVersion.Released {
		return semver.Version{}, nil, errs.New("fixVersion '%s' has either been archived or released\n", fixVersion.Name)
	}

	switch fixVersion.Name {
	case VersionNextFeasible:
		if len(availableVersions) < 1 {
			return semver.Version{}, nil, errs.New("No feasible versions available")
		}
		return availableVersions[0], fixVersion, nil
	case VersionNextUnscheduled:
		return VersionMaster, fixVersion, nil
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
