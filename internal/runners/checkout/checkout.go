package checkout

import (
	"os"
	"path/filepath"
	rt "runtime"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits/checker"
	"github.com/ActiveState/cli/internal/runbits/checkout"
	"github.com/ActiveState/cli/internal/runbits/git"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

type Params struct {
	Namespace     *project.Namespaced
	PreferredPath string
	Branch        string
	RuntimePath   string
	NoClone       bool
	Force         bool
	LibcVersion   string
}

type primeable interface {
	primer.Auther
	primer.Prompter
	primer.Outputer
	primer.Subsheller
	primer.Configurer
	primer.SvcModeler
	primer.Analyticer
}

type Checkout struct {
	auth      *authentication.Auth
	out       output.Outputer
	checkout  *checkout.Checkout
	svcModel  *model.SvcModel
	config    *config.Instance
	subshell  subshell.SubShell
	analytics analytics.Dispatcher
}

func NewCheckout(prime primeable) *Checkout {
	return &Checkout{
		prime.Auth(),
		prime.Output(),
		checkout.New(git.NewRepo(), prime),
		prime.SvcModel(),
		prime.Config(),
		prime.Subshell(),
		prime.Analytics(),
	}
}

func (u *Checkout) Run(params *Params) (rerr error) {
	logging.Debug("Checkout %v", params.Namespace)

	checker.RunUpdateNotifier(u.analytics, u.svcModel, u.out)

	logging.Debug("Checking out %s to %s", params.Namespace.String(), params.PreferredPath)
	var err error
	projectDir, err := u.checkout.Run(params.Namespace, params.Branch, params.RuntimePath, params.PreferredPath, params.NoClone)
	if err != nil {
		return errs.Wrap(err, "Checkout failed")
	}

	proj, err := project.FromPath(projectDir)
	if err != nil {
		return locale.WrapError(err, "err_project_frompath")
	}

	err = setLibcVersion(params.LibcVersion)
	if err != nil {
		return locale.WrapError(err, "Failed to set libc version")
	}

	// If an error occurs, remove the created activestate.yaml file and/or directory.
	if !params.Force {
		defer func() {
			if rerr == nil {
				return
			}
			err := os.Remove(proj.Path())
			if err != nil {
				multilog.Error("Failed to remove activestate.yaml after `state checkout` error: %v", err)
				return
			}
			if cwd, err := osutils.Getwd(); err == nil {
				if createdDir := filepath.Dir(proj.Path()); createdDir != cwd {
					err2 := os.RemoveAll(createdDir)
					if err2 != nil {
						multilog.Error("Failed to remove created directory after `state checkout` error: %v", err2)
					}
				}
			}
		}()
	}

	rti, err := runtime.NewFromProject(proj, target.TriggerCheckout, u.analytics, u.svcModel, u.out, u.auth)
	if err != nil {
		return locale.WrapError(err, "err_checkout_runtime_new", "Could not checkout this project.")
	}

	execDir := setup.ExecDir(rti.Target().Dir())
	u.out.Print(output.Prepare(
		locale.Tr("checkout_project_statement", proj.NamespaceString(), proj.Dir(), execDir),
		&struct {
			Namespace   string `json:"namespace"`
			Path        string `json:"path"`
			Executables string `json:"executables"`
		}{
			proj.NamespaceString(),
			proj.Dir(),
			execDir,
		}))

	return nil
}

func setLibcVersion(libcVersion string) error {
	if libcVersion == "" {
		return nil
	}

	if rt.GOOS != "linux" {
		return locale.NewInputError("err_libc_version_not_supported", "libc version is only supported on linux")
	}

	parts := strings.Split(libcVersion, ".")
	if len(parts) != 2 {
		return locale.NewInputError("err_libc_version_invalid", "libc version must be in the form of major.minor")
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return locale.WrapInputError(err, "err_libc_version_invalid", "libc version must be in the form of major.minor")
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return locale.WrapInputError(err, "err_libc_version_invalid", "libc version must be in the form of major.minor")
	}

	sysinfo.SetLibcInfo(&sysinfo.LibcInfo{
		Name:  sysinfo.Glibc,
		Major: major,
		Minor: minor,
	})

	return nil
}
