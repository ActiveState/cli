package sliceutils

import (
	"reflect"
	"testing"
)

func TestRemoveFromStrings(t *testing.T) {
	type args struct {
		slice []string
		n     int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"Removes Index",
			args{
				[]string{"1", "2", "3"},
				1,
			},
			[]string{"1", "3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveFromStrings(tt.args.slice, tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveFromStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}
