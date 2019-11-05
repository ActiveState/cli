package progress

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type devZero struct {
	count int
}

// Read writes zeros into byte array three times, then return EOF
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

func expectPercentage(t *testing.T, buf *bytes.Buffer, expected int) {

	time.Sleep(150 * time.Millisecond)
	outputLines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	lastFiveOffset := len(outputLines) - 5
	if lastFiveOffset < 0 {
		lastFiveOffset = 0
	}
	output := strings.Join(outputLines[lastFiveOffset:], "\n")
	// remove non-printable characters
	re := regexp.MustCompile("[[:^print:]]")
	stripped := re.ReplaceAllLiteralString(output, "")

	expectedTotal := fmt.Sprintf("%d %%", expected)

	if strings.Count(stripped, expectedTotal) == 0 {
		t.Errorf("expected output bar %s to be at %d %%", stripped, expected)
	}
}

// Test the unpack bar with two times re-scaling
func TestUnpackBar(t *testing.T) {

	buf := new(bytes.Buffer)
	readBuf := make([]byte, 10)
	func() {
		progress := New(WithOutput(buf))
		defer progress.Close()

		bar := progress.AddUnpackBar(30, 70)
		dz := &devZero{}
		wrapped := *bar.NewProxyReader(dz)
		_, err := wrapped.Read(readBuf[:])
		assert.NoError(t, err)
		_, err = wrapped.Read(readBuf[:])
		assert.NoError(t, err)
		_, err = wrapped.Read(readBuf[:])
		assert.EqualError(t, err, "EOF")
		time.Sleep(100 * time.Millisecond)
		bar.Complete()
		expectPercentage(t, buf, 70)

		bar.ReScale(2, 90)
		bar.Increment()
		expectPercentage(t, buf, 80)
		bar.Increment()
		bar.Complete()
		expectPercentage(t, buf, 90)
		bar.ReScale(2, 100)
		bar.Increment()
		expectPercentage(t, buf, 95)
		bar.Increment()
		bar.Complete()
		expectPercentage(t, buf, 100)
	}()
}
