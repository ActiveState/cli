package output

import (
	"reflect"
	"testing"
)

type testMediatorValue struct {
	response map[Format]interface{}
}

func (t *testMediatorValue) MarshalOutput(f Format) interface{} {
	return t.response[f]
}

func Test_mediatorValue(t *testing.T) {
	type args struct {
		value  interface{}
		format Format
	}
	tests := []struct {
		name string
		args args
		want interface{}
	}{
		{
			"Test Plain",
			args{
				"hello",
				PlainFormatName,
			},
			"hello",
		},
		{
			"Test Marshal",
			args{
				&testMediatorValue{
					map[Format]interface{}{
						PlainFormatName: "mediated value",
					},
				},
				PlainFormatName,
			},
			"mediated value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mediatorValue(tt.args.value, tt.args.format); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mediatorValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
