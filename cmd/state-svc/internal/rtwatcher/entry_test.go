package rtwatcher

import (
	"os"
	"testing"

	"github.com/ActiveState/cli/internal/analytics/dimensions"
)

func Test_entry_IsRunning(t *testing.T) {
	type fields struct {
		PID  int
		Exec string
		Dims *dimensions.Values
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "running",
			fields: fields{
				PID:  os.Getpid(),
				Exec: os.Args[0],
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "not running",
			fields: fields{
				PID:  123,
				Exec: "not running",
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := entry{
				PID:  tt.fields.PID,
				Exec: tt.fields.Exec,
				Dims: tt.fields.Dims,
			}
			got, err := e.IsRunning()
			if (err != nil) != tt.wantErr {
				t.Errorf("IsRunning() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsRunning() got = %v, want %v", got, tt.want)
			}
		})
	}
}
