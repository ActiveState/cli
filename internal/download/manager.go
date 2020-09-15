package download

import (
	"io/ioutil"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
)

// Manager is our main download manager, it takes care of processing the downloads and communicating progress
type Manager struct {
	WorkerCount int
	err         error
	entries     []*Entry
	progress    *progress.Progress
}

// Entry is an item to be downloaded, and a path for where it should be downloaded, Data is optional and not used in this package
type Entry struct {
	Path     string
	Download string
	Data     interface{}
}

// Download will start the download progress and blocks until the progress completes
func (m *Manager) Download() error {
	jobs := make(chan *Entry, len(m.entries))
	done := make(chan bool, m.WorkerCount)

	bar := m.progress.AddTotalBar(locale.T("downloading"), len(m.entries))

	for w := 1; w <= m.WorkerCount; w++ {
		// we can't know ahead of time how many jobs each worker will take, so approximate it
		go func(jobs <-chan *Entry, bar *progress.TotalBar) {
			for entry := range jobs {
				m.Job(entry)
				if bar != nil {
					bar.Increment()
				}
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

	return m.err
}

// Job runs an individual download job
func (m *Manager) Job(entry *Entry) {
	if m.err != nil {
		return
	}

	bytes, err := GetWithProgress(entry.Download, m.progress)
	if err != nil {
		m.err = err
		logging.Debug("error occured: %v", err)
		return
	}

	dirname := filepath.Dir(entry.Path)
	fail := fileutils.MkdirUnlessExists(dirname)
	m.err = fail.ToError()
	if m.err != nil {
		return
	}

	err = ioutil.WriteFile(entry.Path, bytes, 0666)
	if err != nil {
		m.err = err
	}
}

// New creates a new Manager
func New(entries []*Entry, workerCount int, progress *progress.Progress) *Manager {
	m := &Manager{WorkerCount: workerCount, entries: entries, progress: progress}
	return m
}
