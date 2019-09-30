package progress

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type devZero struct {
	count int
}

func (dz *devZero) Read(b []byte) (int, error) {
	dz.count++

	if dz.count == 3 {
		return 0, io.EOF
	}
	return len(b), nil
}

func (dz *devZero) Close() error {
	return nil
}

// Test
func TestUnpackBar(t *testing.T) {

	buf := new(bytes.Buffer)
	readBuf := make([]byte, 10)
	func() {
		progress := New(WithBufferedOutput(buf))
		defer progress.Close()

		bar := progress.AddUnpackBar(30)
		dz := &devZero{}
		wrapped := *bar.ProxyReader(dz)
		_, err := wrapped.Read(readBuf[:])
		assert.NoError(t, err)
		_, err = wrapped.Read(readBuf[:])
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		_, err = wrapped.Read(readBuf[:])
		assert.EqualError(t, err, "EOF")
		time.Sleep(100 * time.Millisecond)
		bar.Complete()
	}()

	output := strings.TrimSpace(buf.String())
	fmt.Printf("output: %s\n", output)
	expectedTotal := "100 %"

	if strings.Count(output, expectedTotal) == 0 {
		t.Errorf("expected output bar output %s to be at 100 %%", output)
	}
}
