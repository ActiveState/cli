package buildscript

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

var atTime = "2000-01-01T00:00:00.000Z"

var basicBuildScript = []byte(fmt.Sprintf(
	`at_time = "%s"
runtime = state_tool_artifacts(
	src = sources
)
sources = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "python", namespace = "language", version = Eq(value = "3.10.10"))
	],
	solver_version = null
)

main = runtime`, atTime))

var basicBuildExpression = []byte(`{
  "let": {
    "in": "$runtime",
    "runtime": {
      "state_tool_artifacts": {
        "src": "$sources"
      }
    },
    "sources": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "python",
            "namespace": "language",
            "version_requirements": [
              {
                "comparator": "eq",
                "version": "3.10.10"
              }
            ]
          }
        ],
        "solver_version": null
      }
    }
  }
}`)

// TestRoundTripFromBuildScript tests that if we read a build script from disk and then write it
// again it produces the exact same value.
func TestRoundTripFromBuildScript(t *testing.T) {
	script, err := Unmarshal(basicBuildScript)
	require.NoError(t, err)

	data, err := script.Marshal()
	require.NoError(t, err)
	t.Logf("marshalled:\n%s\n---", string(data))

	roundTripScript, err := Unmarshal(data)
	require.NoError(t, err)

	assert.Equal(t, script, roundTripScript)
	equal, err := script.Equals(roundTripScript)
	require.NoError(t, err)
	assert.True(t, equal)
}

// TestRoundTripFromBuildExpression tests that if we construct a buildscript from a Platform build
// expression and then immediately construct another build expression from that build script, the
// build expressions are identical.
func TestRoundTripFromBuildExpression(t *testing.T) {
	script, err := UnmarshalBuildExpression(basicBuildExpression, nil)
	require.NoError(t, err)

	data, err := script.MarshalBuildExpression()
	require.NoError(t, err)

	require.Equal(t, string(basicBuildExpression), string(data))
}

// TestExpressionToScript tests that creating a build script from a given Platform build expression
// and at time produces the expected result.
func TestExpressionToScript(t *testing.T) {
	ts, err := time.Parse(strfmt.RFC3339Millis, atTime)
	require.NoError(t, err)

	script, err := UnmarshalBuildExpression(basicBuildExpression, &ts)
	require.NoError(t, err)

	data, err := script.Marshal()
	require.NoError(t, err)

	require.Equal(t, string(basicBuildScript), string(data))
}

// TestScriptToExpression tests that we can produce a valid Platform build expression from a build
// script on disk.
func TestScriptToExpression(t *testing.T) {
	bs, err := Unmarshal(basicBuildScript)
	require.NoError(t, err)

	data, err := bs.MarshalBuildExpression()
	require.NoError(t, err)

	require.Equal(t, string(basicBuildExpression), string(data))
}

// TestUnmarshalBuildExpression tests that we can successfully read and convert Platform
// build expressions into build scripts.
func TestUnmarshalBuildExpression(t *testing.T) {
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
		{
			name: "newObjects",
			args: args{
				filename: "buildexpression-new-objects.json",
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

			_, err = UnmarshalBuildExpression(data, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// TestRequirements tests that build scripts can correctly read requirements from build expressions
// and return them in a structured format external to the internal, raw format.
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

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "testdata", tt.args.filename))
			assert.NoError(t, err)

			script, err := UnmarshalBuildExpression(data, nil)
			assert.NoError(t, err)

			got, err := script.Requirements()
			assert.NoError(t, err)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUpdateRequirements tests that build scripts can correctly read requirements from build
// expressions, modify them (add/update/remove), and return them in a structured format external to
// the internal, raw format.
func TestUpdateRequirements(t *testing.T) {
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

			data, err := fileutils.ReadFile(filepath.Join(wd, "pkg", "buildscript", "testdata", tt.args.filename))
			assert.NoError(t, err)

			script, err := UnmarshalBuildExpression(data, nil)
			assert.NoError(t, err)

			err = script.UpdateRequirement(tt.args.operation, tt.args.requirement)
			if err != nil {
				if tt.wantErr {
					return
				}

				t.Errorf("BuildExpression.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := script.Requirements()
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

			script, err := UnmarshalBuildExpression(data, nil)
			assert.NoError(t, err)

			err = script.UpdatePlatform(tt.args.operation, tt.args.platform)
			if err != nil {
				if tt.wantErr {
					return
				}

				t.Errorf("BuildExpression.Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got, err := script.Platforms()
			assert.NoError(t, err)

			sort.Slice(got, func(i, j int) bool {
				return got[i] < got[j]
			})

			sort.Slice(tt.want, func(i, j int) bool {
				return tt.want[i] < tt.want[j]
			})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildExpression.Platforms() = %v, want %v", got, tt.want)
			}
		})
	}
}
