package runtime

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
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

func TestValidateCheckpoint(t *testing.T) {

}

func TestFetchBuildResult(t *testing.T) {

}
