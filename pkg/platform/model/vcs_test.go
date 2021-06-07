package model_test

import (
	"encoding/json"
	"testing"

	gqlModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/pkg/platform/model"
)

type VCSTestSuite struct {
	suite.Suite
}

func (suite *VCSTestSuite) TestNamespaceMatch() {
	suite.True(model.NamespaceMatch("platform", model.NamespacePlatformMatch))
	suite.False(model.NamespaceMatch(" platform ", model.NamespacePlatformMatch))
	suite.False(model.NamespaceMatch("not-platform", model.NamespacePlatformMatch))

	suite.True(model.NamespaceMatch("language", model.NamespaceLanguageMatch))
	suite.False(model.NamespaceMatch(" language ", model.NamespaceLanguageMatch))
	suite.False(model.NamespaceMatch("not-language", model.NamespaceLanguageMatch))

	suite.True(model.NamespaceMatch("language/foo", model.NamespacePackageMatch))
	suite.False(model.NamespaceMatch(" language/foo", model.NamespacePackageMatch))

	suite.True(model.NamespaceMatch("bundles/foo", model.NamespaceBundlesMatch))
	suite.False(model.NamespaceMatch(" bundles/foo", model.NamespaceBundlesMatch))

	suite.True(model.NamespaceMatch("pre-platform-installer", model.NamespacePrePlatformMatch))
	suite.False(model.NamespaceMatch(" pre-platform-installer ", model.NamespacePrePlatformMatch))
}

func (suite *VCSTestSuite) TestChangesetFromRequirements() {
	tests := []struct {
		op   model.Operation
		reqs []*gqlModel.Requirement
		want model.Changeset
	}{
		{
			model.OperationAdded,
			[]*gqlModel.Requirement{
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "a-name",
						Requirement:       "a-req",
						VersionConstraint: "a-vercon",
					},
				},
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "b-name",
						Requirement:       "b-req",
						VersionConstraint: "b-vercon",
					},
				},
			},
			model.Changeset{
				{
					Operation:         string(model.OperationAdded),
					Namespace:         "a-name",
					Requirement:       "a-req",
					VersionConstraint: "a-vercon",
				},
				{
					Operation:         string(model.OperationAdded),
					Namespace:         "b-name",
					Requirement:       "b-req",
					VersionConstraint: "b-vercon",
				},
			},
		},
		{
			model.OperationRemoved,
			[]*gqlModel.Requirement{
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "x-name",
						Requirement:       "x-req",
						VersionConstraint: "x-vercon",
					},
				},
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "y-name",
						Requirement:       "y-req",
						VersionConstraint: "y-vercon",
					},
				},
				{
					Checkpoint: mono_models.Checkpoint{
						Namespace:         "z-name",
						Requirement:       "z-req",
						VersionConstraint: "z-vercon",
					},
				},
			},
			model.Changeset{
				{
					Operation:         string(model.OperationRemoved),
					Namespace:         "x-name",
					Requirement:       "x-req",
					VersionConstraint: "x-vercon",
				},
				{
					Operation:         string(model.OperationRemoved),
					Namespace:         "y-name",
					Requirement:       "y-req",
					VersionConstraint: "y-vercon",
				},
				{
					Operation:         string(model.OperationRemoved),
					Namespace:         "z-name",
					Requirement:       "z-req",
					VersionConstraint: "z-vercon",
				},
			},
		},
	}

	for _, tt := range tests {
		got := model.ChangesetFromRequirements(tt.op, tt.reqs)
		suite.Equal(tt.want, got)
	}
}

func TestVCSTestSuite(t *testing.T) {
	suite.Run(t, new(VCSTestSuite))
}

type requirement struct {
	name      string
	version   string
	operation string
}

func checkpoint(req ...requirement) []*gqlModel.Requirement {
	result := make([]*gqlModel.Requirement, 0)
	for _, r := range req {
		result = append(result, &gqlModel.Requirement{
			Checkpoint: mono_models.Checkpoint{
				Namespace:   "namespaceValue",
				Requirement: r.name,
			},
			VersionConstraints: []*mono_models.Constraint{
				{
					Comparator: "==",
					Version:    r.version,
				},
			},
		})
	}
	return result
}

func change(r requirement) *mono_models.CommitChangeEditable {
	return &mono_models.CommitChangeEditable{
		Namespace:   "namespaceValue",
		Requirement: r.name,
		Operation:   r.operation,
		VersionConstraints: []*mono_models.Constraint{
			{
				Comparator: "==",
				Version:    r.version,
			},
		},
	}
}

func changes(reqs ...requirement) []*mono_models.CommitChangeEditable {
	result := []*mono_models.CommitChangeEditable{}
	for _, r := range reqs {
		result = append(result, change(r))
	}
	return result
}

var added = mono_models.CommitChangeOperationAdded
var removed = mono_models.CommitChangeOperationRemoved
var updated = mono_models.CommitChangeOperationUpdated

func TestDiffCheckpoints(t *testing.T) {
	type args struct {
		cp1 []*gqlModel.Requirement
		cp2 []*gqlModel.Requirement
	}
	tests := []struct {
		name string
		args args
		want []*mono_models.CommitChangeEditable
	}{
		{
			"cp1 is empty",
			args{
				checkpoint(), checkpoint(requirement{"req1", "", ""}),
			},
			changes(requirement{"req1", "", added}),
		},
		{
			"cp2 is empty",
			args{
				checkpoint(requirement{"req1", "", ""}), checkpoint(),
			},
			changes(requirement{"req1", "", removed}),
		},
		{
			"cp2 has 1 addition",
			args{
				checkpoint(requirement{"req1", "", ""}),
				checkpoint(requirement{"req1", "", ""}, requirement{"req2", "", ""}),
			},
			changes(requirement{"req2", "", added}),
		},
		{
			"cp2 has 1 deletion",
			args{
				checkpoint(requirement{"req1", "", ""}, requirement{"req2", "", ""}),
				checkpoint(requirement{"req1", "", ""}),
			},
			changes(requirement{"req2", "", removed}),
		},
		{
			"cp2 updated version",
			args{
				checkpoint(requirement{"req1", "1.0", ""}),
				checkpoint(requirement{"req1", "2.0", ""}),
			},
			changes(requirement{"req1", "2.0", updated}),
		},
		{
			"cp2 added one package",
			args{
				checkpoint(requirement{"req1", "1.0", ""}),
				checkpoint(requirement{"req2", "1.0", ""}),
			},
			changes(requirement{"req2", "1.0", added}),
		},
		{
			"complex change",
			args{
				checkpoint(
					requirement{"req1", "1.0", ""},
					requirement{"req2", "1.0", ""},
					requirement{"req3", "1.0", ""},
					requirement{"req4", "1.0", ""},
				),
				checkpoint(
					requirement{"req10", "1.0", ""},
					requirement{"req20", "1.0", ""},
					requirement{"req1", "2.0", ""},
					requirement{"req2", "1.0", ""},
					requirement{"req3", "2.0", ""},
					requirement{"req30", "1.0", ""},
				),
			},
			changes(
				requirement{"req10", "1.0", added},
				requirement{"req20", "1.0", added},
				requirement{"req1", "2.0", updated},
				requirement{"req3", "2.0", updated},
				requirement{"req4", "1.0", removed},
				requirement{"req30", "1.0", added},
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.DiffCheckpoints(tt.args.cp1, tt.args.cp2); !assert.ElementsMatch(t, got, tt.want) {
				jgot, _ := json.MarshalIndent(got, "", "  ")
				jwant, _ := json.MarshalIndent(tt.want, "", "  ")
				t.Errorf("DiffCheckpoints() = %v, want %v", string(jgot), string(jwant))
			}
		})
	}
}