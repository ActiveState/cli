package deploy

import (
	"testing"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

type InstallableMock struct{}

func (i *InstallableMock) Install() (envGetter envGetter, freshInstallation bool, err error) {
	return nil, false, nil
}

func (i *InstallableMock) Env() (envGetter envGetter, err error) {
	return nil, nil
}

func (i *InstallableMock) IsInstalled() (bool, error) {
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
		runtime   *runtime.Runtime
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
				nil,
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
				nil,
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
				nil,
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
				nil,
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
				nil,
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
			installFunc := func(string, installable, output.Outputer) error {
				installCalled = true
				return nil
			}
			var configCalled bool
			configFunc := func(string, *runtime.Runtime, output.Outputer, subshell.SubShell, project.Namespaced, bool) error {
				configCalled = true
				return nil
			}
			var symlinkCalled bool
			symlinkFunc := func(string, bool, *runtime.Runtime, output.Outputer) error {
				symlinkCalled = true
				return nil
			}
			var reportCalled bool
			reportFunc := func(string, *runtime.Runtime, output.Outputer) error {
				reportCalled = true
				return nil
			}
			catcher := outputhelper.NewCatcher()
			forceOverwrite := true
			userScope := false
			namespace := project.Namespaced{"owner", "project", nil}
			err := runStepsWithFuncs("", forceOverwrite, userScope, namespace, tt.args.step, tt.args.runtime, tt.args.installer, catcher.Outputer, nil, installFunc, configFunc, symlinkFunc, reportFunc)
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

