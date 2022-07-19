package workflow_helpers

import (
	"testing"

	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"github.com/thoas/go-funk"
)

func Test_issueWithVersionLT(t *testing.T) {
	type args struct {
		issues              []*github.Issue
		lessThanThisVersion semver.Version
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "no version",
			args: args{
				issues: []*github.Issue{
					{
						Title:  github.String("this is a commit message"),
						Number: github.Int(1),
					},
				},
				lessThanThisVersion: semver.MustParse("1.2.3"),
			},
			want: 0,
		},
		{
			name: "higher version",
			args: args{
				issues: funk.Shuffle([]*github.Issue{
					{
						Title:  github.String(VersionedPRPrefix + "1.2.4"),
						Number: github.Int(1),
					},
					{
						Title:  github.String(VersionedPRPrefix + "2.3.4"),
						Number: github.Int(2),
					},
				}).([]*github.Issue),
				lessThanThisVersion: semver.MustParse("1.2.3"),
			},
			want: 0,
		},
		{
			name: "matching lower version",
			args: args{
				issues: funk.Shuffle([]*github.Issue{
					{
						Title:  github.String(VersionedPRPrefix + "1.2.2"),
						Number: github.Int(1),
					},
					{
						Title:  github.String(VersionedPRPrefix + "2.3.4"),
						Number: github.Int(2),
					},
				}).([]*github.Issue),
				lessThanThisVersion: semver.MustParse("1.2.3"),
			},
			want: 1,
		},
		{
			name: "multiple matching lower versions",
			args: args{
				issues: funk.Shuffle([]*github.Issue{
					{
						Title:  github.String(VersionedPRPrefix + "1.2.2"),
						Number: github.Int(1),
					},
					{
						Title:  github.String(VersionedPRPrefix + "2.3.4"),
						Number: github.Int(2),
					},
					{
						Title:  github.String(VersionedPRPrefix + "1.0.0"),
						Number: github.Int(1),
					},
				}).([]*github.Issue),
				lessThanThisVersion: semver.MustParse("1.2.3"),
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := issueWithVersionLT(tt.args.issues, tt.args.lessThanThisVersion); got.GetNumber() != tt.want {
				t.Errorf("issueWithVersionLT() = %v, want %v", got.GetNumber(), tt.want)
			}
		})
	}
}
