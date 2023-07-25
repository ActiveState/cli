package workflow_helpers

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

func TestParseJiraKey(t *testing.T) {
	testCases := []struct {
		name     string
		msg      string
		expected string
	}{
		{
			name:     "empty",
			msg:      "",
			expected: "",
		},
		{
			name:     "no jira",
			msg:      "this is a commit message",
			expected: "",
		},
		{
			name:     "jira",
			msg:      "this is a commit message with a JIRA issue ID: DX-123",
			expected: "DX-123",
		},
		{
			name:     "jira with space",
			msg:      "this is a commit message with a JIRA issue ID: DX-123 and some other stuff",
			expected: "DX-123",
		},
		{
			name:     "jira with multiple",
			msg:      "this is a commit message with a JIRA issue ID: DX-123 and some other stuff and another JIRA issue ID: XYZ-456",
			expected: "DX-123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ParseJiraKey(tc.msg)
			if err != nil {
				if tc.expected != "" {
					t.Errorf("expected %s, got error: %s", tc.expected, errs.JoinMessage(err))
				}
				return
			}
			if actual != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, actual)
			}
		})
	}
}

func TestFetchPRs(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	type args struct {
		ghClient *github.Client
		cutoff   time.Time
		opts     *github.PullRequestListOptions
	}
	tests := []struct {
		name        string
		args        args
		wantResults bool
	}{
		{
			name: "success",
			args: args{
				ghClient: InitGHClient(),
				cutoff:   time.Now().AddDate(0, -1, 0),
				opts:     nil,
			},
			wantResults: true,
		},
		{
			name: "empty",
			args: args{
				ghClient: InitGHClient(),
				cutoff:   time.Now(),
				opts:     &github.PullRequestListOptions{},
			},
			wantResults: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchPRs(tt.args.ghClient, tt.args.cutoff, tt.args.opts)
			if err != nil {
				t.Errorf("FetchPRs() error = %s", errs.JoinMessage(err))
				return
			}
			hasResults := len(got) > 0
			if hasResults != tt.wantResults {
				t.Errorf("FetchPRs has %d results, but wantResults is %v", len(got), tt.wantResults)
			}
		})
	}
}

func TestFetchCommitsByRef(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	type args struct {
		ghClient *github.Client
		ref      string
		stop     func(commit *github.RepositoryCommit, count int) bool
	}
	tests := []struct {
		name     string
		args     args
		wantSHAs []string
		wantN    int
	}{
		{
			name: "master ref",
			args: args{
				ghClient: InitGHClient(),
				ref:      "master",
				stop:     func(commit *github.RepositoryCommit, count int) bool { return count > 0 },
			},
			wantN: 1,
		},
		{
			name: "sha ref",
			args: args{
				ghClient: InitGHClient(),
				ref:      "f52d7fa6b87ca15b1f0a726c062fd6a99d92a287",
				stop:     func(commit *github.RepositoryCommit, count int) bool { return count > 0 },
			},
			wantSHAs: []string{"f52d7fa6b87ca15b1f0a726c062fd6a99d92a287"},
			wantN:    1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			got, err := FetchCommitsByRef(tt.args.ghClient, tt.args.ref, func(commit *github.RepositoryCommit) bool {
				defer func() { count++ }()
				return tt.args.stop(commit, count)
			})
			if err != nil {
				t.Errorf("FetchCommitsByRef() error = %s", errs.JoinMessage(err))
				return
			}
			validateCommits(t, got, tt.wantSHAs, tt.wantN)
		})
	}
}

func TestSearchGithubIssues(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	type args struct {
		client *github.Client
		term   string
	}
	tests := []struct {
		name       string
		args       args
		want       []string // issue titles
		maxResults int      // This guards against runaway page iterations, feel free to tweak so long as this stays relatively low
	}{
		{
			name: "Search for Go 1.18 PR",
			args: args{
				client: InitGHClient(),
				term:   "is:pull-request Go 1.18",
			},
			want:       []string{"Go 1.18 compatibility"},
			maxResults: 20,
		},
		{
			name: "Readme issue",
			args: args{
				client: InitGHClient(),
				term:   "is:issue readme",
			},
			want:       []string{"README.md Development section clarifications"},
			maxResults: 5,
		},
		{
			name: "Version Prefix",
			args: args{
				client: InitGHClient(),
				term:   "is:pr " + VersionedPRPrefix,
			},
			want:       []string{"Version 0.34.0-RC1"},
			maxResults: 9999,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchGithubIssues(tt.args.client, tt.args.term)
			if err != nil {
				t.Errorf("SearchGithubIssues() error = %s", errs.JoinMessage(err))
				return
			}
			for _, title := range tt.want {
				found := false
				for _, issue := range got {
					if issue.GetTitle() == title {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SearchGithubIssues() did not return issue %s", title)
				}
			}
		})
	}
}

func TestFetchPRByTitle(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	type args struct {
		ghClient *github.Client
		prName   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Go 1.18 compatibility",
			args: args{
				ghClient: InitGHClient(),
				prName:   "Go 1.18 compatibility",
			},
			want: "Go 1.18 compatibility",
		},
		{
			name: "Non-existant",
			args: args{
				ghClient: InitGHClient(),
				prName:   "Non-existant",
			},
			want: "",
		},
		{
			name: "Version 0.34.0-RC1",
			args: args{
				ghClient: InitGHClient(),
				prName:   "Version 0.34.0-RC1",
			},
			want: "Version 0.34.0-RC1",
		},
		{
			name: "Version 0.40.0-RC1",
			args: args{
				ghClient: InitGHClient(),
				prName:   "Version 0.40.0-RC1",
			},
			want: "Version 0.40.0-RC1",
		},
		{
			name: "Version 0.40.0-RC2",
			args: args{
				ghClient: InitGHClient(),
				prName:   "Version 0.40.0-RC2",
			},
			want: "Version 0.40.0-RC2",
		},
		{
			name: "Version 0.40.0-RC3",
			args: args{
				ghClient: InitGHClient(),
				prName:   "Version 0.40.0-RC3",
			},
			want: "Version 0.40.0-RC3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchPRByTitle(tt.args.ghClient, tt.args.prName)
			if err != nil {
				t.Errorf("FetchPRByTitle() error = %s", errs.JoinMessage(err))
				return
			}
			if got == nil {
				if tt.want != "" {
					t.Errorf("FetchPRByTitle() got = nil, but want %s", tt.want)
				}
				return
			}
			if got.GetTitle() != tt.want {
				t.Errorf("FetchPRByTitle() got = %s, want %v", got.GetTitle(), tt.want)
			}
		})
	}
}

func TestActiveVersionsOnBranch(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	jiraClient, err := InitJiraClient()
	require.NoError(t, err)

	versions, err := ActiveVersionsOnBranch(InitGHClient(), jiraClient, MasterBranch, time.Now().AddDate(0, -6, 0))
	require.NoError(t, err, errs.JoinMessage(err))
	if len(versions) < 2 {
		// Realistically we should have at least 2 versions in development in the past 6 months
		t.Errorf("ActiveVersionsOnBranch() returned %d versions, want at least 2", len(versions))
	}
}

func TestFetchCommitsByShaRange(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	type args struct {
		ghClient *github.Client
		startSha string
		stopSha  string
	}
	tests := []struct {
		name     string
		args     args
		wantSHAs []string
		wantN    int
	}{
		{
			name: "small range",
			args: args{
				ghClient: InitGHClient(),
				startSha: "97cc4d358ba249493222cd2e8928015714881000",
				stopSha:  "69bbdf1466135094efe0ef77108eae9953d76ac3",
			},
			wantSHAs: []string{"a2fe40506b564ab00b1fe46e2bd170898c46244b"},
			wantN:    -1,
		},
		{
			name: "large range",
			args: args{
				ghClient: InitGHClient(),
				startSha: "5d6e103384849ad2cb6f604da84c4dc9f2245c31",
				stopSha:  "69bbdf1466135094efe0ef77108eae9953d76ac3",
			},
			wantSHAs: []string{"a2fe40506b564ab00b1fe46e2bd170898c46244b"},
			wantN:    -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchCommitsByShaRange(tt.args.ghClient, tt.args.startSha, tt.args.stopSha)
			if err != nil {
				t.Errorf("FetchCommitsByShaRange() error = %s", errs.JoinMessage(err))
				return
			}
			validateCommits(t, got, tt.wantSHAs, tt.wantN)
		})
	}
}

func TestUpdatePRTargetBranch(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	err := UpdatePRTargetBranch(InitGHClient(), 1985, "version/0-40-0-RC2")
	require.NoError(t, err, errs.JoinMessage(err))
}

func TestCreateBranch(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	prefix := funk.RandomString(10, []rune("abcdefghijklmnopqrstuvwxyz0123456789"))
	name := prefix + "/" + funk.RandomString(10, []rune("abcdefghijklmnopqrstuvwxyz0123456789"))
	fmt.Printf("Creating branch %s\n", name)
	err := CreateBranch(InitGHClient(), name, "f8a9465c572ed7a26145c7ebf961554da9367ec7")
	require.NoError(t, err, errs.JoinMessage(err))
}

func validateCommits(t *testing.T, commits []*github.RepositoryCommit, wantSHAs []string, wantN int) {
	if wantN != -1 && len(commits) != wantN {
		t.Errorf("FetchCommitsByRef() has %d results, want %d", len(commits), wantN)
	}
	for _, sha := range wantSHAs {
		found := false
		for _, commit := range commits {
			if commit.GetSHA() == sha {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FetchCommitsByRef() did not return sha %s (got %d commits)", sha, len(commits))
		}
	}
}

func TestBehindBy(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	type args struct {
		client *github.Client
		base   string
		head   string
	}
	tests := []struct {
		name       string
		args       args
		wantBehind bool
		wantErr    bool
	}{
		{
			"Should be behind",
			args{
				InitGHClient(),
				"version/0-39-0-RC2",
				"version/0-39-0-RC1",
			},
			true,
			false,
		},
		{
			"Should not be behind",
			args{
				InitGHClient(),
				"version/0-39-0-RC1",
				"version/0-39-0-RC2",
			},
			false,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCommitsBehind(tt.args.client, tt.args.base, tt.args.head)
			if (err != nil) != tt.wantErr {
				t.Errorf("BehindBy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (len(got) > 0) != tt.wantBehind {
				t.Errorf("BehindBy() got = %v, want %v", len(got), tt.wantBehind)
			}
		})
	}
}

func TestFetchVersionPR(t *testing.T) {
	t.Skip("For debugging purposes, comment this line out if you want to test this locally")

	type args struct {
		ghClient         *github.Client
		assert           Assertion
		versionToCompare semver.Version
	}
	tests := []struct {
		name      string
		args      args
		wantTitle string
		wantErr   bool
	}{
		{
			"Previous Version",
			args{
				InitGHClient(),
				AssertLT,
				semver.MustParse("0.39.0-RC2"),
			},
			"Version 0.39.0-RC1",
			false,
		},
		{
			"Next Version",
			args{
				InitGHClient(),
				AssertGT,
				semver.MustParse("0.39.0-RC1"),
			},
			"Version 0.39.0-RC2",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchVersionPR(tt.args.ghClient, tt.args.assert, tt.args.versionToCompare)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchVersionPR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.GetTitle(), tt.wantTitle) {
				t.Errorf("FetchVersionPR() got = %v, want %v", got.GetTitle(), tt.wantTitle)
			}
		})
	}
}
