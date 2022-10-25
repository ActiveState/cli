package workflow_helpers

import (
	"testing"

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
