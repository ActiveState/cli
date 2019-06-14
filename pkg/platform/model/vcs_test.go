package model

import "testing"

func TestIndexedCommits(t *testing.T) {
	t.Run("countBetween", testIndexedCommitsCountBetween)
}

func testIndexedCommitsCountBetween(t *testing.T) {
	basic := indexedCommits{
		"e": "d",
		"d": "c",
		"c": "b",
		"b": "a",
		"a": "",
	}
	split := indexedCommits{
		"e": "d",
		"d": "a",
		"c": "b",
		"b": "a",
		"a": "",
	}

	tests := map[string]struct {
		ocs   indexedCommits
		first string
		last  string
		want  int
		werr  bool
	}{
		"basic: none to last":     {basic, "", "e", 5, false},
		"basic: first to none":    {basic, "a", "", -1, true},
		"basic: first to last":    {basic, "a", "e", 4, false},
		"basic: first to second":  {basic, "a", "b", 1, false},
		"basic: second to fourth": {basic, "b", "d", 2, false},
		"basic: first to badval":  {basic, "a", "x", 0, true},
		"basic: badval to last":   {basic, "x", "e", 0, true},
		"split: none to last":     {split, "", "e", 3, false},
		"split: first to none":    {split, "a", "", -1, true},
		"split: first to last":    {split, "a", "e", 2, false},
		"split: first to second":  {split, "a", "b", 1, false},
		"split: second to broken": {split, "b", "d", 0, true},
	}

	efmt := "%s: got %v, want %v"

	for tn, tt := range tests {
		got, gerr := tt.ocs.countBetween(tt.first, tt.last)
		if tt.werr && gerr == nil {
			t.Errorf(efmt, tn, "nil", "error")
		}
		if !tt.werr && gerr != nil {
			t.Errorf(efmt, tn, gerr, "nil")
		}

		if got != tt.want {
			t.Errorf(efmt, tn, got, tt.want)
		}
	}
}
