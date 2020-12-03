package hail

import (
	"context"
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
	Fail error
}

func newReceived(openedAt time.Time, data []byte, fail error) *Received {
	return &Received{
		Open: openedAt,
		Time: time.Now(),
		Data: data,
		Fail: fail,
	}
}

// Send sends a hail by saving data to the file located by the file name
// provided.
func Send(file string, data []byte) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0660)
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
func Open(ctx context.Context, file string) (<-chan *Received, error) {
	openedAt := time.Now()

	f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		return nil, failures.FailOS.Wrap(err)
	}
	f.Close()

	rcvs, err := monitor(ctx.Done(), openedAt, file)
	if err != nil {
		return nil, failures.FailOS.Wrap(err)
	}

	return rcvs, nil
}

func monitor(done <-chan struct{}, openedAt time.Time, file string) (<-chan *Received, error) {
	w, err := newWatcher(file)
	if err != nil {
		return nil, err
	}

	rcvs := make(chan *Received)

	go func() {
		defer w.Close()
		defer close(rcvs)
		loop(done, w, rcvs, openedAt)
	}()

	return rcvs, nil
}

func loop(done <-chan struct{}, w *watcher, rcvs chan<- *Received, t time.Time) {
	for {
		select {
		case <-done:
			return

		case event, ok := <-w.Events:
			// os.OpenFile calls may trigger both write and chmod events, or a single
			// write|chmod event. Filter chmod and break if no other flag is set.
			if event.Op^fsnotify.Chmod == 0 {
				break
			}

			r := newReceived(t, nil, nil)
			if !ok {
				r.Fail = FailWatcherRead.New(",events channel is closed")
				rcvs <- r
				return
			}

			data, err := w.data()
			if err != nil {
				r.Fail = failures.FailOS.Wrap(err)
				rcvs <- r
				break
			}

			r.Data = data
			rcvs <- r

		case err, ok := <-w.Errors:
			r := newReceived(t, nil, nil)
			if !ok {
				r.Fail = FailWatcherRead.New("errors channel is closed")
				rcvs <- r
				return
			}
			if err != nil {
				r.Fail = FailWatcherInstance.Wrap(err)
				rcvs <- r
				break
			}

			rcvs <- r
		}
	}
}

type watcher struct {
	*fsnotify.Watcher
	file string
}

func newWatcher(file string) (*watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := fw.Add(file); err != nil {
		return nil, err
	}

	w := watcher{
		Watcher: fw,
		file:    file,
	}

	return &w, nil
}

func (w *watcher) data() ([]byte, error) {
	return ioutil.ReadFile(w.file)
}
