/*
Package main demonstrates the progress bar display of the artifacts
download and its installation without downloading any actual artifacts.

This script expects the existence of an actual tarball called "test.tar.gz"
in the current directory. My suggestion is to run

```sh
git archive --format=tar  a92df9400e4f6 | gzip -c > test.tar.gz
```

to generate a reasonably sized tar ball.

In the development we considered three different modes on how to compute the installation
progress bar.

- "simulated" creates a spinner, that reports the number of bytes unpacked
- "exact" counts the number of bytes that will be written to disk prior to decompressing the archive.
- "heuristic" updates the progress bar based on the number of bytes read, and sets the bar to complete with the last byte written.

The winner:

The "heuristic" approach does not introduce extra latency as the "exact" one, it
accurately displays a fairly linearly progressing bar.
*/
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/unarchiver"
)

type devZero struct {
}

func (dz *devZero) Read(b []byte) (int, error) {
	return len(b), nil
}

const tgzTestPath string = "test.tar.gz"

func tarGzDownloadBarHeuristic(p *progress.Progress) (err error) {

	tgz := unarchiver.NewTarGz()

	dir, err := ioutil.TempDir("", "unpack")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	tgz.UnarchiveWithProgress(tgzTestPath, dir, p)

	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Printf("%s failed with error: %s\n", os.Args[0], err)
		os.Exit(1)
	}
}

func progressRun() (elapsed time.Duration, err error) {

	if !fileutils.FileExists(tgzTestPath) {
		return 0, fmt.Errorf("Expected a tarball called 'test.tar.gz' in directory")
	}

	p := progress.New( /*mpb.WithOutput(ioutil.Discard) */ )
	defer p.Close()

	totalBar1 := p.GetTotalBar("downloading", 2)
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

	totalBar2 := p.GetTotalBar("installing", 2)

	err = tarGzDownloadBarHeuristic(p)

	totalBar2.Increment()
	totalBar2.Increment()
	return elapsed, nil
}

func run() error {
	elapsed, err := progressRun()
	fmt.Printf("extra time spent unpacking: %s\n", elapsed)
	return err
}
