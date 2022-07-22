package workflow_helpers

import (
	"testing"
	"time"

	"github.com/google/go-github/v45/github"
	"github.com/stretchr/testify/require"
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
					t.Errorf("expected %s, got error: %v", tc.expected, err)
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
				t.Errorf("FetchPRs() error = %v", err)
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
				t.Errorf("FetchCommitsByRef() error = %v", err)
				return
			}
			if len(got) != tt.wantN {
				t.Errorf("FetchCommitsByRef() has %d results, want %d", len(got), tt.wantN)
			}
			for _, sha := range tt.wantSHAs {
				found := false
				for _, commit := range got {
					if commit.GetSHA() == sha {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("FetchCommitsByRef() did not return sha %s", sha)
				}
			}
		})
	}
}

func TestSearchGithubIssues(t *testing.T) {
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
				t.Errorf("SearchGithubIssues() error = %v", err)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchPRByTitle(tt.args.ghClient, tt.args.prName)
			if err != nil {
				t.Errorf("FetchPRByTitle() error = %v", err)
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
	jiraClient, err := InitJiraClient()
	require.NoError(t, err)

	versions, err := ActiveVersionsOnBranch(InitGHClient(), jiraClient, MasterBranch, time.Now().AddDate(0, -6, 0))
	if len(versions) < 2 {
		// Realistically we should have at least 2 versions in development in the past 6 months
		t.Errorf("ActiveVersionsOnBranch() returned %d versions, want at least 2", len(versions))
	}
}

func Test_sanitizeSearchTerm(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			"hello world",
			"hello world",
		},
		{
			"hello wo.rld",
			"hello",
		},
		{
			"hello .world",
			"hello",
		},
		{
			"hello world.",
			"hello",
		},
		{
			"hel.lo world",
			"world",
		},
		{
			"hel.lo wo.rld",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeSearchTerm(tt.input); got != tt.want {
				t.Errorf("sanitizeSearchTerm() = %v, want %v", got, tt.want)
			}
		})
	}
}
