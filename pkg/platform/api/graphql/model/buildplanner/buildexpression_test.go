package model

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "api", "graphql", "model", "buildplanner", "testdata", tt.args.filename))
			assert.NoError(t, err)

			_, err = NewBuildExpression(data)
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
		want    []Requirement
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				filename: "buildexpression.json",
			},
			want: []Requirement{
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
					VersionRequirement: []VersionRequirement{
						map[Comparator]string{
							"comparator": string(ComparatorEQ),
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

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "api", "graphql", "model", "buildplanner", "testdata", tt.args.filename))
			assert.NoError(t, err)

			bx, err := NewBuildExpression(data)
			assert.NoError(t, err)

			got, err := bx.Requirements()
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildExpression.Requirements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildExpression_UpdateRequirements(t *testing.T) {
	type args struct {
		filename     string
		requirements []Requirement
	}
	tests := []struct {
		name    string
		args    args
		want    []Requirement
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				filename: "buildexpression.json",
				requirements: []Requirement{
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
						VersionRequirement: []VersionRequirement{
							map[Comparator]string{
								"comparator": string(ComparatorEQ),
								"version":    "3.10.10",
							},
						},
					},
					// Removed copier requirement
					{
						Name:      "jupyterlab",
						Namespace: "language/python",
					},
				},
			},
			want: []Requirement{
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
					VersionRequirement: []VersionRequirement{
						map[Comparator]string{
							"comparator": string(ComparatorEQ),
							"version":    "3.10.10",
						},
					},
				},
				// Removed copier requirement
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

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "platform", "api", "graphql", "model", "buildplanner", "testdata", tt.args.filename))
			assert.NoError(t, err)

			bx, err := NewBuildExpression(data)
			assert.NoError(t, err)

			if err := bx.UpdateRequirements(tt.args.requirements); (err != nil) != tt.wantErr {
				t.Errorf("BuildExpression.UpdateRequirements() error = %v, wantErr %v", err, tt.wantErr)
			}

			got, err := bx.Requirements()
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildExpression.Requirements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", got, tt.want)
			}
		})
	}
}
