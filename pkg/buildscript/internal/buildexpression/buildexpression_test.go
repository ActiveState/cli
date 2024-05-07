package buildexpression

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			name: "installer-complex",
			args: args{
				filename: "buildexpression-installer-complex.json",
			},
			wantErr: false,
		},
		{
			name: "nested",
			args: args{
				filename: "buildexpression-nested.json",
			},
			wantErr: false,
		},
		{
			name: "alternate",
			args: args{
				filename: "buildexpression-alternate.json",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "internal", "buildexpression", "testdata", tt.args.filename))
			assert.NoError(t, err)

			_, err = Unmarshal(data)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRequirements(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    []types.Requirement
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				filename: "buildexpression.json",
			},
			want: []types.Requirement{
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
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
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
		{
			name: "installer-complex",
			args: args{
				filename: "buildexpression-installer-complex.json",
			},
			want: []types.Requirement{
				{
					Name:      "perl",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "5.36.0",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "alternate",
			args: args{
				filename: "buildexpression-alternate.json",
			},
			want: []types.Requirement{
				{
					Name:      "Path-Tiny",
					Namespace: "language/perl",
				},
				{
					Name:      "perl",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "5.36.1",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "internal", "buildexpression", "testdata", tt.args.filename))
			assert.NoError(t, err)

			bx, err := Unmarshal(data)
			assert.NoError(t, err)

			got, err := bx.Requirements()
			assert.NoError(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	type args struct {
		requirement types.Requirement
		operation   types.Operation
		filename    string
	}
	tests := []struct {
		name    string
		args    args
		want    []types.Requirement
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				requirement: types.Requirement{
					Name:      "requests",
					Namespace: "language/python",
				},
				operation: types.OperationAdded,
				filename:  "buildexpression.json",
			},
			want: []types.Requirement{
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
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
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
				requirement: types.Requirement{
					Name:      "jupyterlab",
					Namespace: "language/python",
				},
				operation: types.OperationRemoved,
				filename:  "buildexpression.json",
			},
			want: []types.Requirement{
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
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
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
				requirement: types.Requirement{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "3.11.0",
						},
					},
				},
				operation: types.OperationUpdated,
				filename:  "buildexpression.json",
			},
			want: []types.Requirement{
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
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
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
				requirement: types.Requirement{
					Name:      "requests",
					Namespace: "language/python",
				},
				operation: types.OperationRemoved,
				filename:  "buildexpression.json",
			},
			want: []types.Requirement{
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
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
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
		{
			name: "add-installer-complex",
			args: args{
				requirement: types.Requirement{
					Name:      "JSON",
					Namespace: "language/perl",
				},
				operation: types.OperationAdded,
				filename:  "buildexpression-installer-complex.json",
			},
			want: []types.Requirement{
				{
					Name:      "perl",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "5.36.0",
						},
					},
				},
				{
					Name:      "JSON",
					Namespace: "language/perl",
				},
			},
			wantErr: false,
		},
		{
			name: "add-alternate",
			args: args{
				requirement: types.Requirement{
					Name:      "JSON",
					Namespace: "language/perl",
				},
				operation: types.OperationAdded,
				filename:  "buildexpression-alternate.json",
			},
			want: []types.Requirement{
				{
					Name:      "Path-Tiny",
					Namespace: "language/perl",
				},
				{
					Name:      "perl",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "5.36.1",
						},
					},
				},
				{
					Name:      "JSON",
					Namespace: "language/perl",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "internal", "buildexpression", "testdata", tt.args.filename))
			assert.NoError(t, err)

			bx, err := Unmarshal(data)
			assert.NoError(t, err)

			err = bx.UpdateRequirement(tt.args.operation, tt.args.requirement)
			if err != nil {
				if tt.wantErr {
					return
				}

				t.Errorf("BuildExpression.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := bx.Requirements()
			assert.NoError(t, err)

			sort.Slice(got, func(i, j int) bool {
				return got[i].Name < got[j].Name
			})

			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i].Name < tt.want[j].Name
			})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNullValue(t *testing.T) {
	be, err := Unmarshal([]byte(`
{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [],
        "requirements": [],
        "solver_version": null
      }
    }
  }
}
`))
	require.NoError(t, err)

	var null *string
	nullHandled := false
	for _, assignment := range be.Let.Assignments {
		if assignment.Name == "runtime" {
			args := assignment.Value.Ap.Arguments
			require.NotNil(t, args)
			for _, arg := range args {
				if arg.Assignment != nil && arg.Assignment.Name == "solver_version" {
					assert.Equal(t, null, arg.Assignment.Value.Str)
					nullHandled = true
				}
			}
		}
	}
	assert.True(t, nullHandled, "JSON null not encountered")
}

func TestCopy(t *testing.T) {
	be, err := Unmarshal([]byte(`
{
  "let": {
    "in": "$runtime",
    "runtime": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [],
        "requirements": [],
        "solver_version": null
      }
    }
  }
}
`))
	require.NoError(t, err)
	be2, err := be.Copy()
	require.NoError(t, err)
	require.Equal(t, be, be2)
}
