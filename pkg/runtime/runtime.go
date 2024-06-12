package runtime

import (
	"maps"
	"os"
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/ActiveState/cli/pkg/runtime/internal/envdef"
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
	envInherit    Environment
	depot         *depot
}

type Environment struct {
	Variables              map[string]string
	VariablesWithExecutors map[string]string
	ExecutorsPath          string
}

func New(path string) (*Runtime, error) {
	env := envdef.New()

	depot, err := newDepot(path)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create depot")
	}

	r := &Runtime{
		path:          path,
		envCollection: env,
		depot:         depot,
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

	setup, err := newSetup(r.path, bp, r.envCollection, r.depot, opts)
	if err != nil {
		return errs.Wrap(err, "Failed to calculate artifacts to install")
	}

	if err := setup.RunAndWait(); err != nil {
		return errs.Wrap(err, "Failed to install runtime")
	}

	if err := r.saveHash(hash); err != nil {
		return errs.Wrap(err, "Failed to save hash")
	}

	if err := r.hydrateEnvironment(); err != nil {
		return errs.Wrap(err, "Failed to hydrate environment")
	}

	return nil
}

// hydrateEnvironment will populate the environment information so that when Env() is called it's just passing already
// calculated data
func (r *Runtime) hydrateEnvironment() error {
	// Ingest environment files according to artifacts referenced in depot
	for id := range r.depot.List() {
		if _, err := r.envCollection.Load(r.depot.Path(id)); err != nil {
			return errs.Wrap(err, "Failed to load environment")
		}
	}

	vars, execVars, err := r.getEnv(false)
	if err != nil {
		return errs.Wrap(err, "Failed to get environment variables")
	}

	execPath := ExecutorsPath(r.path)

	r.env = Environment{
		Variables:              vars,
		VariablesWithExecutors: execVars,
		ExecutorsPath:          execPath,
	}

	vars, execVars, err = r.getEnv(true)
	if err != nil {
		return errs.Wrap(err, "Failed to get inherited environment variables")
	}

	r.envInherit = Environment{
		Variables:              vars,
		VariablesWithExecutors: execVars,
		ExecutorsPath:          execPath,
	}

	return nil
}

func (r *Runtime) getEnv(inherit bool) (map[string]string, map[string]string, error) {
	empty := map[string]string{}

	vars, err := r.envCollection.Environment(r.path, false)
	if err != nil {
		return empty, empty, errs.Wrap(err, "Failed to get environment variables")
	}

	executorsPath := ExecutorsPath(r.path)

	execVars := maps.Clone(vars)
	execVars["PATH"] = executorsPath
	if _, ok := vars["PATH"]; ok {
		execVars["PATH"] += string(os.PathListSeparator) + vars["PATH"]
	}

	return vars, execVars, nil
}

func (r *Runtime) Env(inherit bool) Environment {
	if inherit {
		return r.envInherit
	}
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
	return filepath.Join(baseDir, executorDir)
}
