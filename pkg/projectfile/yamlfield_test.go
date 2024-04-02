package projectfile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_yamlField_update(t *testing.T) {
	type fields struct {
		field string
		value interface{}
	}
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"Add entry",
			fields{
				"version",
				1,
			},
			args{
				[]byte(`foo: bar`),
			},
			[]byte(`foo: bar
version: 1
`),
			assert.NoError,
		},
		{
			"Update entry",
			fields{
				"version",
				2,
			},
			args{
				[]byte(`foo: bar
version: 1`),
			},
			[]byte(`foo: bar
version: 2`),
			assert.NoError,
		},
		{
			"Update Wrapped entry",
			fields{
				"version",
				2,
			},
			args{
				[]byte(`key1: val1
version: 1
key2: val2`),
			},
			[]byte(`key1: val1
version: 2
key2: val2`),
			assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			y := &yamlField{
				field: tt.fields.field,
				value: tt.fields.value,
			}
			got, err := y.update(tt.args.data)
			if !tt.wantErr(t, err, fmt.Sprintf("update(%v)", tt.args.data)) {
				return
			}
			assert.Equalf(t, string(tt.want), string(got), "update(%v)", tt.args.data)
		})
	}
}
