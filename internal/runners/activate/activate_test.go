package activate

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/project"
)

type checkoutMock struct {
	resultErr error
	called    bool
	cb        func()
}

func (c *checkoutMock) Run(namespace string, path string) error {
	if c.cb != nil {
		c.cb()
	}
	c.called = true
	return c.resultErr
}

type namespaceSelectMock struct {
	resultPath string
	resultErr  error
}

func (n *namespaceSelectMock) Run(namespace string, preferredPath string) (string, error) {
	if preferredPath != "" && n.resultPath == "defer" {
		return preferredPath, n.resultErr
	}
	return n.resultPath, n.resultErr
}

var activatorMock = func(out output.Outputer, subs subshell.SubShell, targetPath string, activator activateFunc) error {
	return nil
}

func TestActivate_run(t *testing.T) {
	var tempDir1 = fileutils.TempDirUnsafe()
	var tempDir2 = fileutils.TempDirUnsafe()
	var tempDir3 = fileutils.TempDirUnsafe()
	var tempDirWithConfig = fileutils.TempDirUnsafe()

	createCfg := func(path, namespace string) {
		fileutils.WriteFile(filepath.Join(path, constants.ConfigFileName), []byte("project: https://"+constants.PlatformURL+"/"+namespace))
	}
	createCfg(tempDirWithConfig, "")

	type fields struct {
		namespaceSelect  namespaceSelectAble
		activateCheckout CheckoutAble
	}
	type args struct {
		params        *ActivateParams
		activatorLoop activationLoopFunc
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		wantCheckout bool
	}{
		{
			"expect no error",
			fields{&namespaceSelectMock{"defer", nil}, &checkoutMock{cb: func() { createCfg(tempDir1, "foo/bar") }}},
			args{&ActivateParams{&project.Namespaced{"foo", "bar", nil}, tempDir1, "", false}, activatorMock},
			false,
			true,
		},
		{
			"expect no error, expect checkout",
			fields{&namespaceSelectMock{"defer", nil}, &checkoutMock{cb: func() { createCfg(tempDir2, "foo/bar") }}},
			args{&ActivateParams{&project.Namespaced{"foo", "bar", nil}, tempDir2, "", false}, activatorMock},
			false,
			true,
		},
		{
			"expect error",
			fields{&namespaceSelectMock{tempDir3, errors.New("mocked error")}, &checkoutMock{resultErr: errors.New("mocked error"), called: true}},
			args{&ActivateParams{&project.Namespaced{"foo", "bar", nil}, "", "", false}, activatorMock},
			true,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Activate{
				namespaceSelect:  tt.fields.namespaceSelect,
				activateCheckout: tt.fields.activateCheckout,
				out:              outputhelper.NewCatcher().Outputer,
			}
			if err := r.run(tt.args.params, tt.args.activatorLoop); (err != nil) != tt.wantErr {
				t.Errorf("Activate.run() error = %v, wantErr %v", errs.Join(err, ": "), tt.wantErr)
			}
			if checkoutCalled := r.activateCheckout.(*checkoutMock).called; checkoutCalled != tt.wantCheckout {
				t.Errorf("Activate.run() checkout = %v, wantCheckout %v", checkoutCalled, tt.wantCheckout)
			}
		})
	}
}

func TestActivate_setupPath(t *testing.T) {
	var tempDirWithConfig = fileutils.TempDirUnsafe()
	fileutils.WriteFile(filepath.Join(tempDirWithConfig, constants.ConfigFileName), []byte("project: https://platform.activestate.com/foo/foo"))

	type fields struct {
		namespaceSelect namespaceSelectAble
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
			"namespace with preferred path",
			fields{&namespaceSelectMock{"defer", nil}},
			args{"foo", tempDirWithConfig},
			tempDirWithConfig,
			false,
		},
		{
			"namespace no path",
			fields{&namespaceSelectMock{tempDirWithConfig, nil}},
			args{"foo", ""},
			tempDirWithConfig,
			false,
		},
		{
			"no namespace with path",
			fields{&namespaceSelectMock{}},
			args{"", tempDirWithConfig},
			tempDirWithConfig,
			false,
		},
		{
			"errors",
			fields{&namespaceSelectMock{"", errors.New("mocked error")}},
			args{"foo", ""},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Activate{
				namespaceSelect: tt.fields.namespaceSelect,
			}
			got, err := r.pathToUse(tt.args.namespace, tt.args.preferredPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Activate.pathToUse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Activate.pathToUse() = %v, want %v", got, tt.want)
			}
		})
	}
}
