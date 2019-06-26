package hail

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/fsnotify/fsnotify"
)

var (
	// FailWatcherRead indicates a failure to read from a Watcher chan
	FailWatcherRead = failures.Type("hail.fail.watcherread")

	// FailWatcherInstance indicates a failure from an active Watcher
	FailWatcherInstance = failures.Type("hail.fail.watcherinstance")
)

// Received represents the data related to a message sent via watched file.
type Received struct {
	Open time.Time
	Time time.Time
	Data []byte
	Fail *failures.Failure
}

func newReceived(openedAt time.Time, data []byte, fail *failures.Failure) *Received {
	return &Received{
		Open: openedAt,
		Time: time.Now(),
		Data: data,
		Fail: fail,
	}
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
		return nil, failures.FailOS.Wrap(err)
	}
	f.Close()

	t := time.Now()
	rc := make(chan *Received)
	if fail := open(done, rc, t, file); fail != nil {
		return nil, fail
	}

	return rc, nil
}

func open(done <-chan struct{}, rc chan<- *Received, t time.Time, file string) *failures.Failure {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	m := &monitor{done, rc}
	go m.run(w, t, file)

	if err := w.Add(file); err != nil {
		return failures.FailOS.Wrap(err)
	}
	return nil
}

type monitor struct {
	done <-chan struct{}
	rcvs chan<- *Received
}

func (m *monitor) run(w *fsnotify.Watcher, t time.Time, file string) {
	defer w.Close()

	for {
		select {
		case <-m.done:
			return

		case _, ok := <-w.Events: // event type is unimportant for now
			r := newReceived(t, nil, nil)
			if !ok {
				r.Fail = FailWatcherRead.New("events channel is closed")
				m.rcvs <- r
				return
			}

			data, err := ioutil.ReadFile(file)
			if err != nil {
				r.Fail = failures.FailOS.Wrap(err)
				m.rcvs <- r
				return
			}

			r.Data = data
			m.rcvs <- r

		case err, ok := <-w.Errors:
			var fail *failures.Failure
			if !ok {
				fail = FailWatcherRead.New("errors channel is closed")
			}
			if err != nil && fail == nil {
				fail = FailWatcherInstance.Wrap(err)
			}

			m.rcvs <- newReceived(t, nil, fail)
		}
	}
}
