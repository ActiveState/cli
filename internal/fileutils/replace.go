package fileutils

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ActiveState/cli/internal/logging"
)

type replacer struct {
	find    string
	replace string
	queue   chan string
	include includeFunc
	errors  chan error
	wg      sync.WaitGroup
	workers int
}

func (r *replacer) replaceAllInDirectory(path string) error {
	r.workers = runtime.GOMAXPROCS(0)
	r.queue = make(chan string, 100000)
	r.errors = make(chan error, 100000)
	defer close(r.queue)
	for i := 0; i < r.workers; i++ {
		go r.processQueue()
	}
	r.queueFiles(path)
	r.wg.Wait()

	if len(r.errors) > 0 {
		return <-r.errors
	}
	return nil
}

func (r *replacer) processQueue() {
	for {
		entry, ok := <-r.queue
		if !ok {
			return
		}
		err := r.replaceAll(entry, r.find, r.replace, r.include)
		if err != nil {
			r.errors <- err
		}
	}
}

func (r *replacer) queueFiles(path string) {
	tree := []string{}
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		tree = append(tree, path)
		return nil
	})
	if err != nil {
		r.errors <- err
	}

	r.wg.Add(len(tree))
	for _, path := range tree {
		r.queue <- path
	}
}

func (r *replacer) replaceAll(filename, find string, replace string, include includeFunc) error {
	defer func() {
		r.wg.Done()
	}()

	// Read the file's bytes and create find and replace byte arrays for search
	// and replace.
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if !include(filename, fileBytes) {
		return nil
	}

	logging.Debug("Replacing %s with %s in %s", find, replace, filename)

	findBytes := []byte(find)
	replaceBytes := []byte(replace)

	// Check if the file is a binary file. If so, the search and replace byte
	// arrays must be of equal length (replacement being NUL-padded as necessary).
	if IsBinary(fileBytes) {
		logging.Debug("Assuming file '%s' is a binary file", filename)
		if len(replaceBytes) > len(findBytes) {
			logging.Debug("Replacement text too long: %s, original text: %s", string(replaceBytes), string(findBytes))
			return errors.New("replacement text cannot be longer than search text in a binary file")
		} else if len(findBytes) > len(replaceBytes) {
			// Pad replacement with NUL bytes.
			logging.Debug("Padding replacement text by %d byte(s)", len(findBytes)-len(replaceBytes))
			paddedReplaceBytes := make([]byte, len(findBytes))
			copy(paddedReplaceBytes, replaceBytes)
			replaceBytes = paddedReplaceBytes
		}
	} else {
		logging.Debug("Assuming file '%s' is a text file", filename)
	}

	chunks := bytes.Split(fileBytes, findBytes)
	if len(chunks) < 2 {
		// nothing to replace
		return nil
	}

	// Open a temporary file for the replacement file and then perform the search
	// and replace.
	buffer := bytes.NewBuffer([]byte{})

	for i, chunk := range chunks {
		// Write chunk up to found bytes.
		if _, err := buffer.Write(chunk); err != nil {
			return err
		}
		if i < len(chunks)-1 {
			// Write replacement bytes.
			if _, err := buffer.Write(replaceBytes); err != nil {
				return err
			}
		}
	}

	return WriteFile(filename, buffer.Bytes()).ToError()
}
