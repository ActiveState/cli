package model

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/assert"
)

func TestProcessCommitError(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no change since last commit",
			args: args{
				filename: "noChangeSinceLastCommit.json",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "api", "buildplanner", "model", "testdata", tt.args.filename))
			assert.NoError(t, err)

			var commit *Commit
			err = json.Unmarshal(data, &commit)
			assert.NoError(t, err)
			assert.NotNil(t, commit)

			data, err = json.MarshalIndent(commit, "", "  ")
			assert.NoError(t, err)
			t.Log(string(data))

			if err := ProcessCommitError(commit); (err != nil) != tt.wantErr {
				t.Errorf("ProcessCommitError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
