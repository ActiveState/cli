package workflow_helpers

import (
	"reflect"
	"testing"

	"github.com/blang/semver"
	"github.com/google/go-github/v45/github"
	"github.com/thoas/go-funk"
)

func Test_issueWithVersionAssert(t *testing.T) {
	type args struct {
		issues           []*github.Issue
		assertion        Assertion
		versionToCompare semver.Version
	}
	tests := []struct {
		name string
		args args
		want []int
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
				assertion:        AssertLT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{},
		},
		{
			name: "Only higher versions when asserting LT",
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
				assertion:        AssertLT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{},
		},
		{
			name: "Don't include version being compared with LT",
			args: args{
				issues: funk.Shuffle([]*github.Issue{
					{
						Title:  github.String(VersionedPRPrefix + "1.2.2"),
						Number: github.Int(1),
					},
					{
						Title:  github.String(VersionedPRPrefix + "1.2.3"),
						Number: github.Int(2),
					},
				}).([]*github.Issue),
				assertion:        AssertLT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{1},
		},
		{
			name: "Matching version when asserting LT",
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
				assertion:        AssertLT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{1},
		},
		{
			name: "Multiple matching version when asserting LT",
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
						Number: github.Int(3),
					},
				}).([]*github.Issue),
				assertion:        AssertLT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{1, 3}, // Should be ordered by closest matching
		},
		{
			name: "Only lower versions when asserting GT",
			args: args{
				issues: funk.Shuffle([]*github.Issue{
					{
						Title:  github.String(VersionedPRPrefix + "1.2.1"),
						Number: github.Int(1),
					},
					{
						Title:  github.String(VersionedPRPrefix + "0.1.2"),
						Number: github.Int(2),
					},
				}).([]*github.Issue),
				assertion:        AssertGT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{},
		},
		{
			name: "Matching version when asserting GT",
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
				assertion:        AssertGT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{2},
		},
		{
			name: "Don't include version being compared with GT",
			args: args{
				issues: funk.Shuffle([]*github.Issue{
					{
						Title:  github.String(VersionedPRPrefix + "1.2.4"),
						Number: github.Int(1),
					},
					{
						Title:  github.String(VersionedPRPrefix + "1.2.3"),
						Number: github.Int(2),
					},
				}).([]*github.Issue),
				assertion:        AssertGT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{1},
		},
		{
			name: "Multiple matching version when asserting GT",
			args: args{
				issues: funk.Shuffle([]*github.Issue{
					{
						Title:  github.String(VersionedPRPrefix + "1.2.2"),
						Number: github.Int(1),
					},
					{
						Title:  github.String(VersionedPRPrefix + "5.4.3"),
						Number: github.Int(2),
					},
					{
						Title:  github.String(VersionedPRPrefix + "2.3.4"),
						Number: github.Int(3),
					},
				}).([]*github.Issue),
				assertion:        AssertGT,
				versionToCompare: semver.MustParse("1.2.3"),
			},
			want: []int{3, 2}, // Should be ordered by closest matching
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := issuesWithVersionAssert(tt.args.issues, tt.args.assertion, tt.args.versionToCompare)
			gotN := []int{}
			for _, v := range got {
				gotN = append(gotN, v.GetNumber())
			}
			if !reflect.DeepEqual(gotN, tt.want) {
				t.Errorf("issueWithVersionLT() = %v, want %v", gotN, tt.want)
			}
		})
	}
}
