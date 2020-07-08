package deploy

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

type InstallableMock struct{}

func (i *InstallableMock) Install() (envGetter envGetter, freshInstallation bool, fail *failures.Failure) {
	return nil, false, nil
}

func (i *InstallableMock) Env() (envGetter envGetter, fail *failures.Failure) {
	return nil, nil
}

func (i *InstallableMock) IsInstalled() (bool, *failures.Failure) {
	return true, nil
}

type EnvGetMock struct {
	callback func(inherit bool, projectDir string) (map[string]string, error)
}

func (e *EnvGetMock) GetEnv(inherit bool, projectDir string) (map[string]string, error) {
	return e.callback(inherit, projectDir)
}

func Test_runStepsWithFuncs(t *testing.T) {
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
				true,
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
				true,
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
			installFunc := func(string, installable, output.Outputer) (runtime.EnvGetter, error) {
				installCalled = true
				return nil, nil
			}
			var configCalled bool
			configFunc := func(string, runtime.EnvGetter, output.Outputer, project.Namespaced, bool) error {
				configCalled = true
				return nil
			}
			var symlinkCalled bool
			symlinkFunc := func(string, bool, runtime.EnvGetter, output.Outputer) error {
				symlinkCalled = true
				return nil
			}
			var reportCalled bool
			reportFunc := func(string, runtime.EnvGetter, output.Outputer) error {
				reportCalled = true
				return nil
			}
			catcher := outputhelper.NewCatcher()
			forceOverwrite := true
			userScope := false
			namespace := project.Namespaced{"owner", "project"}
			err := runStepsWithFuncs("", forceOverwrite, userScope, namespace, tt.args.step, tt.args.installer, catcher.Outputer, installFunc, configFunc, symlinkFunc, reportFunc)
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
		name        string
		installPath string
		args        args
		wantBinary  []string
		wantEnv     map[string]string
		wantErr     error
	}{
		{
			"Report",
			filepath.Join("some", "path"),
			args{
				&EnvGetMock{
					func(inherit bool, projectDir string) (map[string]string, error) {
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
			if err := report(tt.installPath, tt.args.envGetter, &catcher); err != tt.wantErr {
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

func Test_uniqueExes(t *testing.T) {
	tests := []struct {
		name    string
		bins    []string
		pathext string
		want    []string
	}{
		{
			"Returns same bins",
			[]string{"path1/a", "path2/b", "path3/c"},
			"",
			[]string{"path1/a", "path2/b", "path3/c"},
		},
		{
			"Returns exe prioritized",
			[]string{"path1/a.exe", "path2/a.cmd", "path3/c"},
			".exe;.cmd",
			[]string{"path1/a.exe", "path3/c"},
		},
		{
			"Returns cmd prioritized by PATH",
			[]string{"path1/a.exe", "path2/a.cmd", "path2/c"},
			".cmd;.exe",
			[]string{"path1/a.exe", "path2/c"},
		},
		{
			"Returns cmd prioritized by PATHEXT",
			[]string{"path1/a.exe", "path1/a.cmd", "path2/c"},
			".cmd;.exe",
			[]string{"path1/a.cmd", "path2/c"},
		},
		{
			"PATHEXT can be empty",
			[]string{"path1/a", "path2/b", "path3/c"},
			"",
			[]string{"path1/a", "path2/b", "path3/c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := uniqueExes(tt.bins, tt.pathext)
			if err != nil {
				t.Errorf("uniqueExes error: %v", err)
				t.FailNow()
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("uniqueExes() = %v, want %v", got, tt.want)
			}
		})
	}
}
