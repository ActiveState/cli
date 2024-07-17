package requirements

import (
	"fmt"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

func (r *RequirementOperation) Uninstall(requirements []*Requirement) (rerr error) {
	defer r.rationalizeError(&rerr)

	if len(requirements) == 0 {
		return errNoRequirements
	}
	if r.Project == nil {
		return rationalize.ErrNoProject
	}
	if r.Project.IsHeadless() {
		return rationalize.ErrHeadless
	}

	r.Output.Notice(locale.Tr("operating_message", r.Project.NamespaceString(), r.Project.Dir()))

	commitID, err := localcommit.Get(r.Project.Dir())
	if err != nil {
		return errs.Wrap(err, "Unable to get local commit")
	}

	// Start the process of creating a commit with the requested changes.
	bp := bpModel.NewBuildPlannerModel(r.Auth)

	pg := output.StartSpinner(r.Output, locale.T("progress_commit"), constants.TerminalAnimationInterval)
	defer func() {
		if pg != nil {
			pg.Stop(locale.T("progress_fail"))
		}
	}()

	// Prepare a build script with the requested changes.
	script, err := bp.GetBuildScript(commitID.String())
	if err != nil {
		return errs.Wrap(err, "Failed to get build script")
	}

	for _, req := range requirements {
		if req.Namespace == nil {
			// Update it for use in commit messages and user-facing output.
			if ns, err := r.getRequirementNamespace(req, script); err == nil && ns != nil {
				req.Namespace = ns
			} else if err != nil {
				multilog.Error("Unable to get buildscript requirement: %v", err)
			}
		}

		// Update the build script with the removal.
		if req.Namespace == nil || req.Namespace.Type() != model.NamespacePlatform {
			ns := ""
			if req.Namespace != nil {
				ns = req.Namespace.String()
			}
			err = script.UpdateRequirement(req.Operation, types.Requirement{
				Name:      req.Name,
				Namespace: ns,
			})
			if err != nil {
				return errs.Wrap(err, "Failed to remove requirement")
			}
		} else {
			err = script.UpdatePlatform(req.Operation, strfmt.UUID(req.Name))
			if err != nil {
				return errs.Wrap(err, "Failed to remove platform")
			}
		}
	}

	// Stage the commit.
	commitMessages := make([]string, len(requirements))
	for i, req := range requirements {
		message := packageCommitMessage(req.Operation, req.Name, req.Version)
		if req.Namespace != nil && req.Namespace.Type() == model.NamespaceLanguage {
			message = languageCommitMessage(req.Operation, req.Name, req.Version)
		}
		commitMessages[i] = message
	}
	commitMessage := commitMessages[0]
	if len(commitMessages) > 1 {
		locale.Tr("commit_message_multiple", strings.Join(commitMessages, ", "))
	}
	params := bpModel.StageCommitParams{
		Owner:        r.Project.Owner(),
		Project:      r.Project.Name(),
		ParentCommit: commitID.String(),
		Description:  commitMessage,
		Script:       script,
	}
	stagedCommitID, err := bp.StageCommit(params)
	if err != nil {
		return locale.WrapError(err, "err_package_save_and_build", "Error occurred while trying to create a commit")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Solve the runtime.
	rt, rtCommit, err := runtime.Solve(r.Auth, r.Output, r.Analytics, r.Project, &stagedCommitID, target.TriggerPackage, r.SvcModel, r.Config, runtime.OptNone)
	if err != nil {
		return errs.Wrap(err, "Could not solve runtime")
	}

	// Update the runtime.
	if !r.Config.GetBool(constants.AsyncRuntimeConfig) {
		r.Output.Notice("")
		if !rt.HasCache() {
			r.Output.Notice(output.Title(locale.T("install_runtime")))
			r.Output.Notice(locale.T("install_runtime_info"))
		} else {
			r.Output.Notice(output.Title(locale.T("update_runtime")))
			r.Output.Notice(locale.T("update_runtime_info"))
		}

		// refresh or install runtime
		err = runtime.UpdateByReference(rt, rtCommit, r.Auth, r.Project, r.Output, r.Config, runtime.OptMinimalUI)
		if err != nil {
			if !runbits.IsBuildError(err) {
				// If the error is not a build error we want to retain the changes
				if err2 := r.updateCommitID(stagedCommitID); err2 != nil {
					return errs.Pack(err, locale.WrapError(err2, "err_package_update_commit_id"))
				}
			}
			return errs.Wrap(err, "Failed to refresh runtime")
		}
	}

	// Record the new commit ID and update the local build script.
	if err := r.updateCommitID(stagedCommitID); err != nil {
		return locale.WrapError(err, "err_package_update_commit_id")
	}

	// Output overall summary.
	messages := make([]string, len(requirements))
	jsonObjects := make([]interface{}, len(requirements))
	for i, req := range requirements {
		nsType := "package"
		if req.Namespace != nil && req.Namespace.Type() == model.NamespaceLanguage {
			nsType = "language"
		}
		message := locale.Tr(fmt.Sprintf("%s_version_%s", nsType, req.Operation), req.Name, req.Version)
		if req.Version == "" {
			message = locale.Tr(fmt.Sprintf("%s_%s", nsType, req.Operation), req.Name)
		}
		messages = append(messages, message)
		jsonObjects[i] = &struct {
			Name      string `json:"name"`
			Type      string `json:"type"`
			Operation string `json:"operation"`
		}{
			req.Name,
			nsType,
			req.Operation.String(),
		}
	}
	r.Output.Print(output.Prepare(strings.Join(messages, "\n"), jsonObjects))

	r.Output.Notice("") // blank line
	r.Output.Notice(locale.T("operation_success_local"))

	return nil
}

func (r *RequirementOperation) getRequirementNamespace(requirement *Requirement, script *buildscript.BuildScript) (*model.Namespace, error) {
	reqs, err := script.Requirements()
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get build script requirements")
	}

	for _, req := range reqs {
		if req.Name == requirement.Name {
			if lang := model.LanguageFromNamespace(req.Namespace); lang != "" {
				return ptr.To(model.NewNamespacePackage(lang)), nil
			}
			return ptr.To(model.NewNamespaceLanguage()), nil
		}
	}

	platformID, _, err := model.ResolveRequirementNameAndVersion(
		requirement.Name, requirement.Version, requirement.BitWidth, model.NewNamespacePlatform(), r.Auth)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to resolve platform from requirement")
	}

	platforms, err := script.Platforms()
	if err != nil {
		return nil, errs.Wrap(err, "Unable to get build script platforms")
	}

	for _, platID := range platforms {
		if strfmt.UUID(platformID) == platID {
			return ptr.To(model.NewNamespacePlatform()), nil
		}
	}

	return nil, nil
}
