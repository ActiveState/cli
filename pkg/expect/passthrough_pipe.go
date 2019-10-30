// Copyright 2018 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expect

import (
	"fmt"
	"io"
	"os"
	"time"
)

// PassthroughPipe pipes data from a io.Reader and allows setting a read
// deadline. If a timeout is reached the error is returned, otherwise the error
// from the provided io.Reader returned is passed through instead.
type PassthroughPipe struct {
	reader       *os.File
	bytesReadC   chan int
	readRequestC chan chan<- int
	errC         chan error
	deadline     time.Time
}

type errPassthroughTimeout struct {
	error
}

func (errPassthroughTimeout) Timeout() bool { return true }

// NewPassthroughPipe returns a new pipe for a io.Reader that passes through
// non-timeout errors.
func NewPassthroughPipe(reader io.Reader) (*PassthroughPipe, error) {
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	errC := make(chan error, 1)
	bytesWritten := make(chan int, 1)
	bytesRead := make(chan int, 1)
	go func() {
		defer close(errC)
		defer close(bytesWritten)
		var readerErr error
		for {
			buf := make([]byte, 32*1024)
			n, err := reader.Read(buf)
			if err != nil {
				// We always overwrite the error and set it to EOF.  This way we'll always find the end of the stream.
				readerErr = io.EOF
				break
			}
			// fmt.Printf("read %d bytes: first characters are: %s\n", n, string(buf[:20]))
			nw, err := pipeWriter.Write(buf[:n])
			if err != nil {
				fmt.Printf("pipeWriter reported error: %v\n", err)
				// We always overwrite the error and set it to EOF.  This way we'll always find the end of the stream.
				readerErr = err
				break
			}
			bytesWritten <- nw
		}

		// Closing the pipeWriter will unblock the pipeReader.Read.
		err = pipeWriter.Close()
		if err != nil {
			// If we are unable to close the pipe, and the pipe isn't already closed,
			// the caller will hang indefinitely.
			panic(err)
			return
		}

		// When an error is read from reader, we need it to passthrough the err to
		// callers of (*PassthroughPipe).Read.
		errC <- readerErr
	}()

	readRequestC := make(chan chan<- int)
	go func() {
		defer close(readRequestC)
		var totalWritten int
		var totalRead int
		var readyToRead chan<- int
		for {
			select {
			case r := <-readRequestC:
				if r != nil && readyToRead != nil {
					panic("Reading from passthrough pipe needs to be serialized, sorry!")
				}
				readyToRead = r
			case nw, ok := <-bytesWritten:
				totalWritten += nw
				if !ok {
					bytesWritten = nil
				}
			case nr, ok := <-bytesRead:
				totalRead += nr
				if !ok {
					bytesRead = nil
				}
			}
			if bytesWritten == nil && bytesRead == nil {
				break
			}
			if totalWritten > totalRead && readyToRead != nil {
				readyToRead <- (totalWritten - totalRead)
				readyToRead = nil
			}
		}
	}()

	return &PassthroughPipe{
		reader:       pipeReader,
		errC:         errC,
		bytesReadC:   bytesRead,
		readRequestC: readRequestC,
	}, nil
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (pp *PassthroughPipe) Read(p []byte) (n int, err error) {
	timeoutDuration := time.Until(pp.deadline)
	if pp.deadline.IsZero() {
		timeoutDuration = time.Hour * 1000
	}
	readyToRead := make(chan int, 1)
	pp.readRequestC <- readyToRead
	var nmax int
	select {
	case readerErr := <-pp.errC:
		pp.readRequestC <- nil
		return 0, readerErr
	case nmax = <-readyToRead:
	case <-time.After(timeoutDuration):
		pp.readRequestC <- nil
		return 0, &errPassthroughTimeout{fmt.Errorf("i/o timeout")}
	}
	ps := min(nmax, len(p))
	n, err = pp.reader.Read(p[:ps])
	if err != nil {
		return n, err
	}
	pp.bytesReadC <- n

	return n, nil
}

func (pp *PassthroughPipe) Close() error {
	close(pp.bytesReadC)
	return pp.reader.Close()
}

func (pp *PassthroughPipe) SetReadDeadline(t time.Time) error {
	pp.deadline = t
	return nil
}
