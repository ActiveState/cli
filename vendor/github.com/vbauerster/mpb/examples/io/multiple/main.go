package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

func main() {
	log.SetOutput(os.Stderr)

	url1 := "https://homebrew.bintray.com/bottles/youtube-dl-2016.12.12.sierra.bottle.tar.gz"
	url2 := "https://homebrew.bintray.com/bottles/libtiff-4.0.7.sierra.bottle.tar.gz"

	var wg sync.WaitGroup
	p := mpb.New(mpb.WithWaitGroup(&wg))

	for i, url := range [...]string{url1, url2} {
		wg.Add(1)
		name := fmt.Sprintf("url%d:", i+1)
		go download(&wg, p, name, url, i)
	}

	p.Wait()
	fmt.Println("done")
}

func download(wg *sync.WaitGroup, p *mpb.Progress, name, url string, n int) {
	defer wg.Done()
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("%s: %v", name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("non-200 status: %s", resp.Status)
		log.Printf("%s: %v", name, err)
		return
	}

	size := resp.ContentLength

	// create dest
	destName := filepath.Base(url)
	dest, err := os.Create(destName)
	if err != nil {
		err = fmt.Errorf("Can't create %s: %v", destName, err)
		log.Printf("%s: %v", name, err)
		return
	}

	// create bar with appropriate decorators
	bar := p.AddBar(size,
		mpb.PrependDecorators(
			decor.StaticName(name, 0, 0),
			decor.CountersKibiByte("%6.1f / %6.1f", 18, 0),
		),
		mpb.AppendDecorators(decor.ETA(5, decor.DwidthSync)),
	)
	// Respect the order
	p.UpdateBarPriority(bar, n)

	// create proxy reader
	reader := bar.ProxyReader(resp.Body)
	// and copy from reader
	_, err = io.Copy(dest, reader)

	if e := dest.Close(); err == nil {
		err = e
	}
	if err != nil {
		log.Printf("%s: %v", name, err)
	}
}
