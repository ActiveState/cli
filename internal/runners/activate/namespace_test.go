package activate

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"

	"github.com/ActiveState/cli/internal/prompt"
)

type configMock struct {
	result []string
	set    map[string]interface{}
}

func (c *configMock) Set(key string, value interface{}) {
	if c.set == nil {
		c.set = map[string]interface{}{}
	}
	c.set[key] = value
}

func (c *configMock) GetStringSlice(key string) []string {
	return c.result
}

type promptMock struct {
	targetPathResponse   string
	targetPathFailure    *failures.Failure
	existingPathResponse string
	existingPathFailure  *failures.Failure
	targetPrompted       bool
	existingPrompted     bool
}

func (p *promptMock) Input(message, defaultResponse string, flags ...prompt.ValidatorFlag) (string, *failures.Failure) {
	p.targetPrompted = true
	return p.targetPathResponse, p.targetPathFailure
}

func (p *promptMock) Select(message string, choices []string, defaultResponse string) (string, *failures.Failure) {
	p.existingPrompted = true
	return p.existingPathResponse, p.existingPathFailure
}

func TestNamespaceSelect_Run(t *testing.T) {
	var tempDir = fileutils.TempDirUnsafe()
	var tempDirWithConfig = fileutils.TempDirUnsafe()
	fileutils.WriteFile(filepath.Join(tempDirWithConfig, constants.ConfigFileName), []byte("project: https://platform.activestate.com/foo/bar"))

	type fields struct {
		config   configAble
		prompter promptAble
	}
	type args struct {
		namespace     string
		preferredPath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			"namespace with path, empty config",
			fields{
				&configMock{},
				&promptMock{},
			},
			args{"foo/bar", tempDir},
			tempDir,
			false,
		},
		{
			"namespace without path, empty config",
			fields{
				&configMock{},
				&promptMock{tempDir, nil, "", nil, false, false},
			},
			args{"foo/bar", ""},
			tempDir,
			false,
		},
		{
			"namespace without path, existing config",
			fields{
				&configMock{result: []string{tempDirWithConfig}},
				&promptMock{"", nil, tempDirWithConfig, nil, false, false},
			},
			args{"foo/bar", ""},
			tempDirWithConfig,
			false,
		},
		{
			"namespace with path, existing config",
			fields{
				&configMock{result: []string{filepath.Join(tempDirWithConfig, "dont-pick-me")}},
				&promptMock{"", nil, tempDirWithConfig, nil, false, false},
			},
			args{"foo/bar", tempDirWithConfig},
			tempDirWithConfig,
			false,
		},
		{
			"namespace without path, prompt error",
			fields{
				&configMock{},
				&promptMock{"", failures.FailDeveloper.New("Expected error"), "", nil, false, false},
			},
			args{"foo/bar", ""},
			tempDirWithConfig,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &NamespaceSelect{
				config:   tt.fields.config,
				prompter: tt.fields.prompter,
			}
			got, err := r.Run(tt.args.namespace, tt.args.preferredPath)

			// Validate error
			if (err != nil) != tt.wantErr {
				t.Errorf("NamespaceSelect.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // no point validating the rest
			}

			// Validate function call result
			if got != tt.want {
				t.Errorf("NamespaceSelect.Run() = %v, want %v", got, tt.want)
			}

			// Validate config saved
			cfg := r.config.(*configMock)
			if _, ok := cfg.set["project_"+tt.args.namespace]; !ok {
				t.Errorf("Namespace should have been saved to config")
			}
			entries := cfg.set["project_"+tt.args.namespace].([]string)
			if entries[0] != tt.want {
				t.Errorf("Path selection should have been saved to config")
			}

			// Validate prompts called
			var prompter = r.prompter.(*promptMock)
			if tt.args.preferredPath == "" && prompter.targetPathResponse != "" && !prompter.targetPrompted {
				t.Error("Expected to be prompted for target dir, but it didn't happen")
			}
			if tt.args.preferredPath == "" && prompter.existingPathResponse != "" && !prompter.existingPrompted {
				t.Error("Expected to be prompted for existing dir, but it didn't happen")
			}
		})
	}
}
