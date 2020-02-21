package initialize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/pkg/project"
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

func TestInitialize_Run(t *testing.T) {
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

	tempDirWithConfig := filepath.Join(fileutils.TempDirUnsafe(), "withConfig")
	fail := fileutils.Mkdir(tempDirWithConfig)
	if fail != nil {
		panic(fmt.Sprintf("Cannot create dir: %v", fail.ToError()))
	}
	fileutils.WriteFile(filepath.Join(tempDirWithConfig, constants.ConfigFileName), []byte(""))

	tempDirWithFile := filepath.Join(fileutils.TempDirUnsafe(), "withFile")
	fail = fileutils.Mkdir(tempDirWithConfig)
	if fail != nil {
		panic(fmt.Sprintf("Cannot create dir: %v", fail.ToError()))
	}
	fileutils.WriteFile(filepath.Join(tempDirWithFile, "bogus"), []byte(""))

	type fields struct {
		config setter
	}
	type args struct {
		namespace *project.Namespaced
		path      string
		language  language.Supported
	}
	tests := []struct {
		name     string
		wd       string
		fields   fields
		args     args
		wantErr  bool
		wantPath string
	}{
		{
			"namespace without path or language",
			tempDir,
			fields{&configMock{}},
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path: "",
			},
			false,
			tempDir,
		},
		{
			"namespace without path or language, wd has file",
			tempDirWithFile,
			fields{&configMock{}},
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path: "",
			},
			false,
			filepath.Join(tempDirWithFile, "foo/bar"),
		},
		{
			"namespace with path and without language",
			tempDir,
			fields{&configMock{}},
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path: filepath.Join(tempDir, "1"),
			},
			false,
			filepath.Join(tempDir, "1"),
		},
		{
			"namespace with path and language",
			tempDir,
			fields{&configMock{}},
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path:     filepath.Join(tempDir, "2"),
				language: language.Supported{language.Python2},
			},
			false,
			filepath.Join(tempDir, "2"),
		},
		{
			"as.yaml already exists",
			tempDir,
			fields{&configMock{}},
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path: tempDirWithConfig,
			},
			true,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.Chdir(tt.wd)
			if err != nil {
				t.Errorf("Initialize.run() chdir error = %v", err)
			}
			r := &Initialize{
				config: tt.fields.config,
			}
			path, err := run(tt.fields.config, &RunParams{
				Namespace: tt.args.namespace,
				Path:      tt.args.path,
				Language:  tt.args.language,
			})
			if tt.wantErr {
				if err == nil {
					t.Errorf("Initialize.run() error = %v, wantErr %v", err, tt.wantErr)
				}
				return // If we want an error the rest of the tests are pointless
			}

			wantPath, _ := filepath.EvalSymlinks(tt.wantPath)
			if path != wantPath {
				fmt.Println("path: ", path)
				fmt.Println("wantPath: ", wantPath)
				fmt.Println("tt.wantPath: ", tt.wantPath)
				t.Errorf("Initialize.run() path = %s, wantPath %s", path, tt.wantPath)
			}
			configFile := filepath.Join(tt.wantPath, constants.ConfigFileName)
			if !fileutils.FileExists(configFile) {
				t.Errorf("Expected file to exist: %s", configFile)
			} else {
				contents := fileutils.ReadFileUnsafe(configFile)
				if !strings.Contains(string(contents), fmt.Sprintf("%s/%s", tt.args.namespace.Owner, tt.args.namespace.Project)) {
					t.Errorf("Expected %s to contain %s/%s", contents, tt.args.namespace.Owner, tt.args.namespace.Project)
				}
			}
			if tt.args.language.Recognized() && tt.args.language.Executable().Available() && len(r.config.(*configMock).set) == 0 {
				t.Errorf("Expected config to have been written for language")
			}
		})
	}
}
