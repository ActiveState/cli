package history

import (
	"reflect"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	gmodel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
)

func Test_printCommits(t *testing.T) {
	uuid := strfmt.UUID("11111111-1111-1111-1111-111111111111")
	type args struct {
		commits []*mono_models.Commit
		orgs    []gmodel.Organization
	}
	tests := []struct {
		name        string
		args        args
		wantStrings []string
		wantFailure error
	}{
		{
			"Commits",
			args{
				[]*mono_models.Commit{
					&mono_models.Commit{
						Added:    strfmt.DateTime(time.Now()),
						Author:   &uuid,
						CommitID: strfmt.UUID("11111111-1111-1111-1111-111111111111"),
						Message:  "Message",
						Changeset: []*mono_models.CommitChange{
							&mono_models.CommitChange{
								Namespace:         "",
								Operation:         "added",
								Requirement:       "foo",
								VersionConstraint: "1.0",
							},
							&mono_models.CommitChange{
								Namespace:   "",
								Operation:   "removed",
								Requirement: "bar",
							},
						},
					},
				},
				[]gmodel.Organization{
					gmodel.Organization{
						ID:          strfmt.UUID("11111111-1111-1111-1111-111111111111"),
						DisplayName: "Joe",
						URLName:     "joe",
					},
				},
			},
			[]string{
				"Joe", "added foo 1.0", "removed bar", "Message",
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catcher := outputhelper.NewCatcher()
			if got := printCommits(catcher.Outputer, tt.args.commits, tt.args.orgs); !reflect.DeepEqual(got, tt.wantFailure) {
				t.Errorf("printCommits() = %v, want %v", got, tt.wantFailure)
				for _, v := range tt.wantStrings {
					require.Contains(t, catcher.Output(), v)
				}
			}
		})
	}
}
