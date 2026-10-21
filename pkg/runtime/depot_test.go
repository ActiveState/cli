package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/require"
)

// newTestDepot returns a depot rooted at dir holding the given artifacts, each with files under
// installdir/shared/ whose content is the artifact's ID.
func newTestDepot(t *testing.T, dir string, files int, ids ...strfmt.UUID) *depot {
	t.Helper()
	d := &depot{
		depotPath: dir,
		artifacts: map[strfmt.UUID]struct{}{},
		config:    depotConfig{Deployments: map[strfmt.UUID][]deployment{}, Cache: map[strfmt.UUID]*artifactInfo{}},
	}
	for _, id := range ids {
		d.artifacts[id] = struct{}{}
		for f := 0; f < files; f++ {
			path := filepath.Join(d.Path(id), "installdir", "shared", fmt.Sprintf("f%03d.txt", f))
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
			require.NoError(t, os.WriteFile(path, []byte(id), 0644))
		}
	}
	return d
}

// TestConcurrentCopyAndLinkKeepDepotIntact deploys two artifacts that provide the same files into one
// runtime at the same time, one by copy and one by link. Linking creates links back into the depot, so
// if the copy wrote through a link the other artifact had just created, it would overwrite that
// artifact's files in the depot, corrupting the cache for every runtime that uses it.
func TestConcurrentCopyAndLinkKeepDepotIntact(t *testing.T) {
	const files, iterations = 200, 20
	copied := strfmt.UUID("00000000-0000-0000-0000-00000000000a")
	linked := strfmt.UUID("00000000-0000-0000-0000-00000000000b")
	root := t.TempDir()
	d := newTestDepot(t, filepath.Join(root, "depot"), files, copied, linked)

	for i := 0; i < iterations; i++ {
		dest := filepath.Join(root, fmt.Sprintf("runtime%d", i))
		var wg sync.WaitGroup
		var copyErr, linkErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, copyErr = d.DeployViaCopy(copied, "installdir", dest)
		}()
		go func() {
			defer wg.Done()
			_, linkErr = d.DeployViaLink(linked, "installdir", dest)
		}()
		wg.Wait()
		require.NoError(t, copyErr)
		require.NoError(t, linkErr)

		for _, id := range []strfmt.UUID{copied, linked} {
			for f := 0; f < files; f++ {
				content, err := os.ReadFile(filepath.Join(d.Path(id), "installdir", "shared", fmt.Sprintf("f%03d.txt", f)))
				require.NoError(t, err)
				require.Equal(t, string(id), string(content), "depot file f%03d.txt of %s was overwritten (iteration %d)", f, id, i)
			}
		}
	}
}
