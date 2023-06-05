package main

import (
	"os"
	"reflect"
	"testing"
)

func Test_transformedEnv(t *testing.T) {
	I := string(os.PathListSeparator)
	type args struct {
		sourceEnv    []string
		updatesToEnv []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"Simple append",
			args{
				sourceEnv:    []string{"A=1", "B=2"},
				updatesToEnv: []string{"C=3"},
			},
			[]string{"A=1", "B=2", "C=3"},
		},
		{
			"Path append",
			args{
				sourceEnv:    []string{"PATH=A" + I + "B"},
				updatesToEnv: []string{"PATH=C"},
			},
			[]string{"PATH=C" + I + "A" + I + "B"},
		},
		{
			"Path append with missmatched casing",
			args{
				sourceEnv:    []string{"Path=A" + I + "B"},
				updatesToEnv: []string{"PATH=C"},
			},
			[]string{"PATH=C" + I + "A" + I + "B"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transformedEnv(tt.args.sourceEnv, tt.args.updatesToEnv); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("transformedEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
