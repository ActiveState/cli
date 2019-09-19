package download

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vbauerster/mpb"
)

func TestDownload(t *testing.T) {
	cases := []struct {
		Name   string
		UseMpb bool
	}{
		{
			Name:   "without progressbar",
			UseMpb: false,
		},
		{
			Name:   "with progressbar",
			UseMpb: true,
		},
	}

	for _, td := range cases {
		t.Run(td.Name, func(t *testing.T) {

			var progress *mpb.Progress
			if td.UseMpb {
				progress = mpb.New(mpb.WithOutput(ioutil.Discard))
			}

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

			manager := New(entries, 5, progress)
			fail := manager.Download()
			assert.NoError(t, fail.ToError(), "Should download files")

			for i := 1; i <= 3; i++ {
				assert.FileExists(t, filepath.Join(os.TempDir(), "state-test-download", "file"+strconv.Itoa(i)), "Should have created the target file")
			}

			if progress != nil {
				assert.Equal(t, 1, progress.BarCount())
				progress.Wait()
			}
		})

	}

}
