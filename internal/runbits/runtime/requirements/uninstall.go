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
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/internal/runbits/runtime"
	"github.com/ActiveState/cli/internal/runbits/runtime/trigger"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
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

	pg := output.StartSpinner(r.Output, locale.T("progress_solve"), constants.TerminalAnimationInterval)
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

	// Stage the commit and solve the runtime.
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
	stagedCommit, err := bp.StageCommit(params)
	if err != nil {
		return locale.WrapError(err, "err_package_save_and_build", "Error occurred while trying to create a commit")
	}

	pg.Stop(locale.T("progress_success"))
	pg = nil

	// Update the runtime.
	if !r.Config.GetBool(constants.AsyncRuntimeConfig) {
		r.Output.Notice("")

		// For deprecated commands like `platforms remove`, change the trigger.
		trig := trigger.TriggerPackage
		if len(requirements) == 1 && requirements[0].Namespace != nil {
			switch requirements[0].Namespace.Type() {
			case model.NamespacePlatform:
				trig = trigger.TriggerPlatform
			}
		}

		// refresh or install runtime
		_, err = runtime_runbit.Update(r.prime, trig,
			runtime_runbit.WithCommit(stagedCommit),
			runtime_runbit.WithoutBuildscriptValidation(),
		)
		if err != nil {
			if !IsBuildError(err) {
				// If the error is not a build error we want to retain the changes
				if err2 := r.updateCommitID(stagedCommit.CommitID); err2 != nil {
					return errs.Pack(err, locale.WrapError(err2, "err_package_update_commit_id"))
				}
			}
			return errs.Wrap(err, "Failed to refresh runtime")
		}
	}

	// Record the new commit ID and update the local build script.
	if err := r.updateCommitID(stagedCommit.CommitID); err != nil {
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
		if depReq, ok := req.(buildscript.DependencyRequirement); ok && depReq.Name == requirement.Name {
			if lang := model.LanguageFromNamespace(depReq.Namespace); lang != "" {
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
		if platformID == platID.String() {
			requirement.Name = platID.String()
			return ptr.To(model.NewNamespacePlatform()), nil
		}
	}

	return nil, nil
}
