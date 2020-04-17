package fileutils

import (
	"os"
	"path/filepath"
	"sync"
)

type replacer struct {
	path    string
	find    string
	replace string
	include includeFunc
	queue   chan string
	errors  chan error
	wg      sync.WaitGroup
}

func newReplacer(path, find, replace string, include includeFunc) *replacer {
	return &replacer{
		path:    path,
		find:    find,
		replace: replace,
		include: include,
		queue:   make(chan string, 1000),
		errors:  make(chan error, 1),
	}
}

func (r *replacer) run() error {
	r.processFiles()
	r.wg.Wait()
	return r.done()
}

func (r *replacer) processFiles() {
	err := filepath.Walk(r.path, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		r.wg.Add(1)
		go func() {
			err := ReplaceAllInDirectory(path, r.find, r.replace, r.include)
			if err != nil {
				r.errors <- err
			}
			r.wg.Done()
		}()
		return nil
	})
	if err != nil {
		r.errors <- err
	}
}

func (r *replacer) done() error {
	if len(r.errors) > 0 {
		return <-r.errors
	}
	close(r.errors)
	close(r.queue)

	return nil
}
