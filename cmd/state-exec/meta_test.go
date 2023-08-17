package main

import (
	"os"
	"reflect"
	"testing"
)

func Test_transformedEnv(t *testing.T) {
	I := string(os.PathListSeparator)
	type args struct {
		sourceEnv         []string
		updatesToEnv      []string
		onlyTransformPath bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"Simple append",
			args{
				sourceEnv:         []string{"A=1", "B=2"},
				updatesToEnv:      []string{"C=3"},
				onlyTransformPath: false,
			},
			[]string{"A=1", "B=2", "C=3"},
		},
		{
			"Path append",
			args{
				sourceEnv:         []string{"PATH=A" + I + "B"},
				updatesToEnv:      []string{"PATH=C"},
				onlyTransformPath: false,
			},
			[]string{"PATH=C" + I + "A" + I + "B"},
		},
		{
			"Path append with missmatched casing",
			args{
				sourceEnv:         []string{"Path=A" + I + "B"},
				updatesToEnv:      []string{"PATH=C"},
				onlyTransformPath: false,
			},
			[]string{"PATH=C" + I + "A" + I + "B"},
		},
		{
			"Only update path",
			args{
				sourceEnv:         []string{"PATH=A", "B=2", "C=3"},
				updatesToEnv:      []string{"PATH=D", "B=20", "C=30"},
				onlyTransformPath: true,
			},
			[]string{"PATH=D" + I + "A", "B=2", "C=3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transformedEnv(tt.args.sourceEnv, tt.args.updatesToEnv, tt.args.onlyTransformPath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("transformedEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
