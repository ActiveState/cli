//go:build darwin
// +build darwin

package autostart

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestIsLegacyPlist(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "legacy plist",
			args: args{
				filename: "legacy.plist",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "current plist",
			args: args{
				filename: "current.plist",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "non-existent plist",
			args: args{
				filename: "not-there.plist",
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := environment.GetRootPath()
			assert.NoError(t, err, "should detect root path")

			path := filepath.Join(root, "internal", "osutils", "autostart", "testdata", tt.args.filename)
			got, err := isLegacyPlist(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("isLegacyPlist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isLegacyPlist() = %v, want %v", got, tt.want)
			}
		})
	}
}
