package runtime

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/ActiveState/cli/pkg/runtime/internal/envdef"
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

type Runtime struct {
	path          string
	hash          string // The stored hash for the given runtime path, if one exists (otherwise empty)
	envCollection *envdef.Collection
	env           Environment
}

type Environment struct {
	Variables              map[string]string
	VariablesWithExecutors map[string]string
	ExecutorsPath          string
}

func New(path string) (*Runtime, error) {
	env, err := envdef.NewCollection(filepath.Join(path, configDir, environmentFile))
	if err != nil && !errors.Is(err, envdef.ErrFileNotFound) { // File not found is not an error if this is a new checkout
		return nil, errs.Wrap(err, "Failed to create environment collection")
	}

	r := &Runtime{
		path:          path,
		envCollection: env,
	}

	if err := r.loadHash(); err != nil {
		return nil, errs.Wrap(err, "Failed to load hash")
	}

	if err := r.hydrateEnvironment(); err != nil {
		return nil, errs.Wrap(err, "Failed to hydrate environment")
	}

	return r, nil
}

func (r *Runtime) Hash() string {
	return r.hash
}

func (r *Runtime) HasCache() bool {
	return r.hash != ""
}

func (r *Runtime) Update(bp *buildplan.BuildPlan, hash string, setOpts ...SetOpt) error {
	if r.hash == hash {
		logging.Debug("Runtime is already up to date")
		return nil
	}

	opts := &Opts{}
	for _, setOpt := range setOpts {
		setOpt(opts)
	}

	if opts.BuildlogFilePath == "" {
		opts.BuildlogFilePath = filepath.Join(r.path, configDir, buildLogFile)
	}

	setup, err := newSetup(r.path, bp, r.envCollection, opts)
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

// hydrateEnvironment will populate the environment information so that when Env() is called it's just passing already
// calculated data
func (r *Runtime) hydrateEnvironment() error {
	vars, err := r.envCollection.Environment()
	if err != nil {
		return errs.Wrap(err, "Failed to get environment variables")
	}

	executorsPath := ExecutorsPath(r.path)

	execVars := vars
	execVars["PATH"] = executorsPath
	if _, ok := vars["PATH"]; ok {
		execVars["PATH"] += string(os.PathListSeparator) + vars["PATH"]
	}

	r.env = Environment{
		Variables:              vars,
		VariablesWithExecutors: execVars,
		ExecutorsPath:          executorsPath,
	}

	return nil
}

func (r *Runtime) Env() Environment {
	return r.env
}

func (r *Runtime) Path() string {
	return r.path
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

func IsRuntimeDir(dir string) bool {
	return fileutils.TargetExists(filepath.Join(dir, configDir, hashFile))
}

func ExecutorsPath(baseDir string) string {
	return filepath.Join(baseDir, configDir, executorDir)
}
