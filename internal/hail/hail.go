package hail

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/fsnotify/fsnotify"
)

// Received represents the data related to a message sent via watched file.
type Received struct {
	Open time.Time
	Time time.Time
	Data []byte
	Fail *failures.Failure
}

// Send sends a hail by saving data to the file located by the file name
// provided.
func Send(file string, data []byte) *failures.Failure {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0660)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}
	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return failures.FailOS.Wrap(err)
	}

	return nil
}

// Open opens a channel for hailing. A *Received is sent in the returned
// channel whenever the file located by the file name provided is created,
// updated, or deleted.
func Open(done <-chan struct{}, file string) (<-chan *Received, *failures.Failure) {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		return nil, failures.FailDeveloper.Wrap(err) // fix fail type
	}
	f.Close()

	t := time.Now()
	rc := make(chan *Received)

	if err := open(done, rc, t, file); err != nil {
		return nil, failures.FailDeveloper.Wrap(err) // fix fail type
	}

	return rc, nil
}

func open(done <-chan struct{}, rc chan<- *Received, t time.Time, file string) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		defer w.Close()

		for {
			select {
			case <-done:
				return

			case _, ok := <-w.Events:
				// consider checking for deletion
				r := &Received{Open: t, Time: time.Now()}
				if !ok {
					r.Fail = failures.FailDeveloper.Wrap(
						errors.New("oops - closed events"),
					) // fix fail type
					rc <- r
					return
				}

				data, err := ioutil.ReadFile(file)
				if err != nil {
					r.Fail = failures.FailDeveloper.Wrap(err) // fix fail type
					rc <- r
					return
				}

				r.Data = data
				rc <- r

			case err, ok := <-w.Errors:
				if !ok {
					err = errors.New("oh no - closed errors")
				}

				rc <- &Received{
					Open: t,
					Time: time.Now(),
					Fail: failures.FailDeveloper.Wrap(err), // fix fail type
				}
			}
		}
	}()

	return w.Add(file)
}
