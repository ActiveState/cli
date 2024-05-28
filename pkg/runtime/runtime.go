package runtime

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/go-openapi/strfmt"
)

// Constants covering the stored runtime
const (
	configDir       = ".activestate"
	stagingDir      = "staging"
	hashFile        = "hash.txt"
	buildLogFile    = "build.log"
	environmentFile = "environment.json"
	executorDir     = "exec"
)

// depotName is the directory name under which we store our artifact depot; ie. we symlink these artifacts into the
// actual runtimes.
const depotName = "depot"

// maxConcurrency is the maximum number of concurrent workers that can be running at any given time during an update
const maxConcurrency = 5

type Runtime struct {
	path string
	opts *Opts
	hash string // The stored hash for the given runtime path, if one exists (otherwise empty)
}

type Opts struct {
	PreferredLibcVersion string
	EventHandlers        []events.HandlerFunc
	BuildlogFilePath     string

	// Annotations are used strictly to pass information for the purposes of analytics
	// These should never be used for business logic. If the need to use them for business logic arises either we are
	// going down a wrong rabbit hole or we need to revisit the architecture.
	Annotations struct {
		Owner      string
		Project    string
		CommitUUID strfmt.UUID
	}
}

type SetOpt func(*Opts)

func New(path string, setOpts ...SetOpt) (*Runtime, error) {
	r := &Runtime{
		path: path,
		opts: &Opts{},
	}

	for _, setOpt := range setOpts {
		setOpt(r.opts)
	}

	if r.opts.BuildlogFilePath == "" {
		r.opts.BuildlogFilePath = filepath.Join(path, configDir, buildLogFile)
	}

	if err := r.loadHash(); err != nil {
		return nil, errs.Wrap(err, "Failed to load hash")
	}

	return r, nil
}

func (r *Runtime) Hash() string {
	return r.hash
}

func (r *Runtime) Update(bp *buildplan.BuildPlan, hash string) error {
	if r.hash == hash {
		logging.Debug("Runtime is already up to date")
		return nil
	}

	setup, err := newSetup(r.path, bp, r.opts)
	if err != nil {
		return errs.Wrap(err, "Failed to calculate artifacts to install")
	}

	if err := setup.RunAndWait(); err != nil {
		return errs.Wrap(err, "Failed to install runtime")
	}

	if err := r.saveHash(hash); err != nil {
		return errs.Wrap(err, "Failed to save hash")
	}

	return nil
}

func (r *Runtime) Env() Environment {
	return Environment{}
}

func WithEventHandlers(handlers ...events.HandlerFunc) SetOpt {
	return func(opts *Opts) { opts.EventHandlers = handlers }
}

func WithBuildlogFilePath(path string) SetOpt {
	return func(opts *Opts) { opts.BuildlogFilePath = path }
}

func WithPreferredLibcVersion(version string) SetOpt {
	return func(opts *Opts) { opts.PreferredLibcVersion = version }
}

func WithAnnotations(owner, project string, commitUUID strfmt.UUID) SetOpt {
	return func(opts *Opts) {
		opts.Annotations.Owner = owner
		opts.Annotations.Project = project
		opts.Annotations.CommitUUID = commitUUID
	}
}
