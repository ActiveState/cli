package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
)

type configMock struct {
	set map[string]interface{}
}

func (c *configMock) Set(key string, value interface{}) {
	if c.set == nil {
		c.set = map[string]interface{}{}
	}
	c.set[key] = value
}

func TestInit_Run(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Cannot get wd: %v", err))
	}
	defer os.Chdir(wd)

	var tempDir = fileutils.TempDirUnsafe()
	os.Chdir(tempDir)

	// Set tempDir according to Getwd() as the fully resolved path tends to look different on macOS
	tempDir, err = os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Cannot get wd: %v", err))
	}

	tempDirWithConfig := filepath.Join(tempDir, "withConfig")
	fail := fileutils.Mkdir(tempDirWithConfig)
	if fail != nil {
		panic(fmt.Sprintf("Cannot create dir: %v", fail.ToError()))
	}
	fileutils.WriteFile(filepath.Join(tempDirWithConfig, constants.ConfigFileName), []byte(""))

	type fields struct {
		config configAble
	}
	type args struct {
		namespace string
		path      string
		language  string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		wantPath string
	}{
		{
			"namespace without path or language",
			fields{&configMock{}},
			args{"foo/bar", "", ""},
			false,
			filepath.Join(tempDir, "foo/bar"),
		},
		{
			"namespace with path and without language",
			fields{&configMock{}},
			args{"foo/bar", filepath.Join(tempDir, "1"), ""},
			false,
			filepath.Join(tempDir, "1"),
		},
		{
			"namespace with path and language",
			fields{&configMock{}},
			args{"foo/bar", filepath.Join(tempDir, "2"), "python2"},
			false,
			filepath.Join(tempDir, "2"),
		},
		{
			"as.yaml already exists",
			fields{&configMock{}},
			args{"foo/bar", tempDirWithConfig, ""},
			true,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Init{
				config: tt.fields.config,
			}
			path, err := r.run(tt.args.namespace, tt.args.path, tt.args.language)
			if (err != nil) != tt.wantErr {
				t.Errorf("Init.run() error = %v, wantErr %v", err, tt.wantErr)
			}
			// If we want an error the rest of the tests are pointless
			if tt.wantErr == true {
				return
			}
			if path != tt.wantPath {
				t.Errorf("Init.run() path = %s, wantPath %s", path, tt.wantPath)
			}
			configFile := filepath.Join(tt.wantPath, constants.ConfigFileName)
			if !fileutils.FileExists(configFile) {
				t.Errorf("Expected file to exist: %s", configFile)
			} else {
				contents := fileutils.ReadFileUnsafe(configFile)
				if !strings.Contains(string(contents), tt.args.namespace) {
					t.Errorf("Expected %s to contain %s", contents, tt.args.namespace)
				}
			}
			if tt.args.language != "" && len(r.config.(*configMock).set) == 0 {
				t.Errorf("Expected config to have been written for language")
			}
		})
	}
}
