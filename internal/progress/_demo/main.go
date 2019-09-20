package main

import (
	"time"

	"github.com/ActiveState/cli/internal/progress"
)

type devZero struct {
}

func (dz *devZero) Read(b []byte) (int, error) {
	return len(b), nil
}

func main() {
	p := progress.New()
	defer p.Close()

	totalBar1 := p.GetNewTotalbar("downloading", 2, true)
	downloadBar := p.AddByteProgressBar(100)

	dz1 := downloadBar.ProxyReader(&devZero{})
	var pos int
	for pos != 100 {
		b := make([]byte, 10)
		off, _ := dz1.Read(b)
		time.Sleep(300 * time.Millisecond)
		pos += off
	}

	totalBar1.Increment()
	totalBar1.Increment()
	totalBar2 := p.GetNewTotalbar("installing", 2, false)
	downloadBar2 := p.AddDynamicByteProgressbar(10*1024, 2048)
	for i := 0; i < 10*1000*1024; i += 100 * 1024 {
		downloadBar2.IncrBy(100 * 1024)
		time.Sleep(50 * time.Millisecond)
	}
	downloadBar2.Complete()
	totalBar2.Increment()
	totalBar2.Increment()
}
