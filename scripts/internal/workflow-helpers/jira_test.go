package workflow_helpers

import (
	"testing"

	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/stretchr/testify/require"
)

func TestFetchAvailableVersions(t *testing.T) {
	client, err := InitJiraClient()
	require.NoError(t, err)

	versions, err := FetchAvailableVersions(client)
	require.NoError(t, err)

	if len(versions) == 0 {
		t.Fatal("No versions found")
	}

	emptySemver := semver.Version{}
	for _, v := range versions {
		if emptySemver.EQ(v) {
			t.Errorf("Empty version found: %#v", v)
		}
	}
}

func TestParseTargetFixVersion(t *testing.T) {
	getIssue := func(fixVersion, desc string) *jira.Issue {
		return &jira.Issue{
			Fields: &jira.IssueFields{
				FixVersions: []*jira.FixVersion{
					{
						Name:        fixVersion,
						Description: desc,
					},
				},
			},
		}
	}

	type args struct {
		issue             *jira.Issue
		availableVersions []semver.Version
	}
	tests := []struct {
		name    string
		args    args
		want    semver.Version
		wantErr bool
	}{
		{
			name: "Next Feasible",
			args: args{
				getIssue("Next Feasible", "v1.2.5-RC1 -- bogus."),
				[]semver.Version{
					{Major: 1, Minor: 2, Patch: 3},
					{Major: 2, Minor: 3, Patch: 4},
					{Major: 1, Minor: 2, Patch: 5, Pre: []semver.PRVersion{{VersionStr: "RC1"}}},
				},
			},
			want: semver.Version{Major: 1, Minor: 2, Patch: 5, Pre: []semver.PRVersion{{VersionStr: "RC1"}}},
		},
		{
			name: "Next Unscheduled",
			args: args{
				getIssue("Next Unscheduled", ""),
				[]semver.Version{
					{Major: 1, Minor: 2, Patch: 3},
					{Major: 2, Minor: 3, Patch: 4},
					{Major: 1, Minor: 2, Patch: 5},
				},
			},
			want: VersionMaster,
		},
		{
			name: "Custom Version",
			args: args{
				getIssue("1.2.3", ""),
				[]semver.Version{},
			},
			want: semver.Version{Major: 1, Minor: 2, Patch: 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := ParseTargetFixVersion(tt.args.issue, tt.args.availableVersions)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTargetFixVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !got.EQ(tt.want) {
				t.Errorf("ParseTargetFixVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}
