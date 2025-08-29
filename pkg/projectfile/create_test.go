package projectfile

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Create(t *testing.T) {
	tempDir := fileutils.TempDirUnsafe()
	type args struct {
		org       string
		project   string
		directory string
		language  string
	}
	tests := []struct {
		name         string
		args         args
		want         error
		wantContents string
	}{
		{
			"orgName/projName",
			args{"orgName", "projName", tempDir, "python3"},
			nil,
			"orgName/projName?branch=main",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Create(&CreateParams{
				Owner:     tt.args.org,
				Project:   tt.args.project,
				Directory: tt.args.directory,
				Language:  tt.args.language,
				Host:      "test.example.com",
			})
			assert.NoError(t, err)
			configFile := filepath.Join(tempDir, constants.ConfigFileName)
			require.FileExists(t, configFile)
			assert.Contains(t, string(fileutils.ReadFileUnsafe(configFile)), tt.wantContents)
		})
	}
}
