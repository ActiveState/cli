package deploy

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	rt "runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type InstallableMock struct{}

func (i *InstallableMock) Install() (envGetter envGetter, freshInstallation bool, fail *failures.Failure) {
	return nil, false, nil
}

func (i *InstallableMock) Env() (envGetter envGetter, fail *failures.Failure) {
	return nil, nil
}

type EnvGetMock struct {
	callback func(inherit bool, projectDir string) (map[string]string, *failures.Failure)
}

func (e *EnvGetMock) GetEnv(inherit bool, projectDir string) (map[string]string, *failures.Failure) {
	return e.callback(inherit, projectDir)
}

type OutputterMock struct{}

func (o *OutputterMock) Print(value interface{})  {}
func (o *OutputterMock) Error(value interface{})  {}
func (o *OutputterMock) Notice(value interface{}) {}
func (o *OutputterMock) Config() *output.Config {
	return nil
}

func Test_runStepsWithFuncs(t *testing.T) {
	runSymlinks := runSymlinkTests()
	type args struct {
		installer installable
		step      Step
	}
	type want struct {
		err           error
		installCalled bool
		configCalled  bool
		symlinkCalled bool
		reportCalled  bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"Deploy without steps",
			args{
				&InstallableMock{},
				UnsetStep,
			},
			want{
				nil,
				true,
				true,
				runSymlinks,
				true,
			},
		},
		{
			"Deploy with install step",
			args{
				&InstallableMock{},
				InstallStep,
			},
			want{
				nil,
				true,
				false,
				false,
				false,
			},
		},
		{
			"Deploy with config step",
			args{
				&InstallableMock{},
				ConfigureStep,
			},
			want{
				nil,
				false,
				true,
				false,
				false,
			},
		},
		{
			"Deploy with symlink step",
			args{
				&InstallableMock{},
				SymlinkStep,
			},
			want{
				nil,
				false,
				false,
				runSymlinks,
				false,
			},
		},
		{
			"Deploy with report step",
			args{
				&InstallableMock{},
				ReportStep,
			},
			want{
				nil,
				false,
				false,
				false,
				true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var installCalled bool
			installFunc := func(installable, output.Outputer) (runtime.EnvGetter, error) {
				installCalled = true
				return nil, nil
			}
			var configCalled bool
			configFunc := func(runtime.EnvGetter, output.Outputer) error {
				configCalled = true
				return nil
			}
			var symlinkCalled bool
			symlinkFunc := func(string, bool, runtime.EnvGetter, output.Outputer) error {
				symlinkCalled = true
				return nil
			}
			var reportCalled bool
			reportFunc := func(runtime.EnvGetter, output.Outputer) error {
				reportCalled = true
				return nil
			}
			catcher := outputhelper.NewCatcher()
			err := runStepsWithFuncs("", true, tt.args.step, tt.args.installer, catcher.Outputer, installFunc, configFunc, symlinkFunc, reportFunc)
			if err != tt.want.err {
				t.Errorf("runStepsWithFuncs() error = %v, wantErr %v", err, tt.want.err)
			}
			if installCalled != tt.want.installCalled {
				t.Errorf("runStepsWithFuncs() installCalled = %v, want %v", installCalled, tt.want.installCalled)
			}
			if configCalled != tt.want.configCalled {
				t.Errorf("runStepsWithFuncs() configCalled = %v, want %v", configCalled, tt.want.configCalled)
			}
			if symlinkCalled != tt.want.symlinkCalled {
				t.Errorf("runStepsWithFuncs() symlinkCalled = %v, want %v", symlinkCalled, tt.want.symlinkCalled)
			}
			if reportCalled != tt.want.reportCalled {
				t.Errorf("runStepsWithFuncs() reportCalled = %v, want %v", reportCalled, tt.want.reportCalled)
			}
		})
	}
}

func Test_report(t *testing.T) {
	type args struct {
		envGetter runtime.EnvGetter
	}
	tests := []struct {
		name       string
		args       args
		wantBinary []string
		wantEnv    map[string]string
		wantErr    error
	}{
		{
			"Report",
			args{
				&EnvGetMock{
					func(inherit bool, projectDir string) (map[string]string, *failures.Failure) {
						return map[string]string{
							"KEY1": "VAL1",
							"KEY2": "VAL2",
							"PATH": "PATH1" + string(os.PathListSeparator) + "PATH2",
						}, nil
					},
				},
			},
			[]string{"PATH1", "PATH2"},
			map[string]string{
				"KEY1": "VAL1",
				"KEY2": "VAL2",
			},
			nil,
		},
	}
	for _, tt := range tests {
		catcher := outputhelper.TypedCatcher{}
		t.Run(tt.name, func(t *testing.T) {
			if err := report(tt.args.envGetter, &catcher); err != tt.wantErr {
				t.Errorf("report() error = %v, wantErr %v", err, tt.wantErr)
				t.FailNow()
			}
			report, ok := catcher.Prints[0].(Report)
			if !ok {
				t.Errorf("Printed unknown structure, expected Report type. Value: %v", report)
				t.FailNow()
			}

			if !reflect.DeepEqual(report.Environment, tt.wantEnv) {
				t.Errorf("Expected envs to be the same. Want: %v, got: %v", tt.wantEnv, report.Environment)
			}

			if !reflect.DeepEqual(report.BinaryDirectories, tt.wantBinary) {
				t.Errorf("Expected bins to be the same. Want: %v, got: %v", tt.wantBinary, report.BinaryDirectories)
			}
		})
	}
}

func Test_symlinkWithTarget(t *testing.T) {
	if !runSymlinkTests() {
		t.Skip("Windows developer mode is not active")
	}

	root, err := environment.GetRootPath()
	if err != nil {
		t.Error(err)
	}

	testDataDir := filepath.Join(root, "internal", "runners", "deploy", "testdata")
	testFile := filepath.Join(testDataDir, "main.go")
	binaryName := "test-bin"
	if rt.GOOS == "windows" {
		binaryName += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join(testDataDir, binaryName), testFile)
	err = cmd.Run()
	if err != nil {
		t.Error(err)
	}

	installDir, err := ioutil.TempDir("", "install-dir")
	if err != nil {
		t.Error(err)
	}

	targetDir, err := ioutil.TempDir("", "target-dir")
	if err != nil {
		t.Error(err)
	}

	fail := fileutils.CopyFile(filepath.Join(testDataDir, binaryName), filepath.Join(installDir, binaryName))
	if fail != nil {
		t.Error(fail.ToError())
	}

	if rt.GOOS != "windows" {
		cmd = exec.Command("chmod", "+x", filepath.Join(installDir, binaryName))
		err = cmd.Run()
		if err != nil {
			t.Error(err)
		}
	}

	type args struct {
		overwrite bool
		path      string
		bins      []string
		out       output.Outputer
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Symlink binary file",
			args: args{
				overwrite: false,
				path:      targetDir,
				bins:      []string{installDir},
				out:       &OutputterMock{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := symlinkWithTarget(tt.args.overwrite, tt.args.path, tt.args.bins, tt.args.out); (err != nil) != tt.wantErr {
				t.Errorf("symlinkWithTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
			defer func() {
				os.RemoveAll(installDir)
				os.RemoveAll(targetDir)
				os.Remove(filepath.Join(testDataDir, binaryName))
			}()

			cmd := exec.Command(filepath.Join(targetDir, binaryName))
			out, err := cmd.Output()
			if err != nil {
				t.Error(err)
			}
			if strings.TrimSpace(string(out)) != "Hello World!" {
				t.Fatalf("Unexpected output: %s", string(out))
			}
		})
	}
}
