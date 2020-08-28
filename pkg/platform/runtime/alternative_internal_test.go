package runtime

import (
	"reflect"
	"testing"

	"github.com/go-openapi/strfmt"
)

func headchefArtifactByUUID(uuids string) *HeadChefArtifact {
	uuid := strfmt.UUID(uuids)
	return &HeadChefArtifact{
		ArtifactID:          &uuid,
		IngredientVersionID: strfmt.UUID(uuids),
		URI:                 strfmt.URI("https://" + uuids + ".tld/file.tar.gz"),
	}
}

func Test_artifactsToDownload(t *testing.T) {
	type args struct {
		artifactCacheUuids []strfmt.UUID
		artifactsRequested []*HeadChefArtifact
	}
	tests := []struct {
		name string
		args args
		want []strfmt.UUID
	}{
		{
			"No Cache",
			args{
				[]strfmt.UUID{},
				[]*HeadChefArtifact{
					headchefArtifactByUUID("00000000-0000-0000-0000-000000000000"),
					headchefArtifactByUUID("00000000-0000-0000-0000-000000000001"),
				},
			},
			[]strfmt.UUID{
				strfmt.UUID("00000000-0000-0000-0000-000000000000"),
				strfmt.UUID("00000000-0000-0000-0000-000000000001"),
			},
		},
		{
			"Addition",
			args{
				[]strfmt.UUID{
					strfmt.UUID("00000000-0000-0000-0000-000000000000"),
				},
				[]*HeadChefArtifact{
					headchefArtifactByUUID("00000000-0000-0000-0000-000000000000"),
					headchefArtifactByUUID("00000000-0000-0000-0000-000000000001"),
				},
			},
			[]strfmt.UUID{
				strfmt.UUID("00000000-0000-0000-0000-000000000001"),
			},
		},
		{
			"No Changes",
			args{
				[]strfmt.UUID{
					strfmt.UUID("00000000-0000-0000-0000-000000000000"),
					strfmt.UUID("00000000-0000-0000-0000-000000000001"),
				},
				[]*HeadChefArtifact{
					headchefArtifactByUUID("00000000-0000-0000-0000-000000000000"),
					headchefArtifactByUUID("00000000-0000-0000-0000-000000000001"),
				},
			},
			[]strfmt.UUID{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := artifactsToUuids(artifactsToDownload(tt.args.artifactCacheUuids, tt.args.artifactsRequested))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("artifactsToDownload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_artifactsToKeepAndDelete(t *testing.T) {
	type args struct {
		artifactCache        []artifactCacheMeta
		artifactRequestUuids []strfmt.UUID
	}
	tests := []struct {
		name       string
		args       args
		wantKeep   []strfmt.UUID
		wantDelete []strfmt.UUID
	}{
		{
			"Keep 1, Delete 1",
			args{
				[]artifactCacheMeta{
					artifactCacheMeta{ArtifactID: strfmt.UUID("00000000-0000-0000-0000-000000000000")},
					artifactCacheMeta{ArtifactID: strfmt.UUID("00000000-0000-0000-0000-000000000001")},
				},
				[]strfmt.UUID{
					strfmt.UUID("00000000-0000-0000-0000-000000000001"),
				},
			},
			[]strfmt.UUID{
				strfmt.UUID("00000000-0000-0000-0000-000000000001"),
			},
			[]strfmt.UUID{
				strfmt.UUID("00000000-0000-0000-0000-000000000000"),
			},
		},
		{
			"Keep all",
			args{
				[]artifactCacheMeta{
					artifactCacheMeta{ArtifactID: strfmt.UUID("00000000-0000-0000-0000-000000000000")},
					artifactCacheMeta{ArtifactID: strfmt.UUID("00000000-0000-0000-0000-000000000001")},
				},
				[]strfmt.UUID{
					strfmt.UUID("00000000-0000-0000-0000-000000000000"),
					strfmt.UUID("00000000-0000-0000-0000-000000000001"),
				},
			},
			[]strfmt.UUID{
				strfmt.UUID("00000000-0000-0000-0000-000000000000"),
				strfmt.UUID("00000000-0000-0000-0000-000000000001"),
			},
			[]strfmt.UUID{},
		},
		{
			"Delete all",
			args{
				[]artifactCacheMeta{
					artifactCacheMeta{ArtifactID: strfmt.UUID("00000000-0000-0000-0000-000000000000")},
					artifactCacheMeta{ArtifactID: strfmt.UUID("00000000-0000-0000-0000-000000000001")},
				},
				[]strfmt.UUID{
					strfmt.UUID("00000000-0000-0000-0000-000000000003"),
					strfmt.UUID("00000000-0000-0000-0000-000000000004"),
				},
			},
			[]strfmt.UUID{},
			[]strfmt.UUID{
				strfmt.UUID("00000000-0000-0000-0000-000000000000"),
				strfmt.UUID("00000000-0000-0000-0000-000000000001"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKeep, gotDelete := artifactsToKeepAndDelete(tt.args.artifactCache, tt.args.artifactRequestUuids)
			gotKeepUuids := artifactCacheToUuids(gotKeep)
			gotDeleteUuids := artifactCacheToUuids(gotDelete)
			if !reflect.DeepEqual(gotKeepUuids, tt.wantKeep) {
				t.Errorf("artifactsToKeepAndDelete() gotKeepUuids = %v, want %v", gotKeepUuids, tt.wantKeep)
			}
			if !reflect.DeepEqual(gotDeleteUuids, tt.wantDelete) {
				t.Errorf("artifactsToKeepAndDelete() gotDeleteUuids = %v, want %v", gotDeleteUuids, tt.wantDelete)
			}
		})
	}
}
