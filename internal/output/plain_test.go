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
			"field_v: hello",
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
				Name  string
				Value string `locale:"value"`
				Field string `locale:"localized_field"`
			}{
				"hello", "world", "value",
			}},
			"field_name: hello\nfield_value: world\nLocalized Field: value",
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
			"field_value1: 1\n" +
				"field_value2: 1.10\n" +
				"field_value3: false\n" +
				"field_value4: \n - 1\n - true\n - 1.10\n - field_v: value\n - \n - 1\n - 2\n" +
				"field_value5: field_v: value",
			"",
		},
		{
			"table",
			args{[]struct {
				Header1 string
				Header2 string
				Header3 string
			}{
				{"valueA.1", "valueA.2", "valueA.3"},
				{"valueB.1", "valueB.2", "valueB.3"},
				{"valueC.1", "valueC.2", "valueC.3"},
			}},
			" field_header1       field_header2       field_header3    \n" +
				" valueA.1            valueA.2            valueA.3         \n" +
				" valueB.1            valueB.2            valueB.3         \n" +
				" valueC.1            valueC.2            valueC.3         \n",
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

func TestPlain_Error(t *testing.T) {
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

			f.Error(tt.args.value)
			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedErr, errWriter.String(), "Errors did not match")
		})
	}
}
