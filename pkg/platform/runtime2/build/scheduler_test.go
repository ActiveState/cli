package build

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactScheduler(t *testing.T) {
	var dummyArtifacts []ArtifactDownload
	numArtifacts := 5
	for i := 0; i < numArtifacts; i++ {
		artID := ArtifactID(fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i))
		dummyArtifacts = append(dummyArtifacts, ArtifactDownload{ArtifactID: artID, DownloadURI: fmt.Sprintf("uri:/artifact%d", i)})
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
