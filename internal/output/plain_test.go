package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlain_Print(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name        string
		args        args
		expectedOut string
		expectedErr string
	}{
		{
			"simple string",
			args{"hello"},
			"hello",
			"",
		},
		{
			"int",
			args{1},
			"1",
			"",
		},
		{
			"float",
			args{1.1},
			"1.10",
			"",
		},
		{
			"boolean",
			args{true},
			"true",
			"",
		},
		{
			"slice with ints and floats",
			args{[]interface{}{
				int(1), int16(2), int32(3), int64(4),
				uint(5), uint16(6), uint32(7), uint64(8),
				float32(9.1), float64(10.1),
			}},
			"\n - 1\n - 2\n - 3\n - 4\n - 5\n - 6\n - 7\n - 8\n - 9.10\n - 10.10",
			"",
		},
		{
			"pointer",
			args{&struct{ V string }{"hello"}},
			"v: hello",
			"",
		},
		{
			"unexported",
			args{&struct{ v string }{"hello"}},
			"",
			"",
		},
		{
			"struct",
			args{struct {
				Tt_Name string
				Value   string `serialized:"tt_value"`
				Field   string `serialized:"localized_field"`
			}{
				"hello", "world", "value",
			}},
			"tt_Name: hello\ntt_value: world\nLocalized Field: value",
			"",
		},
		{
			"complex mixed",
			args{struct {
				Value1 int
				Value2 float32
				Value3 bool
				Value4 []interface{}
				Value5 struct{ V string }
			}{
				1, 1.1, false,
				[]interface{}{
					1, true, 1.1, struct{ V string }{"value"}, []interface{}{1, 2},
				},
				struct{ V string }{"value"},
			}},
			"value1: 1\nvalue2: 1.10\nvalue3: false\nvalue4: \n - 1\n - true\n - 1.10\n - v: value\n - \n - 1\n - 2\nvalue5: v: value",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outWriter := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			f := &Plain{&Config{
				OutWriter:   outWriter,
				ErrWriter:   errWriter,
				Colored:     false,
				Interactive: false,
			}}

			f.Print(tt.args.value)
			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedErr, errWriter.String(), "Errors did not match")
		})
	}
}
