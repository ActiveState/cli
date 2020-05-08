package output

import (
	"reflect"
	"testing"
)

func Test_parseStructMeta(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    structMeta
		wantErr bool
	}{
		{
			"Parses struct with tags",

			struct {
				Key string `locale:"label,Label" opts:"opt"`
			}{"value"},
			[]structField{{
				"Key",
				"label,Label",
				[]string{"opt"},
				"value",
			}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStructMeta(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStructMeta() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStructMeta() got = %v, want %v", got, tt.want)
			}
		})
	}
}
