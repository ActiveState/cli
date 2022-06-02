package jira_helpers

import (
	"os"

	"github.com/ActiveState/cli/internal/testhelpers/secrethelper"
	"github.com/andygrunwald/go-jira"
)

func InitClient() *jira.Client {
	username := secrethelper.GetSecretIfEmpty(os.Getenv("JIRA_USERNAME"), "user.JIRA_USERNAME")
	password := secrethelper.GetSecretIfEmpty(os.Getenv("JIRA_TOKEN"), "user.JIRA_TOKEN")

	tp := &jira.BasicAuthTransport{
		Username: username,
		Password: password,
	}
	jiraClient, _ := jira.NewClient(tp.Client(), "https://activestatef.atlassian.net/")
	return jiraClient
}
