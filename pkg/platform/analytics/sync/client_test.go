package sync

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/rtutils/p"
)

func Test_mergeDimensions(t *testing.T) {
	type args struct {
		target *analytics.Dimensions
		dims   []*analytics.Dimensions
	}
	tests := []struct {
		name string
		args args
		want *analytics.Dimensions
	}{
		{
			"Sequence favours source",
			args{
				&analytics.Dimensions{
					Sequence: p.IntP(10),
				},
				[]*analytics.Dimensions{
					{
						Sequence: p.IntP(100),
					},
				},
			},
			&analytics.Dimensions{Sequence: p.IntP(100)},
		},
		{
			"Sequence favours source and accepts 0 value",
			args{
				&analytics.Dimensions{
					Sequence: p.IntP(10),
				},
				[]*analytics.Dimensions{
					{
						Sequence: p.IntP(0),
					},
				},
			},
			&analytics.Dimensions{Sequence: p.IntP(0)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := *tt.args.target.Sequence
			if got := mergeDimensions(tt.args.target, tt.args.dims...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeDimensions() = %#v, want %#v", got, tt.want)
			}
			if *tt.args.target.Sequence != before {
				t.Errorf("Target struct should not have been modified")
			}
		})
	}
}
