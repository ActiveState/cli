package initialize

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

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

	tempDirWithConfig := fileutils.Join(fileutils.TempDirUnsafe(), "withConfig")
	fail := fileutils.Mkdir(tempDirWithConfig)
	if fail != nil {
		panic(fmt.Sprintf("Cannot create dir: %v", fail.ToError()))
	}
	fileutils.WriteFile(fileutils.Join(tempDirWithConfig, constants.ConfigFileName), []byte(""))

	tempDirWithFile := fileutils.Join(fileutils.TempDirUnsafe(), "withFile")
	fail = fileutils.Mkdir(tempDirWithConfig)
	if fail != nil {
		panic(fmt.Sprintf("Cannot create dir: %v", fail.ToError()))
	}
	fileutils.WriteFile(fileutils.Join(tempDirWithFile, "bogus"), []byte(""))

	type args struct {
		namespace *project.Namespaced
		path      string
		language  string
		version   string
	}
	tests := []struct {
		name            string
		wd              string
		args            args
		wantErr         error
		wantPath        string
		resultPath      string
		wantLanguage    string
		wantLangVersion string
	}{
		{
			"namespace without path or language",
			tempDir,
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path: "",
			},
			errors.New(locale.T("err_init_no_language")),
			osutil.PrepareDir(tempDir),
			osutil.PrepareDir(tempDir),
			"",
			"",
		},
		{
			"namespace without path and with language",
			osutil.PrepareDir(fileutils.Join(tempDir, "0")),
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path:     "",
				language: language.Python2.String(),
			},
			nil,
			osutil.PrepareDir(fileutils.Join(tempDir, "0")),
			osutil.PrepareDir(fileutils.Join(tempDir, "0")),
			language.Python2.String(),
			"",
		},
		{
			"namespace without path and with language, wd has file",
			tempDirWithFile,
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path:     "",
				language: language.Python2.String(),
			},
			nil,
			tempDirWithFile,
			tempDirWithFile,
			language.Python2.String(),
			"",
		},
		{
			"namespace with path and without language",
			tempDir,
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path: fileutils.Join(tempDir, "1"),
			},
			errors.New(locale.T("err_init_no_language")),
			"",
			tempDir,
			"",
			"",
		},
		{
			"namespace with path and language",
			tempDir,
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path:     fileutils.Join(tempDir, "2"),
				language: language.Python2.String(),
			},
			nil,
			osutil.PrepareDir(fileutils.Join(tempDir, "2")),
			osutil.PrepareDir(fileutils.Join(tempDir, "2")),
			language.Python2.String(),
			"",
		},
		{
			"namespace with path, language and version",
			tempDir,
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path:     fileutils.Join(tempDir, "3"),
				language: language.Python2.String() + "@1.0",
			},
			nil,
			osutil.PrepareDir(fileutils.Join(tempDir, "3")),
			osutil.PrepareDir(fileutils.Join(tempDir, "3")),
			language.Python2.String(),
			"1.0",
		},
		{
			"as.yaml already exists",
			tempDir,
			args{
				namespace: &project.Namespaced{
					Owner:   "foo",
					Project: "bar",
				},
				path:     tempDirWithConfig,
				language: language.Python2.String(),
			},
			failures.FailUserInput.New("err_init_file_exists", tempDirWithConfig),
			"",
			tempDir,
			language.Python2.String(),
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.Chdir(tt.wd)
			if err != nil {
				t.Errorf("Initialize.run() chdir error = %v", err)
			}
			path, err := run(&RunParams{
				Namespace: tt.args.namespace,
				Path:      tt.args.path,
				Language:  tt.args.language,
			}, outputhelper.NewCatcher(), nil)
			path = osutil.PrepareDir(path)

			if tt.wantErr != nil {
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("Initialize.run() error = %v, wantErr %v", err, tt.wantErr)
				}
				return // If we want an error the rest of the tests are pointless
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if confirmPath(t, path, tt.wantPath) {
				t.Errorf("Initialize.run() path = %s, wantPath %s", path, tt.wantPath)
			}
			configFile := fileutils.Join(tt.wantPath, constants.ConfigFileName)
			if !fileutils.FileExists(configFile) {
				t.Fatalf("Expected file to exist: %s", configFile)
			}

			pj, fail := projectfile.Parse(configFile)
			if fail != nil {
				t.Fatalf("Projectfile failed to parse: %s", fail.Error())
			}
			if !strings.Contains(pj.Project, fmt.Sprintf("%s/%s", tt.args.namespace.Owner, tt.args.namespace.Project)) {
				t.Errorf("Expected %s to contain %s/%s", pj.Project, tt.args.namespace.Owner, tt.args.namespace.Project)
			}

			if len(pj.Languages) != 1 {
				t.Fatalf("Expected 1 languages but got %d: %v", len(pj.Languages), pj.Languages)
			}
			if pj.Languages[0].Name != tt.wantLanguage {
				t.Errorf("Expected language %s, actual: %s", pj.Languages[0].Name, tt.wantLanguage)
			}
			if pj.Languages[0].Version != tt.wantLangVersion {
				t.Errorf("Expected language version %s, actual: %s", pj.Languages[0].Version, tt.wantLangVersion)
			}
		})
	}
}

func confirmPath(t *testing.T, path, want string) bool {
	if runtime.GOOS == "windows" {
		longPath, err := fileutils.GetLongPathName(path)
		if err != nil {
			t.Fatal(err)
		}

		longWant, err := fileutils.GetLongPathName(want)
		if err != nil {
			t.Fatal(err)
		}
		return longPath != longWant
	}
	wantEval, _ := filepath.EvalSymlinks(want)
	return path != wantEval
}
