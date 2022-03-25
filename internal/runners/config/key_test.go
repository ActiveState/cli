package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKey_Set(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		k       Key
		args    args
		wantErr bool
	}{
		{
			"empty",
			Key(""),
			args{""},
			true,
		},
		{
			"valid",
			Key(""),
			args{"validKey"},
			false,
		},
		{
			"invalid",
			Key(""),
			args{"invalid_key"},
			true,
		},
		{
			"valid with dot",
			Key(""),
			args{"valid.key"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.k.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Key.Set() error = %v, wantErr %v", err, tt.wantErr)
				assert.Equal(t, tt.args.v, tt.k.String())
			}
		})
	}
}
