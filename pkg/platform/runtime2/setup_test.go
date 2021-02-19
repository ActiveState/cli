package runtime

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
	"github.com/ActiveState/cli/pkg/platform/runtime2/testhelper"
	"github.com/autarch/testify/require"
	"github.com/go-openapi/strfmt"
)

// readReadyChannel is helper function that returns how many artifactIDs have been orchestrated
// This is used TestOrchestrateSetup
func readReadyChannel(called <-chan build.ArtifactID) int {
	numCalled := 0
	for {
		select {
		case _ = <-called:
			numCalled++
		default:
			return numCalled
		}
	}
}

func TestOrchestrateSetup(t *testing.T) {
	tests := []struct {
		Name        string
		Callback    func(chan<- build.ArtifactID, build.ArtifactDownload) error
		ExpectError bool
	}{
		{
			"without errors",
			func(called chan<- build.ArtifactID, a build.ArtifactDownload) error {
				called <- a.ArtifactID
				return nil
			},
			false,
		},
		{
			"with timeouts",
			func(called chan<- build.ArtifactID, a build.ArtifactDownload) error {
				// wait a second to ensure that waiting for tasks to finish works
				time.Sleep(time.Millisecond * 100)
				called <- a.ArtifactID
				return nil
			},
			false,
		},
		{
			"with one error",
			func(called chan<- build.ArtifactID, a build.ArtifactDownload) error {
				if a.ArtifactID == "00000000-0000-0000-0000-000000000003" {
					return errors.New("dummy error")
				}
				called <- a.ArtifactID
				return nil
			},
			true,
		},
		{
			"with several errors",
			func(called chan<- build.ArtifactID, a build.ArtifactDownload) error {
				return errors.New("dummy error")
			},
			true,
		},
	}

	numArtifacts := 5

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ch := make(chan build.ArtifactDownload)
			go func() {
				defer close(ch)
				for i := 0; i < numArtifacts; i++ {
					artID := build.ArtifactID(fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i))
					ad := build.ArtifactDownload{ArtifactID: artID, DownloadURI: fmt.Sprintf("uri:/artifact%d", i)}
					ch <- ad
				}
			}()
			called := make(chan build.ArtifactID, numArtifacts)
			defer close(called)
			err := orchestrateArtifactSetup(context.Background(), ch, func(a build.ArtifactDownload) error {
				return tt.Callback(called, a)
			})
			if tt.ExpectError == (err == nil) {
				t.Fatalf("Unexpected error value: %v", err)
			}
			if !tt.ExpectError {
				numCalled := readReadyChannel(called)
				if numCalled != numArtifacts {
					t.Fatalf("callback called %d times, expected %d", numCalled, numArtifacts)
				}
			}
		})
	}

	t.Run("queue is closed", func(t *testing.T) {
		ch := make(chan build.ArtifactDownload)
		close(ch)
		called := make(chan build.ArtifactID, numArtifacts)
		defer close(called)
		err := orchestrateArtifactSetup(context.Background(), ch, func(a build.ArtifactDownload) error {
			called <- a.ArtifactID
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected err=%v", err)
		}
		numCalled := readReadyChannel(called)
		if numCalled != 0 {
			t.Fatalf("callback should not have been called, was called %d times", numCalled)
		}
	})

	t.Run("context is canceled", func(t *testing.T) {
		ch := make(chan build.ArtifactDownload)
		defer close(ch)
		called := make(chan build.ArtifactID, numArtifacts)
		defer close(called)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := orchestrateArtifactSetup(ctx, ch, func(a build.ArtifactDownload) error {
			called <- a.ArtifactID
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected err=%v", err)
		}
		numCalled := readReadyChannel(called)
		if numCalled != 0 {
			t.Fatalf("callback should not have been called, was called %d times", numCalled)
		}
	})

}

func TestChangeSummaryArgs(t *testing.T) {
	// TODO: This function should compute the change summary arguments that supports
	// our message handler function to print out a summary of changes relative to the
	// installed build.
	// My suggestion is to implement the message handler function first to understand
	// the requirements for this function better.
}

type mockModel struct {
	CheckPointResponse      model.Checkpoint
	RecipeResponse          *inventory_models.Recipe
	BuildStatusEnumResponse headchef.BuildStatusEnum
	BuildStatusResponse     *headchef_models.BuildStatusResponse
}

func (mm *mockModel) FetchCheckpointForCommit(commitID strfmt.UUID) (model.Checkpoint, strfmt.DateTime, error) {
	return mm.CheckPointResponse, strfmt.NewDateTime(), nil
}

func (mm *mockModel) ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error) {
	return mm.RecipeResponse, nil
}

func (mm *mockModel) RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error) {
	return mm.BuildStatusEnumResponse, mm.BuildStatusResponse, nil
}

func (mm *mockModel) BuildLog(ctx context.Context, artifactMap map[build.ArtifactID]build.Artifact, msgHandler build.BuildLogMessageHandler, recipeID strfmt.UUID) (*build.BuildLog, error) {
	return nil, nil
}

func TestValidateCheckpoint(t *testing.T) {
	t.Run("no commit", func(t *testing.T) {
		mm := &mockModel{}
		s := NewSetupWithAPI(nil, nil, mm)
		err := s.ValidateCheckpoint("")
		require.Error(t, err)
	})
	t.Run("valid commit", func(t *testing.T) {
		mm := &mockModel{CheckPointResponse: testhelper.LoadCheckpoint(t, "perl-order")}
		s := NewSetupWithAPI(nil, nil, mm)
		err := s.ValidateCheckpoint(strfmt.UUID(constants.ValidZeroUUID))
		require.NoError(t, err)
	})
	t.Run("preplatform order commit", func(t *testing.T) {
		mm := &mockModel{CheckPointResponse: testhelper.LoadCheckpoint(t, "pre-platform-order")}
		s := NewSetupWithAPI(nil, nil, mm)
		err := s.ValidateCheckpoint(strfmt.UUID(constants.ValidZeroUUID))
		require.Error(t, err)
	})
}

func TestFetchBuildResult(t *testing.T) {
	mock := &mockModel{}
	/*s :=*/ NewSetupWithAPI(nil, nil, mock)
	// s.FetchBuildResult("123", "owner", "project")
}
