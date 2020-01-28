package fork

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/cmdlets/auth"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/projects"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_client/version_control"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

var (
	failUpdateBranch = failures.Type("fork.fail.updatebranch")

	failEditProject = failures.Type("fork.fail.editproject")

	// FailForkProjectConflict represents a failure while creating a project
	FailForkProjectConflict = failures.Type(
		"fork.fail.forkprojectconflict", failures.FailUser,
	)
)

type errorData struct {
	Code    int32  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    string `json:"data,omitempty"`
}

type resultWrap struct {
	Result map[string]string `json:"result,omitempty"`
	Error  *errorData        `json:"error,omitempty"`
}

var prompter prompt.Prompter

func init() {
	prompter = prompt.New()
}

// Command holds the fork command definition
var Command = &commands.Command{
	Name:        "fork",
	Description: "fork_project",
	Run:         Execute,
	Arguments: []*commands.Argument{
		&commands.Argument{
			Name:        "arg_state_fork_namespace",
			Description: "arg_state_fork_namespace_description",
			Variable:    &Args.Namespace,
			Required:    true,
		},
	},
	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "org",
			Description: "flag_state_fork_org_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Organization,
		},
		&commands.Flag{
			Name:        "name",
			Description: "flag_state_fork_name_description",
			Type:        commands.TypeString,
			StringVar:   &Flags.Name,
		},
		&commands.Flag{
			Name:        "private",
			Description: "flag_state_fork_private_description",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Private,
		},
	},
}

// Flags hold the arg values passed through the command line
var Flags struct {
	Organization string
	Private      bool
	Name         string
	Output       *output.Format
}

// Args holds the values passed through the command line
var Args struct {
	Namespace string
}

// Execute the fork command
func Execute(cmd *cobra.Command, args []string) {
	fail := auth.RequireAuthentication(locale.T("auth_required_activate"))
	if fail != nil {
		failures.Handle(fail, locale.T("err_fork_auth_required"))
		return
	}

	namespace, fail := project.ParseNamespace(Args.Namespace)
	if fail != nil {
		failures.Handle(fail, locale.T("err_fork_invalid_namespace"))
		return
	}

	originalOwner := namespace.Owner
	originalName := namespace.Project

	newName := Flags.Name
	if newName == "" {
		newName = originalName
	}

	newOwner := Flags.Organization
	if newOwner == "" {
		newOwner, fail = promptForOwner()
		if fail != nil {
			failures.Handle(fail, locale.T("err_fork_get_owner"))
			return
		}
	}

	outfmt := output.FormatUnset
	if Flags.Output != nil {
		outfmt = *Flags.Output
	}
	outputJSON := (outfmt == output.FormatJSON || outfmt == output.FormatEditorV0)

	fail = createFork(originalOwner, newOwner, originalName, newName)
	if fail != nil {
		if outputJSON && fail.Type.Matches(FailForkProjectConflict) {
			payload := resultWrap{
				Error: &errorData{
					Code:    -16,
					Message: fail.Error(),
				},
			}
			data, err := json.Marshal(&payload)
			if err != nil {
				failures.Handle(err, locale.T("err_cannot_marshal_data"))
				return
			}

			print.Line(string(data))

			fail = failures.FailSilent.Wrap(fail)
		}

		failures.Handle(fail, locale.T("err_fork_create_fork"))
		return
	}

	result := map[string]string{
		"OriginalOwner": originalOwner,
		"OriginalName":  originalName,
		"NewOwner":      newOwner,
		"NewName":       newName,
	}

	if outputJSON {
		payload := resultWrap{Result: result}
		data, err := json.Marshal(&payload)
		if err != nil {
			failures.Handle(err, locale.T("err_cannot_marshal_data"))
			return
		}

		print.Line(string(data))
		return
	}

	print.Info(locale.T("state_fork_success", result))
}

func promptForOwner() (string, *failures.Failure) {
	currentUser := authentication.Get().WhoAmI()
	orgs, fail := model.FetchOrganizations()
	if fail != nil {
		return "", fail
	}
	if len(orgs) == 0 {
		return currentUser, nil
	}

	options := make([]string, len(orgs))
	for i, org := range orgs {
		options[i] = org.Name
	}
	options = append([]string{currentUser}, options...)

	return prompter.Select(locale.T("fork_select_org"), options, "")
}

func createFork(originalOwner, newOwner, originalName, newName string) *failures.Failure {
	originalProject, fail := model.FetchProjectByName(originalOwner, originalName)
	if fail != nil {
		return fail
	}

	newProject, fail := addNewProject(newOwner, newName)
	if fail != nil {
		return fail
	}

	originalBranch, fail := model.DefaultBranchForProject(originalProject)
	if fail != nil {
		return fail
	}

	newBranch, fail := model.DefaultBranchForProject(newProject)
	if fail != nil {
		return fail
	}

	fail = updateForkBranch(newBranch, originalBranch)
	if fail != nil {
		return fail
	}

	return editProjectDetails(originalOwner, newOwner, newName)
}

func addNewProject(owner, name string) (*mono_models.Project, *failures.Failure) {
	addParams := projects.NewAddProjectParams()
	addParams.SetOrganizationName(owner)
	addParams.SetProject(&mono_models.Project{Name: name})
	addOK, err := authentication.Client().Projects.AddProject(addParams, authentication.ClientAuth())
	if err != nil {
		msg := api.ErrorMessageFromPayload(err)

		if _, ok := err.(*projects.AddProjectConflict); ok {
			return nil, FailForkProjectConflict.New(msg)
		}

		return nil, api.FailUnknown.New(msg)
	}

	return addOK.Payload, nil
}

func updateForkBranch(new, original *mono_models.Branch) *failures.Failure {
	// The default tracking type for forked projects
	trackingType := "notify"

	updateParams := version_control.NewUpdateBranchParams()
	branch := &mono_models.BranchEditable{
		TrackingType: &trackingType,
		Tracks:       &original.BranchID,
	}
	updateParams.SetBranch(branch)
	updateParams.SetBranchID(new.BranchID)

	_, err := authentication.Client().VersionControl.UpdateBranch(updateParams, authentication.ClientAuth())
	if err != nil {
		return failUpdateBranch.Wrap(err)
	}
	return nil
}

func editProjectDetails(originalOwner, newOwner, name string) *failures.Failure {
	editParams := projects.NewEditProjectParams()
	updates := &mono_models.Project{
		ForkedFrom: &mono_models.ProjectForkedFrom{
			Organization: originalOwner,
			Project:      name,
		},
		Private: Flags.Private,
	}
	editParams.SetProject(updates)
	editParams.SetOrganizationName(newOwner)
	editParams.SetProjectName(name)

	_, err := authentication.Client().Projects.EditProject(editParams, authentication.ClientAuth())
	if err != nil {
		return failEditProject.Wrap(err)
	}

	return nil
}
