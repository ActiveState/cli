package packages

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/runtime/requirements"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Packages      captain.PackagesValue
	Timestamp     captain.TimeValue
	Revision      captain.IntValue
	NamespaceType *model.NamespaceType
}

// Install manages the installing execution context.
type Install struct {
	prime primeable
}

type errNamespaceMismatch struct {
	pkg *captain.PackageValue
}

func (e *errNamespaceMismatch) Error() string {
	return "namespace mismatch"
}

// NewInstall prepares an installation execution context for use.
func NewInstall(prime primeable) *Install {
	return &Install{prime}
}

// Run executes the install behavior.
func (a *Install) Run(params *InstallRunParams) (rerr error) {
	defer rationalizeError(a.prime.Auth(), &rerr)

	logging.Debug("ExecuteInstall")

	var reqs []*requirements.Requirement
	var namespaces []*model.Namespace

	for _, p := range params.Packages {
		req := &requirements.Requirement{
			Name:      p.Name,
			Version:   p.Version,
			Operation: types.OperationAdded,
		}

		if p.Namespace != "" {
			if params.NamespaceType != nil {
				// Specifying a namespace in a deprecated command like `languages install` or `bundles
				// install` is an input error.
				return &errNamespaceMismatch{&p}
			}
			req.Namespace = ptr.To(model.NewRawNamespace(p.Namespace))
		}

		req.Revision = params.Revision.Int

		reqs = append(reqs, req)
	}

	// For deprecated commands like `bundles install` and `languages install`, manually construct the
	// namespace to search in.
	switch params.NamespaceType {
	case &model.NamespaceBundle:
		commitID, err := localcommit.Get(a.prime.Project().Dir())
		if err != nil {
			return errs.Wrap(err, "Unable to get local commit")
		}

		if languages, err := model.FetchLanguagesForCommit(commitID, a.prime.Auth()); err == nil {
			for _, lang := range languages {
				namespaces = append(namespaces, ptr.To(model.NewNamespaceBundle(lang.Name)))
			}
		} else {
			return errs.Wrap(err, "Could not get language(s) from project")
		}

	case &model.NamespaceLanguage:
		namespaces = append(namespaces, ptr.To(model.NewNamespaceLanguage()))
	}

	ts, err := getTime(&params.Timestamp, a.prime.Auth(), a.prime.Project())
	if err != nil {
		return errs.Wrap(err, "Unable to get timestamp from params")
	}

	return requirements.NewRequirementOperation(a.prime).Install(ts, reqs, namespaces)
}
