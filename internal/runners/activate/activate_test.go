package activate

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
)

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

var activatorMock = func(string, activateFunc) error {
	return nil
}

func TestActivate_run(t *testing.T) {
	type fields struct {
		namespaceSelect namespaceSelectAble
	}
	type args struct {
		namespace     string
		preferredPath string
		activatorLoop activationLoopFunc
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"expect no error",
			fields{&namespaceSelectMock{"defer", nil}},
			args{"foo", fileutils.TempDirUnsafe(), activatorMock},
			false,
		},
		{
			"expect error",
			fields{&namespaceSelectMock{fileutils.TempDirUnsafe(), errors.New("mocked error")}},
			args{"foo", "", activatorMock},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Activate{
				namespaceSelect: tt.fields.namespaceSelect,
			}
			if err := r.run(tt.args.namespace, tt.args.preferredPath, tt.args.activatorLoop); (err != nil) != tt.wantErr {
				t.Errorf("Activate.run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestActivate_setupPath(t *testing.T) {
	var tempDir = fileutils.TempDirUnsafe()

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
			args{"foo", filepath.Join(tempDir, "1")},
			filepath.Join(tempDir, "1"),
			false,
		},
		{
			"namespace no path",
			fields{&namespaceSelectMock{filepath.Join(tempDir, "2"), nil}},
			args{"foo", ""},
			filepath.Join(tempDir, "2"),
			false,
		},
		{
			"no namespace with path",
			fields{&namespaceSelectMock{}},
			args{"", filepath.Join(tempDir, "3")},
			filepath.Join(tempDir, "3"),
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
			got, err := r.setupPath(tt.args.namespace, tt.args.preferredPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Activate.setupPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Activate.setupPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
