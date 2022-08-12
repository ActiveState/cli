package wc

import (
	"testing"

	wh "github.com/ActiveState/cli/scripts/internal/workflow-helpers"
	"github.com/blang/semver"
	"github.com/stretchr/testify/require"
)

type testMeta struct {
	version           semver.Version
	jiraVersion       string
	versionBranchName string
	versionPRName     string
}

func (t testMeta) GetVersion() semver.Version {
	return t.version
}

func (t testMeta) GetJiraVersion() string {
	return t.jiraVersion
}

func (t testMeta) GetVersionBranchName() string {
	return t.versionBranchName
}

func (t testMeta) GetVersionPRName() string {
	return t.versionPRName
}

func TestDetectBaseRef(t *testing.T) {
	ghClient := wh.InitGHClient()
	jiraClient, err := wh.InitJiraClient()
	require.NoError(t, err)
	ref, err := DetectBaseRef(ghClient, jiraClient, &testMeta{
		// This test is using hard coded string values as it's meant to protect against regressions
		version:           semver.MustParse("0.34.1-RC1"),
		jiraVersion:       "v0.34.1-RC1",
		versionBranchName: "version/0-34-1-RC1",
		versionPRName:     "Version 0.34.1-RC1",
	})
	require.NoError(t, err)
	require.Equal(t, "86464907122e34baadeb79f4b989f0f097590174", ref)
}
