package sync

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/rtutils/p"
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
					Sequence: p.IntP(10),
				},
				[]*dimensions.Values{
					{
						Sequence: p.IntP(100),
					},
				},
			},
			&dimensions.Values{Sequence: p.IntP(100)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeDimensions(tt.args.target, tt.args.dims...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeDimensions() = %v, want %v", got, tt.want)
			}
		})
	}
}
