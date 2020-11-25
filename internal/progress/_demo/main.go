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
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
	"github.com/ActiveState/cli/internal/unarchiver"
)

type devZero struct {
}

func (dz *devZero) Read(b []byte) (int, error) {
	return len(b), nil
}

func (dz *devZero) Close() error {
	return nil
}

const tgzTestPath string = "test.tar.gz"

func tarGzDownloadBarHeuristic(p *progress.Progress) (err error) {

	tgz := unarchiver.NewTarGz()

	dir, err := ioutil.TempDir("", "unpack")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	aFile, aSize, err := tgz.PrepareUnpacking(tgzTestPath, dir)
	if err != nil {
		return err
	}

	ub := p.AddUnpackBar(aSize, 70)
	aStream := progress.NewReaderProxy(ub.Bar(), ub, aFile)
	err = tgz.Unarchive(aStream, aSize, dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unarchiving %v\n", err)
	}
	ub.Complete()

	time.Sleep(100 * time.Millisecond)

	ub.ReScale(4)
	time.Sleep(1 * time.Second)
	ub.Increment()
	time.Sleep(1 * time.Second)
	ub.Increment()
	time.Sleep(1 * time.Second)
	ub.Increment()
	time.Sleep(1 * time.Second)
	ub.Increment()

	return nil
}

func main() {
	logging.CurrentHandler().SetVerbose(false)
	logging.SetMinimalLevel(logging.DEBUG)
	logging.SetOutput(os.Stderr)
	logging.Debug("test\n")
	err := run()
	if err != nil {
		fmt.Printf("%s failed with error: %s\n", os.Args[0], err)
		os.Exit(1)
	}
}

func progressRun() (err error) {

	if !fileutils.FileExists(tgzTestPath) {
		return fmt.Errorf("Expected a tarball called 'test.tar.gz' in directory")
	}

	p := progress.New( /* progress.WithOutput(nil) */ )
	defer p.Close()

	totalBar1 := p.AddTotalBar("downloading", 1)
	downloadBar := p.AddByteProgressBar(100)

	dz1 := downloadBar.ProxyReader(&devZero{})
	var pos int
	for pos != 100 {
		b := make([]byte, 10)
		off, _ := dz1.Read(b)
		time.Sleep(200 * time.Millisecond)
		pos += off
	}

	totalBar1.Increment()

	totalBar2 := p.AddTotalBar("installing", 1)

	err = tarGzDownloadBarHeuristic(p)
	if err != nil {
		return err
	}

	totalBar2.Increment()
	return nil
}

func run() error {
	err := progressRun()
	return err
}
