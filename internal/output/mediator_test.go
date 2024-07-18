package output

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
)

type testMediatorValue struct {
	response map[Format]interface{}
}

func (t *testMediatorValue) MarshalOutput(f Format) interface{} {
	return t.response[f]
}

func (t *testMediatorValue) MarshalStructured(f Format) interface{} {
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
		{
			"Test JSON",
			args{
				&testMediatorValue{
					map[Format]interface{}{
						JSONFormatName: "[1,2,3]",
					},
				},
				JSONFormatName,
			},
			"[1,2,3]",
		},
		{
			"Test No Structured Output",
			args{
				"unstructured",
				JSONFormatName,
			},
			StructuredError{Message: locale.Tr("err_unsupported_structured_output", constants.ForumsURL)},
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
