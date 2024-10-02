package buildscript

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

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

			script, err := UnmarshalBuildExpression(data, "", nil)
			assert.NoError(t, err)

			got, err := script.Requirements()
			assert.NoError(t, err)

			gotReqs := []types.Requirement{}
			for _, g := range got {
				gotReqs = append(gotReqs, g.(DependencyRequirement).Requirement)
			}

			if !reflect.DeepEqual(gotReqs, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", gotReqs, tt.want)
			}
		})
	}
}

const ValidZeroUUID = "00000000-0000-0000-0000-000000000000"

// TestRevision tests that build scripts can correctly read revisions from build expressions
// and return them in a structured format external to the internal, raw format.
func TestRevision(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    []RevisionRequirement
		wantErr bool
	}{
		{
			name: "basic",
			args: args{
				filename: "buildexpression_rev.json",
			},
			want: []RevisionRequirement{
				{
					Name:       "revision-pkg",
					RevisionID: ValidZeroUUID,
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

			script, err := UnmarshalBuildExpression(data, "", nil)
			assert.NoError(t, err)

			got, err := script.Requirements()
			assert.NoError(t, err)

			gotReqs := []RevisionRequirement{}
			for _, g := range got {
				gotReqs = append(gotReqs, g.(RevisionRequirement))
			}

			if !reflect.DeepEqual(gotReqs, tt.want) {
				t.Errorf("BuildExpression.Requirements() = %v, want %v", gotReqs, tt.want)
			}
		})
	}
}
