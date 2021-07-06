package machineid_test

import (
	"errors"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/stretchr/testify/require"
)

func Test_uniqID(t *testing.T) {
	type args struct {
		machineIDGetter func() (string, error)
		uuidGetter      func() string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"UniqID from machineID",
			args{
				func() (string, error) { return "foo", nil },
				func() string { return "bar" },
			},
			"foo",
		},
		{
			"UniqID from fallback",
			args{
				func() (string, error) { return "foo", errors.New("") },
				func() string { return "bar" },
			},
			"bar",
		},
	}
	cfg, err := config.New()
	require.NoError(t, err)
	defer require.NoError(t, cfg.Close())
	machineid.SetConfiguration(cfg)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := machineid.UniqIDCustom(tt.args.machineIDGetter, tt.args.uuidGetter); got != tt.want {
				t.Errorf("uniqID() = %v, want %v", got, tt.want)
			}
		})
	}
}
