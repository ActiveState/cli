package setup

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	monomodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup/buildlog"
)

// readReadyChannel is helper function that returns how many artifactIDs have been orchestrated
// This is used TestOrchestrateSetup
func readReadyChannel(called <-chan model.ArtifactID) int {
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
		Callback    func(chan<- model.ArtifactID, model.ArtifactDownload) error
		ExpectError bool
	}{
		{
			"without errors",
			func(called chan<- model.ArtifactID, a model.ArtifactDownload) error {
				called <- a.ArtifactID
				return nil
			},
			false,
		},
		{
			"with timeouts",
			func(called chan<- model.ArtifactID, a model.ArtifactDownload) error {
				// wait a second to ensure that waiting for tasks to finish works
				time.Sleep(time.Millisecond * 100)
				called <- a.ArtifactID
				return nil
			},
			false,
		},
		{
			"with one error",
			func(called chan<- model.ArtifactID, a model.ArtifactDownload) error {
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
			func(called chan<- model.ArtifactID, a model.ArtifactDownload) error {
				return errors.New("dummy error")
			},
			true,
		},
	}

	numArtifacts := 5

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ch := make(chan model.ArtifactDownload)
			go func() {
				defer close(ch)
				for i := 0; i < numArtifacts; i++ {
					artID := model.ArtifactID(fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i))
					ad := model.ArtifactDownload{ArtifactID: artID, DownloadURI: fmt.Sprintf("uri:/artifact%d", i)}
					ch <- ad
				}
			}()
			called := make(chan model.ArtifactID, numArtifacts)
			defer close(called)
			err := orchestrateArtifactSetup(context.Background(), ch, func(a model.ArtifactDownload) error {
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
		ch := make(chan model.ArtifactDownload)
		close(ch)
		called := make(chan model.ArtifactID, numArtifacts)
		defer close(called)
		err := orchestrateArtifactSetup(context.Background(), ch, func(a model.ArtifactDownload) error {
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
		ch := make(chan model.ArtifactDownload)
		defer close(ch)
		called := make(chan model.ArtifactID, numArtifacts)
		defer close(called)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := orchestrateArtifactSetup(ctx, ch, func(a model.ArtifactDownload) error {
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

func TestArtifactScheduler(t *testing.T) {
	var dummyArtifacts []model.ArtifactDownload
	numArtifacts := 5
	for i := 0; i < numArtifacts; i++ {
		artID := model.ArtifactID(fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i))
		dummyArtifacts = append(dummyArtifacts, model.ArtifactDownload{ArtifactID: artID, DownloadURI: fmt.Sprintf("uri:/artifact%d", i)})
	}

	t.Run("read all artifacts", func(t *testing.T) {
		sched := newArtifactScheduler(context.Background(), dummyArtifacts)
		go func() {
			for i := 0; i < numArtifacts; i++ {
				<-sched.BuiltArtifactsChannel()
			}
		}()
		err := sched.Wait()
		require.NoError(t, err)
	})

	t.Run("cancel context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		sched := newArtifactScheduler(ctx, dummyArtifacts)
		err := sched.Wait()
		require.EqualError(t, err, context.Canceled.Error())
	})
}

func TestChangeSummaryArgs(t *testing.T) {
	// TODO: This function should compute the change summary arguments that supports
	// our message handler function to print out a summary of changes relative to the
	// installed model.
	// My suggestion is to implement the message handler function first to understand
	// the requirements for this function better.
}

type mockModel struct {
	CheckPointResponse      monomodel.Checkpoint
	RecipeResponse          *inventory_models.Recipe
	BuildStatusEnumResponse headchef.BuildStatusEnum
	BuildStatusResponse     *headchef_models.BuildStatusResponse
}

func (mm *mockModel) FetchCheckpointForCommit(commitID strfmt.UUID) (monomodel.Checkpoint, strfmt.DateTime, error) {
	return mm.CheckPointResponse, strfmt.NewDateTime(), nil
}

func (mm *mockModel) ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error) {
	return mm.RecipeResponse, nil
}

func (mm *mockModel) RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error) {
	return mm.BuildStatusEnumResponse, mm.BuildStatusResponse, nil
}

func (mm *mockModel) BuildLog(ctx context.Context, artifactMap map[model.ArtifactID]model.Artifact, msgHandler buildlog.BuildLogMessageHandler, recipeID strfmt.UUID) (*buildlog.BuildLog, error) {
	return nil, nil
}

func TestFetchBuildResult(t *testing.T) {
	// mock := &mockModel{}
	// /*s :=*/ NewWithModel(nil, nil, nil, mock)
	// s.FetchBuildResult("123", "owner", "project")
}
