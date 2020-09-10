package download

import (
	"io/ioutil"
	"net/url"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/progress"
)

// Manager is our main download manager, it takes care of processing the downloads and communicating progress
type Manager struct {
	WorkerCount int
	failure     *failures.Failure
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
func (m *Manager) Download() *failures.Failure {
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

	return m.failure
}

// Job runs an individual download job
func (m *Manager) Job(entry *Entry) {
	if m.failure != nil {
		return
	}

	u, err := url.Parse(entry.Download)
	if err != nil {
		m.failure = failures.FailNetwork.Wrap(err, locale.Tl("err_dl_url", "Invalid URL: {{.V0}}.", entry.Download))
		logging.Debug("Failure occured: %v", m.failure)
		return
	}

	bytes, err := GetWithProgress(u, m.progress)
	fail := failures.FailNetwork.Wrap(err)

	if fail != nil {
		m.failure = fail
		logging.Debug("Failure occured: %v", fail)
		return
	}

	dirname := filepath.Dir(entry.Path)
	m.failure = fileutils.MkdirUnlessExists(dirname)
	if m.failure != nil {
		return
	}

	err = ioutil.WriteFile(entry.Path, bytes, 0666)
	if err != nil {
		m.failure = failures.FailIO.Wrap(err)
	}
}

// New creates a new Manager
func New(entries []*Entry, workerCount int, progress *progress.Progress) *Manager {
	m := &Manager{WorkerCount: workerCount, entries: entries, progress: progress}
	return m
}
