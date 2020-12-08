package model

import (
	"testing"
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
		indexed  indexedCommits
		first    string
		last     string
		want     int
		failType errorType
	}{
		"basic: none to last":     {basic, "", "e", 5, nil},
		"basic: first to none":    {basic, "a", "", 0, FailCommitCountImpossible},
		"basic: first to last":    {basic, "a", "e", 4, nil},
		"basic: first to second":  {basic, "a", "b", 1, nil},
		"basic: second to fourth": {basic, "b", "d", 2, nil},
		"basic: first to badval":  {basic, "a", "x", 0, FailCommitCountUnknowable},
		"basic: badval to last":   {basic, "x", "e", 0, FailCommitCountUnknowable},
		"split: none to last":     {split, "", "e", 3, nil},
		"split: first to none":    {split, "a", "", 0, FailCommitCountImpossible},
		"split: first to last":    {split, "a", "e", 2, nil},
		"split: first to second":  {split, "a", "b", 1, nil},
		"split: second to broken": {split, "b", "d", 0, FailCommitCountUnknowable},
	}

	for label, test := range tests {
		got, gotFail := test.indexed.countBetween(test.first, test.last)

		if test.failType != nil && !test.failType.Matches(gotFail.Type) {
			t.Errorf("%s: got %v, want %v", label, gotFail, test.failType)
		}
		if test.failType == nil && gotFail != nil {
			t.Errorf("%s: got %v, want %v", label, gotFail, test.failType)
		}

		if got != test.want {
			t.Errorf("%s: got %v, want %v", label, got, test.want)
		}
	}
}
