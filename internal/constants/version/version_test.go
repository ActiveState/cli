package version

import (
	"testing"
)

func TestNumberIsProduction(t *testing.T) {
	tests := []struct {
		num  string
		want bool
	}{
		{"0.0.0", false},
		{"0.0.1", true},
		{"0.1.0", true},
		{"1.0.0", true},
		{"1.1.0", true},
		{"0.1.1", true},
		{"junk", false},
	}

	for _, tt := range tests {
		got := NumberIsProduction(tt.num)
		if got != tt.want {
			t.Errorf("%q: got %v, want %v", tt.num, got, tt.want)
		}
	}
}
