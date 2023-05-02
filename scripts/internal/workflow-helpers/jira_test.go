package workflow_helpers

import (
	"testing"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/andygrunwald/go-jira"
	"github.com/blang/semver"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

func TestFetchJiraIDsInCommits(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	tests := []struct {
		name        string
		commitRange []string
		wantFound   []string
	}{
		{
			"Find DX-946 - small range",
			[]string{"98543a901b11c36c5255b9ffb748259789918644", "69bbdf1466135094efe0ef77108eae9953d76ac3"},
			[]string{"DX-946"},
		},
		{
			"Find DX-946 - large range",
			[]string{"5d6e103384849ad2cb6f604da84c4dc9f2245c31", "c07df2b7df591bf2c84306bb79f9a57f9cd9aaa4"},
			[]string{"DX-946"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commits, err := FetchCommitsByShaRange(InitGHClient(), tt.commitRange[0], tt.commitRange[1])
			if err != nil {
				t.Errorf("FetchCommitsByShaRange() error = %v", err)
				return
			}
			gotFound := FetchJiraIDsInCommits(commits)
			for _, want := range tt.wantFound {
				if !funk.Contains(gotFound, want) {
					t.Errorf("FetchJiraIDsInCommits() did not include %s, got: %v", want, gotFound)
				}
			}
		})
	}
}

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
		if emptySemver.EQ(v.Version) {
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
		availableVersions []Version
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
				[]Version{
					{semver.Version{Major: 1, Minor: 2, Patch: 3}, ""},
					{semver.Version{Major: 2, Minor: 3, Patch: 4}, ""},
					{semver.Version{Major: 1, Minor: 2, Patch: 5, Pre: []semver.PRVersion{{VersionStr: "RC1"}}}, ""},
				},
			},
			want: semver.Version{Major: 1, Minor: 2, Patch: 5, Pre: []semver.PRVersion{{VersionStr: "RC1"}}},
		},
		{
			name: "Next Unscheduled",
			args: args{
				getIssue("Next Unscheduled", ""),
				[]Version{
					{semver.Version{Major: 1, Minor: 2, Patch: 3}, ""},
					{semver.Version{Major: 2, Minor: 3, Patch: 4}, ""},
					{semver.Version{Major: 1, Minor: 2, Patch: 5}, ""},
				},
			},
			want: VersionMaster,
		},
		{
			name: "Custom Version",
			args: args{
				getIssue("1.2.3", ""),
				[]Version{},
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

func TestJqlUnpaged(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	client, err := InitJiraClient()
	require.NoError(t, err)

	issues, err := JqlUnpaged(client, "project = DX AND status=Done ORDER BY created")
	require.NoError(t, err, errs.JoinMessage(err))
	require.Greater(t, len(issues), 0)
}

func TestUpdateJiraStatus(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally -- THIS WILL MAKE CHANGES TO THE TARGET ISSUE")
	c, err := InitJiraClient()
	require.NoError(t, err)
	type args struct {
		client     *jira.Client
		issue      *jira.Issue
		statusName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"foo",
			args{
				c,
				&jira.Issue{
					ID:  "33792",
					Key: "DX-1584",
				},
				"In Progress",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateJiraStatus(tt.args.client, tt.args.issue, tt.args.statusName); (err != nil) != tt.wantErr {
				t.Errorf("UpdateJiraStatus() error = %s, wantErr %v", errs.JoinMessage(err), tt.wantErr)
			}
		})
	}
}
