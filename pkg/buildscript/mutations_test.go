package buildscript

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

// TestUpdateRequirements tests that build scripts can correctly read requirements from build
// expressions, modify them (add/update/remove), and return them in a structured format external to
// the internal, raw format.
func TestUpdateRequirements(t *testing.T) {
	type args struct {
		requirement DependencyRequirement
		operation   types.Operation
		filename    string
	}
	tests := []struct {
		name    string
		args    args
		want    []DependencyRequirement
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				requirement: DependencyRequirement{types.Requirement{
					Name:      "requests",
					Namespace: "language/python",
				}},
				operation: types.OperationAdded,
				filename:  "buildexpression.json",
			},
			want: []DependencyRequirement{
				{types.Requirement{
					Name:      "jinja2-time",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "3.10.10",
						},
					},
				}},
				{types.Requirement{
					Name:      "copier",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "jupyterlab",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "requests",
					Namespace: "language/python",
				}},
			},
			wantErr: false,
		},
		{
			name: "remove",
			args: args{
				requirement: DependencyRequirement{types.Requirement{
					Name:      "jupyterlab",
					Namespace: "language/python",
				}},
				operation: types.OperationRemoved,
				filename:  "buildexpression.json",
			},
			want: []DependencyRequirement{
				{types.Requirement{
					Name:      "jinja2-time",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "3.10.10",
						},
					},
				}},
				{types.Requirement{
					Name:      "copier",
					Namespace: "language/python",
				}},
			},
			wantErr: false,
		},
		{
			name: "update",
			args: args{
				requirement: DependencyRequirement{types.Requirement{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "3.11.0",
						},
					},
				}},
				operation: types.OperationUpdated,
				filename:  "buildexpression.json",
			},
			want: []DependencyRequirement{
				{types.Requirement{
					Name:      "jinja2-time",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "3.11.0",
						},
					},
				}},
				{types.Requirement{
					Name:      "copier",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "jupyterlab",
					Namespace: "language/python",
				}},
			},
			wantErr: false,
		},
		{
			name: "remove not existing",
			args: args{
				requirement: DependencyRequirement{types.Requirement{
					Name:      "requests",
					Namespace: "language/python",
				}},
				operation: types.OperationRemoved,
				filename:  "buildexpression.json",
			},
			want: []DependencyRequirement{
				{types.Requirement{
					Name:      "jinja2-time",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "jupyter-contrib-nbextensions",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "python",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "3.10.10",
						},
					},
				}},
				{types.Requirement{
					Name:      "copier",
					Namespace: "language/python",
				}},
				{types.Requirement{
					Name:      "jupyterlab",
					Namespace: "language/python",
				}},
			},
			wantErr: true,
		},
		{
			name: "add-installer-complex",
			args: args{
				requirement: DependencyRequirement{types.Requirement{
					Name:      "JSON",
					Namespace: "language/perl",
				}},
				operation: types.OperationAdded,
				filename:  "buildexpression-installer-complex.json",
			},
			want: []DependencyRequirement{
				{types.Requirement{
					Name:      "perl",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "5.36.0",
						},
					},
				}},
				{types.Requirement{
					Name:      "JSON",
					Namespace: "language/perl",
				}},
			},
			wantErr: false,
		},
		{
			name: "add-alternate",
			args: args{
				requirement: DependencyRequirement{types.Requirement{
					Name:      "JSON",
					Namespace: "language/perl",
				}},
				operation: types.OperationAdded,
				filename:  "buildexpression-alternate.json",
			},
			want: []DependencyRequirement{
				{types.Requirement{
					Name:      "Path-Tiny",
					Namespace: "language/perl",
				}},
				{types.Requirement{
					Name:      "perl",
					Namespace: "language",
					VersionRequirement: []types.VersionRequirement{
						map[string]string{
							"comparator": string(types.ComparatorEQ),
							"version":    "5.36.1",
						},
					},
				}},
				{types.Requirement{
					Name:      "JSON",
					Namespace: "language/perl",
				}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "testdata", tt.args.filename))
			assert.NoError(t, err)

			script, err := UnmarshalBuildExpression(data, "", nil)
			assert.NoError(t, err)

			err = script.UpdateRequirement(tt.args.operation, tt.args.requirement.Requirement)
			if err != nil {
				if tt.wantErr {
					return
				}

				t.Errorf("BuildExpression.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := script.Requirements()
			assert.NoError(t, err)

			gotReqs := []DependencyRequirement{}
			for _, g := range got {
				gotReqs = append(gotReqs, g.(DependencyRequirement))
			}

			sort.Slice(gotReqs, func(i, j int) bool { return gotReqs[i].Name < gotReqs[j].Name })
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i].Name < tt.want[j].Name })

			if !reflect.DeepEqual(gotReqs, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", gotReqs, tt.want)
			}
		})
	}
}

// TestUpdatePlatform tests that build scripts can correctly read platforms from build
// expressions, modify them (add/remove), and return them in a structured format external to the
// internal, raw format.
func TestUpdatePlatform(t *testing.T) {
	type args struct {
		platform  strfmt.UUID
		operation types.Operation
		filename  string
	}
	tests := []struct {
		name    string
		args    args
		want    []strfmt.UUID
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				platform:  strfmt.UUID("78977bc8-0f32-519d-80f3-9043f059398c"),
				operation: types.OperationAdded,
				filename:  "buildexpression.json",
			},
			want: []strfmt.UUID{
				strfmt.UUID("78977bc8-0f32-519d-80f3-9043f059398c"),
				strfmt.UUID("96b7e6f2-bebf-564c-bc1c-f04482398f38"),
			},
			wantErr: false,
		},
		{
			name: "remove",
			args: args{
				platform:  strfmt.UUID("0fa42e8c-ac7b-5dd7-9407-8aa15f9b993a"),
				operation: types.OperationRemoved,
				filename:  "buildexpression-alternate.json",
			},
			want: []strfmt.UUID{
				strfmt.UUID("46a5b48f-226a-4696-9746-ba4d50d661c2"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wd, err := environment.GetRootPath()
			assert.NoError(t, err)

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "testdata", tt.args.filename))
			assert.NoError(t, err)

			script, err := UnmarshalBuildExpression(data, "", nil)
			assert.NoError(t, err)

			if tt.args.operation == types.OperationAdded {
				err = script.AddPlatform(tt.args.platform)
			} else {
				err = script.RemovePlatform(tt.args.platform)
			}
			if err != nil {
				if tt.wantErr {
					return
				}

				t.Errorf("BuildExpression.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := script.Platforms()
			assert.NoError(t, err)

			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			sort.Slice(tt.want, func(i, j int) bool { return tt.want[i] < tt.want[j] })

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Platforms() = %v, want %v", got, tt.want)
			}
		})
	}
}
