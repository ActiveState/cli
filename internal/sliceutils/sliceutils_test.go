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

func TestGetInt(t *testing.T) {
	type args struct {
		slice []int
		index int
	}
	tests := []struct {
		name  string
		args  args
		want  int
		want1 bool
	}{
		{
			"first entry",
			args{[]int{1, 2, 3}, 0},
			1,
			true,
		},
		{
			"last entry",
			args{[]int{1, 2, 3}, 2},
			3,
			true,
		},
		{
			"entry doesn't exist",
			args{[]int{1, 2, 3}, 4},
			-1,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetInt(tt.args.slice, tt.args.index)
			if got != tt.want {
				t.Errorf("GetInt() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetInt() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	type args struct {
		slice []string
		index int
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			"first entry",
			args{[]string{"a", "b", "c"}, 0},
			"a",
			true,
		},
		{
			"last entry",
			args{[]string{"a", "b", "c"}, 2},
			"c",
			true,
		},
		{
			"entry doesn't exist",
			args{[]string{"a", "b", "c"}, 4},
			"",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := GetString(tt.args.slice, tt.args.index)
			if got != tt.want {
				t.Errorf("GetString() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetString() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
