package sliceutils

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveFromStrings(t *testing.T) {
	type args struct {
		slice []string
		ns    []int
	}

	simpleSlice := []string{"0", "1", "2", "3"}
	integers := func(ns ...int) []int { return ns }

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"first index",
			args{simpleSlice, integers(0)},
			[]string{"1", "2", "3"},
		},
		{
			"last index",
			args{simpleSlice, integers(len(simpleSlice) - 1)},
			[]string{"0", "1", "2"},
		},
		{
			"inner index",
			args{simpleSlice, integers(1)},
			[]string{"0", "2", "3"},
		},
		{
			"multiple indexes",
			args{simpleSlice, integers(1, len(simpleSlice)-1)},
			[]string{"0", "2"},
		},
		{
			"index too low",
			args{simpleSlice, integers(-1)},
			simpleSlice,
		},
		{
			"index too high",
			args{simpleSlice, integers(42)},
			simpleSlice,
		},
		{
			"nil indexes",
			args{simpleSlice, nil},
			simpleSlice,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RemoveFromStrings(tt.args.slice, tt.args.ns...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
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

func TestRange(t *testing.T) {
	type args struct {
		in    []int
		start int
		end   int
	}
	tests := []struct {
		name string
		args args
		want []int
	}{
		{
			"Slice is smaller than range",
			args{
				[]int{1, 2, 3},
				1,
				5,
			},
			[]int{2, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntRangeUncapped(tt.args.in, tt.args.start, tt.args.end)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Range() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInsertStringAt(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c", "d", "e", "f"}, InsertStringAt([]string{"a", "b", "c", "e", "f"}, 3, "d"))
}

func TestPop(t *testing.T) {
	type args struct {
		data []any
	}
	tests := []struct {
		name       string
		args       args
		wantValue  any
		wantResult []any
		wantErr    bool
	}{
		{
			name: "string",
			args: args{
				data: []any{"a", "b", "c"},
			},
			wantValue:  "c",
			wantResult: []any{"a", "b"},
			wantErr:    false,
		},
		{
			name: "int",
			args: args{
				data: []any{1, 2, 3},
			},
			wantValue:  3,
			wantResult: []any{1, 2},
			wantErr:    false,
		},
		{
			name: "empty",
			args: args{
				data: []any{},
			},
			wantValue:  nil,
			wantResult: nil,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotSlice, err := Pop(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Pop() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotValue, tt.wantValue) {
				t.Errorf("Pop() got value = %v, want %v", gotValue, tt.wantValue)
			}
			if !reflect.DeepEqual(gotSlice, tt.wantResult) {
				t.Errorf("Pop() got slice = %v, want %v", gotSlice, tt.wantResult)
			}
		})
	}
}

func TestContains(t *testing.T) {
	type args struct {
		data []any
		v    any
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "string contains",
			args: args{
				data: []any{"a", "b", "c"},
				v:    "b",
			},
			want: true,
		},
		{
			name: "string doesn't contain",
			args: args{
				data: []any{"a", "b", "c"},
				v:    "d",
			},
			want: false,
		},
		{
			name: "int contains",
			args: args{
				data: []any{1, 2, 3},
				v:    2,
			},
			want: true,
		},
		{
			name: "int doesn't contain",
			args: args{
				data: []any{1, 2, 3},
				v:    4,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Contains(tt.args.data, tt.args.v); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUniqueByProperty(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3, 4}, UniqueByProperty([]int{1, 2, 3, 3, 4, 4, 2, 4}, func(v int) any { return v }))

	type s struct{ name string }
	assert.Equal(t,
		[]s{{"a"}, {"b"}, {"c"}},
		UniqueByProperty([]s{{"a"}, {"a"}, {"b"}, {"c"}}, func(v s) any { return v.name }),
	)
}

func TestToLookupMapByKey(t *testing.T) {
	type customType struct {
		Key   string
		Value string
	}
	v1 := &customType{"Key1", "Val1"}
	v2 := &customType{"Key2", "Val2"}
	v3 := &customType{"Key3", "Val3"}
	lookupSlice := []*customType{v1, v2, v3}
	lookupMap := ToLookupMapByKey(lookupSlice, func(v *customType) string { return v.Key })
	assert.Equal(t, map[string]*customType{
		"Key1": v1,
		"Key2": v2,
		"Key3": v3,
	}, lookupMap)
}

func TestEqualValues(t *testing.T) {
	tests := []struct {
		name       string
		comparison func() bool
		expected   bool
	}{
		{
			name: "Equal int slices",
			comparison: func() bool {
				return EqualValues([]int{1, 2, 3}, []int{3, 2, 1})
			},
			expected: true,
		},
		{
			name: "Unequal int slices",
			comparison: func() bool {
				return EqualValues([]int{1, 2, 3}, []int{4, 5, 6})
			},
			expected: false,
		},
		{
			name: "Equal string slices",
			comparison: func() bool {
				return EqualValues([]string{"a", "b", "c"}, []string{"c", "b", "a"})
			},
			expected: true,
		},
		{
			name: "Unequal string slices",
			comparison: func() bool {
				return EqualValues([]string{"a", "b", "c"}, []string{"a", "b", "d"})
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.comparison()
			if result != tt.expected {
				t.Errorf("%s = %v; want %v", tt.name, result, tt.expected)
			}
		})
	}
}
