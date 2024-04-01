package hail

import (
	"context"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/ActiveState/cli/internal/errs"
)

// Received represents the data related to a message sent via watched file.
type Received struct {
	Open  time.Time
	Time  time.Time
	Data  []byte
	Error error
}

func newReceived(openedAt time.Time, data []byte, err error) *Received {
	return &Received{
		Open:  openedAt,
		Time:  time.Now(),
		Data:  data,
		Error: err,
	}
}

// Send sends a hail by saving data to the file located by the file name
// provided.
func Send(file string, data []byte) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0660)
	if err != nil {
		return errs.Wrap(err, "OpenFile %s failed", file)
	}
	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return errs.Wrap(err, "Write %s failed", file)
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
		return nil, errs.Wrap(err, "OpenFile %s failed", file)
	}
	f.Close()

	rcvs, err := monitor(ctx.Done(), openedAt, file)
	if err != nil {
		return nil, errs.Wrap(err, "monitor %s failed", file)
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
				r.Error = errs.New("events channel is closed")
				rcvs <- r
				return
			}

			data, err := w.data()
			if err != nil {
				r.Error = errs.Wrap(err, "watcher.data failed")
				rcvs <- r
				break
			}

			r.Data = data
			rcvs <- r

		case err, ok := <-w.Errors:
			r := newReceived(t, nil, nil)
			if !ok {
				r.Error = errs.New("errors channel is closed")
				rcvs <- r
				return
			}
			if err != nil {
				r.Error = errs.Wrap(err, "watched failed")
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
	return os.ReadFile(w.file)
}
