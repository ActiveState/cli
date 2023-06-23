package sync

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

func Test_mergeDimensions(t *testing.T) {
	type args struct {
		target *dimensions.Values
		dims   []*dimensions.Values
	}
	tests := []struct {
		name string
		args args
		want *dimensions.Values
	}{
		{
			"Sequence favours source",
			args{
				&dimensions.Values{
					Sequence: ptr.To(10),
				},
				[]*dimensions.Values{
					{
						Sequence: ptr.To(100),
					},
				},
			},
			&dimensions.Values{Sequence: ptr.To(100)},
		},
		{
			"Sequence favours source and accepts 0 value",
			args{
				&dimensions.Values{
					Sequence: ptr.To(10),
				},
				[]*dimensions.Values{
					{
						Sequence: ptr.To(0),
					},
				},
			},
			&dimensions.Values{Sequence: ptr.To(0)},
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
