package progress

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vbauerster/mpb/v4"
)

type mockTask struct {
	Error error
}

func (mt *mockTask) mockFileSizeTask(cb FileSizeCallback) error {
	cb(10000)
	cb(20000)
	cb(10000)
	return mt.Error
}

// Test
func TestDynamicProgressbar(t *testing.T) {

	buf := new(bytes.Buffer)
	func() {
		progress := New(mpb.WithOutput(buf))
		defer progress.Close()

		mt := mockTask{Error: nil}
		bar := progress.AddDynamicByteProgressbar(0, 2048)

		err := mt.mockFileSizeTask(bar.IncrBy)

		assert.NoError(t, err, "expected no error")

	}()

	output := strings.TrimSpace(buf.String())
	expectedTotal := "39.1KiB"

	if len(output) >= len(expectedTotal) {
		assert.Equal(t, expectedTotal, output[0:len(expectedTotal)])
	} else {
		assert.False(t, true)
	}
}
