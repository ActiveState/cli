package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func nilStr(s string) *string {
	return &s
}

func TestPlain_Print(t *testing.T) {
	type TableStruct struct {
		Header1 string
		Header2 *string
		Header3 *string
	}

	type TableStructs []*TableStruct

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
			"hello\n",
			"",
		},
		{
			"error string",
			args{errors.New("hello")},
			"hello\n",
			"",
		},
		{
			"int",
			args{1},
			"1\n",
			"",
		},
		{
			"float",
			args{1.1},
			"1.10\n",
			"",
		},
		{
			"boolean",
			args{true},
			"true\n",
			"",
		},
		{
			"slice with ints and floats",
			args{[]interface{}{
				int(1), int16(2), int32(3), int64(4),
				uint(5), uint16(6), uint32(7), uint64(8),
				float32(9.1), float64(10.1),
			}},
			" - 1\n - 2\n - 3\n - 4\n - 5\n - 6\n - 7\n - 8\n - 9.10\n - 10.10\n",
			"",
		},
		{
			"map with string, int and float",
			args{map[string]interface{}{
				"string": "hello",
				"int":    int(1),
				"float":  float32(9.1),
			}},
			"\n float: 9.10 \n int: 1 \n string: hello \n",
			"",
		},
		{
			"pointer",
			args{&struct{ V string }{"hello"}},
			"field_v: hello\n",
			"",
		},
		{
			"unexported",
			args{&struct{ v string }{"hello"}},
			"\n",
			"",
		},
		{
			"struct",
			args{struct {
				TestName string
				Value    string `locale:"test_value"`
				Field    string `locale:"localized_field"`
			}{
				"hello", "world", "value",
			}},
			"field_testname: hello\nfield_test_value: world\nLocalized Field: value\n",
			"",
		},
		{
			"complex mixed",
			args{
				struct {
					Value1 int
					Value2 float32
					Value3 bool
					Value4 []interface{}
					Value5 *TableStruct
					Value6 []TableStruct
					Value7 []*TableStruct
					Nil1   *int               // nil ptr to builtin
					Nil2   []interface{}      // nil slice
					Nil3   []interface{}      // slice of nils
					Nil4   *TableStruct       // nil ptr to struct
					Nil5   struct{ N *int }   // struct w/ ptr to builtin field
					Nil6   []struct{ N *int } // slice of structs w/ ptr to builtin field
					Nil7   interface{}        // typed nil
				}{
					1, 1.1, false,
					[]interface{}{
						1, true, 1.1, struct{ V string }{"value"}, []interface{}{1, 2},
					},
					&TableStruct{"AAA", nil, nilStr("CCC")},
					[]TableStruct{
						{"611", nilStr("622"), nil},
					},
					[]*TableStruct{
						{"711", nilStr("722"), nil},
					},
					nil,
					nil,
					[]interface{}{nil, nil, nil},
					nil,
					struct{ N *int }{nil},
					[]struct{ N *int }{
						{nil}, {nil}, {nil},
					},
					(*int)(nil),
				},
			},
			"field_value1: 1\n" +
				"field_value2: 1.10\n" +
				"field_value3: false\n" +
				"field_value4: \n - 1\n - true\n - 1.10\n - field_v: value\n - 1\n - 2\n" +
				"field_value5: \nfield_header1: AAA\nfield_header3: CCC\n" +
				"field_value6: \n" +
				"  field_header1    field_header2    field_header3  \n" +
				"───────────────────────────────────────────────────\n" +
				"  611              622              <nil>          \n" +
				"field_value7: \n" +
				"  field_header1    field_header2    field_header3  \n" +
				"───────────────────────────────────────────────────\n" +
				"  711              722              <nil>          \n" +
				"field_nil3: \n - <nil>\n - <nil>\n - <nil>\n" +
				"field_nil5: \n\n" +
				"field_nil6: \n" +
				"  field_n  \n" +
				"───────────\n" +
				"  <nil>    \n" +
				"  <nil>    \n" +
				"  <nil>    \n",
			"",
		},
		{
			"table",
			args{[]TableStruct{
				{"valueA.1", nil, nilStr("valueA.3")},
				{"valueB.1", nilStr("valueB.2"), nil},
				{"valueC.1", nilStr("valueC.2"), nilStr("valueC.3")},
			}},
			"  field_header1    field_header2    field_header3  \n" +
				"───────────────────────────────────────────────────\n" +
				"  valueA.1         <nil>            valueA.3       \n" +
				"  valueB.1         valueB.2         <nil>          \n" +
				"  valueC.1         valueC.2         valueC.3       \n",
			"",
		},
		{
			"table with pointers",
			args{[]*TableStruct{
				{"valueA.1", nil, nilStr("valueA.3")},
				{"valueB.1", nilStr("valueB.2"), nil},
				{"valueC.1", nilStr("valueC.2"), nilStr("valueC.3")},
			}},
			"  field_header1    field_header2    field_header3  \n" +
				"───────────────────────────────────────────────────\n" +
				"  valueA.1         <nil>            valueA.3       \n" +
				"  valueB.1         valueB.2         <nil>          \n" +
				"  valueC.1         valueC.2         valueC.3       \n",
			"",
		},
		{
			"table embed ptr non-slice with vert tag",
			args{
				struct {
					*TableStruct `opts:"verticalTable"`
				}{
					&TableStruct{"A", nilStr("B"), nilStr("C")},
				},
			},
			"  field_header1    A  \n" +
				"  field_header2    B  \n" +
				"  field_header3    C  \n",
			"",
		},
		{
			"table embed slice with vert tag",
			args{
				struct {
					TableStructs `opts:"verticalTable"`
				}{
					TableStructs{
						&TableStruct{"1A", nilStr("1B"), nilStr("1C")},
						&TableStruct{"2A", nilStr("2B"), nilStr("2C")},
					},
				},
			},
			"  field_header1    1A  \n" +
				"  field_header2    1B  \n" +
				"  field_header3    1C  \n" +
				"\n" +
				"  field_header1    2A  \n" +
				"  field_header2    2B  \n" +
				"  field_header3    2C  \n",
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

func TestPlain_Notice(t *testing.T) {
	type args struct {
		value interface{}
	}
	tests := []struct {
		name           string
		args           args
		expectedOut    string
		expectedNotice string
	}{
		{
			"simple string",
			args{"hello"},
			"",
			"hello\n",
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

			f.Notice(tt.args.value)
			assert.Equal(t, tt.expectedOut, outWriter.String(), "Output did not match")
			assert.Equal(t, tt.expectedNotice, errWriter.String(), "Notice did not match")
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
			"",
			"hello\n",
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

func Test_localizedField(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"Input locale",
			"localized_field",
			"Localized Field",
		},
		{
			"Input locale, nonexistant",
			"non_localized_field",
			"field_non_localized_field",
		},
		{
			"Input locale, nonexistant with fallback",
			"non_localized_field,fallback",
			"fallback",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := localizedField(tt.input); got != tt.want {
				t.Errorf("localizedField() = %v, want %v", got, tt.want)
			}
		})
	}
}
