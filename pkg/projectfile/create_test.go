package projectfile

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
)

func Test_Create(t *testing.T) {
	var tempDir = fileutils.TempDirUnsafe()
	var uuid = strfmt.UUID("00010001-0001-0001-0001-000100010001")
	type args struct {
		org       string
		project   string
		directory string
		language  string
		commitID  *strfmt.UUID
	}
	tests := []struct {
		name         string
		args         args
		want         error
		wantContents string
	}{
		{
			"orgName/projName",
			args{"orgName", "projName", tempDir, "python3", &uuid},
			nil,
			"orgName/projName?branch=main",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Create(&CreateParams{
				Owner:     tt.args.org,
				Project:   tt.args.project,
				CommitID:  tt.args.commitID,
				Directory: tt.args.directory,
				Language:  tt.args.language,
			})
			assert.NoError(t, err)
			configFile := filepath.Join(tempDir, constants.ConfigFileName)
			assert.FileExists(t, configFile)
			assert.Contains(t, string(fileutils.ReadFileUnsafe(configFile)), tt.wantContents)

			// Verify .activestate/commit file was created with commitID
			commitIdFile := filepath.Join(tempDir, constants.ProjectConfigDirName, constants.CommitIdFileName)
			assert.FileExists(t, commitIdFile)
			assert.Equal(t, tt.args.commitID.String(), string(fileutils.ReadFileUnsafe(commitIdFile)))

			// Verify .gitignore was created with .activestate/commit entry (simulating fresh checkout)
			gitignoreFile := filepath.Join(tempDir, ".gitignore")
			assert.FileExists(t, gitignoreFile)
			assert.Contains(t, string(fileutils.ReadFileUnsafe(gitignoreFile)), fmt.Sprintf("%s/%s", constants.ProjectConfigDirName, constants.CommitIdFileName))
		})
	}
}
