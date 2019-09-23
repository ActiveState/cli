/*
Package main demonstrates the progress bar display of the artifacts
download and its installation without downloading any actual artifacts.

You can select three different modes on how to compute the installation
progress bar.

- "simulated" creates a spinner, that reports the number of bytes unpacked
- "exact" counts the number of bytes that will be written to disk prior to decompressing the archive.
- "heuristic" updates the progress bar based on the number of bytes read, and sets the bar to complete with the last byte written.

"Exact" and "heuristic" are expecting the existence of an actual tarball called "test.tar.gz"
in the current directory. My suggestion is to run `git archive --format=tar HEAD | gzip -c > test.tar.gz`
to generate it.

The winner:

The "heuristic" approach does not introduce extra latency as the "exact" one, it
accurately displays a fairly linearly progressing bar.
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/unarchiver"
)

var modeString = flag.String(
	"mode", "simulated",
	"how to simulate the installation bar. "+
		"Choices are 'simulated', 'exact' or 'heuristic',  "+
		"if not simulated, a tarball called 'test.tar.gz' needs to be in the current directory.",
)

type installProgressMode int

const (
	simulatedMode installProgressMode = iota
	exactMode
	heuristicMode
)

func getMode(m string) (installProgressMode, error) {
	switch m {
	case "simulated":
		return simulatedMode, nil
	case "exact":
		return exactMode, nil
	case "heuristic":
		return heuristicMode, nil
	}
	return simulatedMode, fmt.Errorf("unknown installation progress model %s", m)
}

type devZero struct {
}

func (dz *devZero) Read(b []byte) (int, error) {
	return len(b), nil
}

const tgzTestPath string = "test.tar.gz"

func tarGzDownloadBarExact(p *progress.Progress) (elapsed time.Duration, err error) {

	tgz := unarchiver.NewTarGz()

	start := time.Now()
	size, err := tgz.GetExtractedSize(tgzTestPath)
	elapsed = time.Since(start)
	if err != nil {
		return elapsed, err
	}

	downloadBar := p.AddDynamicByteProgressbar(size, 2048)

	dir, err := ioutil.TempDir("", "unpack")
	if err != nil {
		return elapsed, err
	}
	defer os.RemoveAll(dir)

	tgz.UnarchiveWithProgress(tgzTestPath, dir, downloadBar.IncrBy)

	downloadBar.Complete()
	return elapsed, nil
}

func tarGzDownloadBarHeuristic(p *progress.Progress) (err error) {

	tgz := unarchiver.NewTarGz()

	s, err := os.Stat(tgzTestPath)
	if err != nil {
		return err
	}
	size := s.Size()

	downloadBar := p.AddDynamicByteProgressbar(int(float64(size)*1.02), 0)
	tgz.SetInputStreamWrapper(downloadBar.ProxyReader)

	dir, err := ioutil.TempDir("", "unpack")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	tgz.UnarchiveWithProgress(tgzTestPath, dir, func(r int) {})
	downloadBar.Complete()

	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Printf("%s failed with error: %s\n", os.Args[0], err)
		os.Exit(1)
	}
}

func progressRun(mode installProgressMode) (elapsed time.Duration, err error) {

	if mode != simulatedMode {
		if !fileutils.FileExists(tgzTestPath) {
			return 0, fmt.Errorf("Expected a tarball called 'test.tar.gz' in directory")
		}
	}

	p := progress.New( /*mpb.WithOutput(ioutil.Discard) */ )
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

	switch mode {
	case simulatedMode:
		downloadBar2 := p.AddDynamicByteProgressbar(0, 2048)
		for i := 0; i < 10*1000*1024; i += 100 * 1024 {
			downloadBar2.IncrBy(100 * 1024)
			time.Sleep(50 * time.Millisecond)
		}
		downloadBar2.Complete()
	case exactMode:
		elapsed, err = tarGzDownloadBarExact(p)
	case heuristicMode:
		err = tarGzDownloadBarHeuristic(p)
	}
	totalBar2.Increment()
	totalBar2.Increment()
	return elapsed, nil
}

func run() error {
	flag.Parse()
	mode, err := getMode(*modeString)
	if err != nil {
		return err
	}

	elapsed, err := progressRun(mode)
	fmt.Printf("extra time spent unpacking: %s\n", elapsed)
	return err
}
