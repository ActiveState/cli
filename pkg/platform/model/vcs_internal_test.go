package model

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
)

func TestIndexedCommits(t *testing.T) {
	t.Run("countBetween", testIndexedCommitsCountBetween)
}

func testIndexedCommitsCountBetween(t *testing.T) {
	// linked data
	// a-b-c-d-e
	basic := indexedCommits{
		"e": "d",
		"d": "c",
		"c": "b",
		"b": "a",
		"a": "",
	}
	// linked data with split
	// a-b-c
	//  \
	//   d-e
	split := indexedCommits{
		"e": "d",
		"d": "a",
		"c": "b",
		"b": "a",
		"a": "",
	}

	tests := map[string]struct {
		indexed indexedCommits
		first   string
		last    string
		want    int
		wantErr bool
	}{
		"basic: none to last":     {basic, "", "e", 5, false},
		"basic: first to none":    {basic, "a", "", 0, true},
		"basic: first to last":    {basic, "a", "e", 4, false},
		"basic: first to second":  {basic, "a", "b", 1, false},
		"basic: second to fourth": {basic, "b", "d", 2, false},
		"basic: first to badval":  {basic, "a", "x", 0, true},
		"basic: badval to last":   {basic, "x", "e", 0, true},
		"split: none to last":     {split, "", "e", 3, false},
		"split: first to none":    {split, "a", "", 0, true},
		"split: first to last":    {split, "a", "e", 2, false},
		"split: first to second":  {split, "a", "b", 1, false},
		"split: second to broken": {split, "b", "d", 0, true},
	}

	for label, test := range tests {
		got, err := test.indexed.countBetween(test.first, test.last)
		gotErr := err != nil

		if test.wantErr != gotErr {
			t.Errorf("%s: got %v, want %v", label, gotErr, test.wantErr)
		}

		if got != test.want {
			t.Errorf("%s: got %v, want %v", label, got, test.want)
		}
	}
}

func cid(id int) strfmt.UUID {
	return strfmt.UUID(fmt.Sprintf("00000000-0000-0000-0000-%011d", id))
}

func cidp(id int) *strfmt.UUID {
	v := cid(id)
	return &v
}

func Test_commonParentWithHistory(t *testing.T) {
	type args struct {
		commit1  *strfmt.UUID
		commit2  *strfmt.UUID
		history1 []*mono_models.Commit
		history2 []*mono_models.Commit
	}
	tests := []struct {
		name string
		args args
		want *strfmt.UUID
	}{
		{
			"Same commits",
			args{
				cidp(1), cidp(1), nil, nil,
			},
			cidp(1),
		},
		{
			"One nil",
			args{
				cidp(1), nil, nil, nil,
			},
			nil,
		},
		{
			"Both nil",
			args{
				cidp(1), nil, nil, nil,
			},
			nil,
		},
		{
			"Commit2 is in Commit1 history",
			args{
				cidp(1), cidp(2), []*mono_models.Commit{{CommitID: cid(2)}}, nil,
			},
			cidp(2),
		},
		{
			"Commit1 is in Commit2 history",
			args{
				cidp(1), cidp(2), nil, []*mono_models.Commit{{CommitID: cid(1)}},
			},
			cidp(1),
		},
		{
			"Commit1 and Commit2 have common parent",
			args{
				cidp(1), cidp(2),
				[]*mono_models.Commit{
					{CommitID: cid(11)},
					{CommitID: cid(12)},
					{CommitID: cid(100)},
					{CommitID: cid(13)},
				},
				[]*mono_models.Commit{
					{CommitID: cid(21)},
					{CommitID: cid(22)},
					{CommitID: cid(23)},
					{CommitID: cid(100)},
				},
			},
			cidp(100),
		},
		{
			"Commit1 and Commit2 have no common parent",
			args{
				cidp(1), cidp(2),
				[]*mono_models.Commit{
					{CommitID: cid(11)},
					{CommitID: cid(12)},
					{CommitID: cid(13)},
				},
				[]*mono_models.Commit{
					{CommitID: cid(21)},
					{CommitID: cid(22)},
					{CommitID: cid(23)},
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commonParentWithHistory(tt.args.commit1, tt.args.commit2, tt.args.history1, tt.args.history2); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("commonParentWithHistory() = %v, want %v", got, tt.want)
			}
		})
	}
}