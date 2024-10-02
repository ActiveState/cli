package buildscript

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
)

// TestUnmarshalBuildExpression tests that we can successfully read and convert Platform
// build expressions into build scripts.
func TestUnmarshalBuildExpression(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				filename: "buildexpression.json",
			},
			wantErr: false,
		},
		{
			name: "complex",
			args: args{
				filename: "buildexpression-complex.json",
			},
			wantErr: false,
		},
		{
			name: "unordered",
			args: args{
				filename: "buildexpression-unordered.json",
			},
			wantErr: false,
		},
		{
			name: "installer",
			args: args{
				filename: "buildexpression-installer.json",
			},
			wantErr: false,
		},
		{
			name: "installer-complex",
			args: args{
				filename: "buildexpression-installer-complex.json",
			},
			wantErr: false,
		},
		{
			name: "nested",
			args: args{
				filename: "buildexpression-nested.json",
			},
			wantErr: false,
		},
		{
			name: "alternate",
			args: args{
				filename: "buildexpression-alternate.json",
			},
			wantErr: false,
		},
		{
			name: "newObjects",
			args: args{
				filename: "buildexpression-new-objects.json",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "testdata", tt.args.filename))
			assert.NoError(t, err)

			_, err = UnmarshalBuildExpression(data, "", nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
