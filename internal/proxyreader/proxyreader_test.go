package proxyreader

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type progressReport struct {
	reports []int
}

func (p *progressReport) ReportIncrement(i int) error {
	p.reports = append(p.reports, i)
	return nil
}

func TestProxyReader(t *testing.T) {
	b := []byte(strings.Repeat("bogus line\n", 100))
	p := &progressReport{}
	reader := NewProxyReader(p, bytes.NewBuffer(b))
	for i := 0; i < 10; i++ {
		read := make([]byte, 10)
		n, err := reader.Read(read)
		if err != nil && !errors.Is(err, io.EOF) {
			t.Errorf("Error reading: %v", err)
		}
		if n != 10 {
			t.Errorf("Expected 10 bytes, got %d", n)
		}
	}
	if len(p.reports) != 10 {
		t.Errorf("Expected 10 reports, got %d", len(p.reports))
	}
}
