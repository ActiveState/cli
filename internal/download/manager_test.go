package download

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/progress"
)

func TestDownload(t *testing.T) {

	p := progress.New(progress.WithOutput(nil))
	defer p.Close()

	var entries []*Entry
	for i := 1; i <= 3; i++ {
		target := filepath.Join(os.TempDir(), "state-test-download", "file"+strconv.Itoa(i))
		os.Remove(target)
		defer os.Remove(target)
		entries = append(entries, &Entry{
			Path:     target,
			Download: filepath.Join("download", "file"+strconv.Itoa(i)),
		})
	}

	manager := New(entries, 5, p)
	fail := manager.Download()
	assert.NoError(t, fail, "Should download files")

	for i := 1; i <= 3; i++ {
		assert.FileExists(t, filepath.Join(os.TempDir(), "state-test-download", "file"+strconv.Itoa(i)), "Should have created the target file")
	}
}
