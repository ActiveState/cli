package download

import (
	"io/ioutil"
	"os"

	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

// Manager is our main download manager, it takes care of processing the downloads and communicating progress
type Manager struct {
	WorkerCount int
	failure     *failures.Failure
	entries     []*Entry
}

// Entry is an item to be downloaded, and a path for where it should be downloaded, Data is optional and not used in this package
type Entry struct {
	Path     string
	Download string
	Data     interface{}
}

// Download will start the download progress and blocks until the progress completes
func (m *Manager) Download() *failures.Failure {
	jobs := make(chan *Entry, len(m.entries))
	done := make(chan bool, m.WorkerCount)

	progress := mpb.New()
	bar := progress.AddBar(int64(len(m.entries)),
		mpb.PrependDecorators(
			decor.CountersNoUnit("%d / %d", 10, 0),
		),
		mpb.AppendDecorators(
			decor.ETA(3, 0),
		))

	for w := 1; w <= m.WorkerCount; w++ {
		// we can't know ahead of time how many jobs each worker will take, so approximate it
		go func(jobs <-chan *Entry, bar *mpb.Bar) {
			for entry := range jobs {
				m.Job(entry)
				bar.Increment()
			}
			done <- true
		}(jobs, bar)
	}
	for _, entry := range m.entries {
		jobs <- entry
	}
	close(jobs)

	for w := 1; w <= m.WorkerCount; w++ {
		<-done
	}

	bar.Complete()
	progress.Wait()

	return m.failure
}

// Job runs an individual download job
func (m *Manager) Job(entry *Entry) {
	if m.failure != nil {
		return
	}

	bytes, fail := Get(entry.Download)

	if fail != nil {
		m.failure = fail
		logging.Debug("Failure occured: %v", fail)
		return
	}

	err := ioutil.WriteFile(entry.Path, bytes, os.ModePerm)
	if err != nil {
		m.failure = failures.FailIO.Wrap(err)
	}
}

// New creates a new Manager
func New(entries []*Entry, workerCount int) *Manager {
	m := &Manager{WorkerCount: workerCount, entries: entries}
	return m
}
