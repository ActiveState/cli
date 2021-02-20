package build

import (
	"context"
	"fmt"
	"testing"

	"github.com/autarch/testify/require"
)

func TestArtifactScheduler(t *testing.T) {
	dummyArtifacts := make(map[ArtifactID]Artifact)
	numArtifacts := 5
	for i := 0; i < numArtifacts; i++ {
		artID := ArtifactID(fmt.Sprintf("%d", i))
		dummyArtifacts[artID] = Artifact{ArtifactID: artID}
	}

	t.Run("read all artifacts", func(t *testing.T) {
		sched := NewArtifactScheduler(context.Background(), dummyArtifacts)
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
		sched := NewArtifactScheduler(ctx, dummyArtifacts)
		err := sched.Wait()
		require.EqualError(t, err, context.Canceled.Error())
	})
}
