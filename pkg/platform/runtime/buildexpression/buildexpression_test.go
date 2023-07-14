package buildexpression

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				filename: "buildexpression.json",
			},
			wantErr: false,
		},
		{
			name: "complex",
			args: args{
				filename: "buildexpression-complex.json",
			},
			wantErr: false,
		},
		{
			name: "unordered",
			args: args{
				filename: "buildexpression-unordered.json",
			},
			wantErr: false,
		},
		{
			name: "installer",
			args: args{
				filename: "buildexpression-installer.json",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "runtime", "buildexpression", "testdata", tt.args.filename))
			assert.NoError(t, err)

			_, err = New(data)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestBuildExpression_Requirements(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    []model.Requirement
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				filename: "buildexpression.json",
			},
			want: []model.Requirement{
				{
					Name:      "jinja2-time",
					Namespace: "language/python",
				},
				{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				},
				{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []model.VersionRequirement{
						map[string]string{
							"comparator": string(model.ComparatorEQ),
							"version":    "3.10.10",
						},
					},
				},
				{
					Name:      "copier",
					Namespace: "language/python",
				},
				{
					Name:      "jupyterlab",
					Namespace: "language/python",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "runtime", "buildexpression", "testdata", tt.args.filename))
			assert.NoError(t, err)

			bx, err := New(data)
			assert.NoError(t, err)

			got := bx.Requirements()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildExpression_Update(t *testing.T) {
	type args struct {
		requirement model.Requirement
		operation   model.Operation
	}
	tests := []struct {
		name    string
		args    args
		want    []model.Requirement
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				requirement: model.Requirement{
					Name:      "requests",
					Namespace: "language/python",
				},
				operation: model.OperationAdded,
			},
			want: []model.Requirement{
				{
					Name:      "jinja2-time",
					Namespace: "language/python",
				},
				{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				},
				{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []model.VersionRequirement{
						map[string]string{
							"comparator": string(model.ComparatorEQ),
							"version":    "3.10.10",
						},
					},
				},
				{
					Name:      "copier",
					Namespace: "language/python",
				},
				{
					Name:      "jupyterlab",
					Namespace: "language/python",
				},
				{
					Name:      "requests",
					Namespace: "language/python",
				},
			},
			wantErr: false,
		},
		{
			name: "remove",
			args: args{
				requirement: model.Requirement{
					Name:      "jupyterlab",
					Namespace: "language/python",
				},
				operation: model.OperationRemoved,
			},
			want: []model.Requirement{
				{
					Name:      "jinja2-time",
					Namespace: "language/python",
				},
				{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				},
				{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []model.VersionRequirement{
						map[string]string{
							"comparator": string(model.ComparatorEQ),
							"version":    "3.10.10",
						},
					},
				},
				{
					Name:      "copier",
					Namespace: "language/python",
				},
			},
			wantErr: false,
		},
		{
			name: "update",
			args: args{
				requirement: model.Requirement{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []model.VersionRequirement{
						map[string]string{
							"comparator": string(model.ComparatorEQ),
							"version":    "3.11.0",
						},
					},
				},
				operation: model.OperationUpdated,
			},
			want: []model.Requirement{
				{
					Name:      "jinja2-time",
					Namespace: "language/python",
				},
				{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				},
				{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []model.VersionRequirement{
						map[string]string{
							"comparator": string(model.ComparatorEQ),
							"version":    "3.11.0",
						},
					},
				},
				{
					Name:      "copier",
					Namespace: "language/python",
				},
				{
					Name:      "jupyterlab",
					Namespace: "language/python",
				},
			},
			wantErr: false,
		},
		{
			name: "remove not existing",
			args: args{
				requirement: model.Requirement{
					Name:      "requests",
					Namespace: "language/python",
				},
				operation: model.OperationRemoved,
			},
			want: []model.Requirement{
				{
					Name:      "jinja2-time",
					Namespace: "language/python",
				},
				{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				},
				{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []model.VersionRequirement{
						map[string]string{
							"comparator": string(model.ComparatorEQ),
							"version":    "3.10.10",
						},
					},
				},
				{
					Name:      "copier",
					Namespace: "language/python",
				},
				{
					Name:      "jupyterlab",
					Namespace: "language/python",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "runtime", "buildexpression", "testdata", "buildexpression.json"))
			assert.NoError(t, err)

			bx, err := New(data)
			assert.NoError(t, err)

			err = bx.UpdateRequirement(tt.args.operation, tt.args.requirement)
			if err != nil {
				if tt.wantErr {
					return
				}

				t.Errorf("BuildExpression.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got := bx.Requirements()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", got, tt.want)
			}
		})
	}
}
