package publish

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api"
	graphModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/graphql"
	"github.com/go-openapi/strfmt"
	"github.com/skratchdot/open-golang/open"
	"gopkg.in/yaml.v3"
)

type Params struct {
	Name         string
	Version      string
	Namespace    string
	Owner        string
	Description  string
	Authors      captain.UsersValue
	Depends      captain.PackagesValue
	Features     captain.PackagesValue
	Filepath     string
	MetaFilepath string
	Edit         bool
	Editor       bool
}

type Runner struct {
	auth    *auth.Auth
	out     output.Outputer
	prompt  prompt.Prompter
	project *project.Project
	client  *gqlclient.Client
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
	primer.Prompter
}

func New(prime primeable) *Runner {
	client := gqlclient.NewWithOpts(
		api.GetServiceURL(api.ServiceBuildPlanner).String(), 0,
		graphql.WithHTTPClient(http.DefaultClient),
		graphql.UseMultipartForm(),
	)
	client.SetTokenProvider(prime.Auth())
	client.EnableDebugLog()
	return &Runner{auth: prime.Auth(), out: prime.Output(), prompt: prime.Prompt(), project: prime.Project(), client: client}
}

type ParentIngredient struct {
	IngredientID        strfmt.UUID
	IngredientVersionID strfmt.UUID
	Version             string
	Dependencies        []inventory_models.Dependency `json:"dependencies"`
}

var nameRegexp = regexp.MustCompile(`\w+([_-]\w+)*`)

func (r *Runner) Run(params *Params) error {
	if !r.auth.Authenticated() {
		return locale.NewInputError("err_auth_required")
	}

	if params.Filepath != "" {
		if !fileutils.FileExists(params.Filepath) {
			return locale.NewInputError("err_uploadingredient_file_not_found", "File not found: {{.V0}}", params.Filepath)
		}
		if !strings.HasSuffix(strings.ToLower(params.Filepath), ".zip") &&
			!strings.HasSuffix(strings.ToLower(params.Filepath), ".tar.gz") {
			return locale.NewInputError("err_uploadingredient_file_not_supported", "Expected file extension to be either .zip or .tar.gz: '{{.V0}}'", params.Filepath)
		}
	} else if !params.Edit {
		return locale.NewInputError("err_uploadingredient_file_required", "You have to supply the source archive unless editing.")
	}

	reqVars := request.PublishVariables{}

	// Pass input from meta file
	if params.MetaFilepath != "" {
		if !fileutils.TargetExists(params.MetaFilepath) {
			return locale.NewInputError("err_uploadingredient_metafile_not_found", "Meta file not found: {{.V0}}", params.MetaFilepath)
		}

		b, err := fileutils.ReadFile(params.MetaFilepath)
		if err != nil {
			return locale.WrapInputError(err, "err_uploadingredient_file_read", "Could not read file: {{.V0}}", params.MetaFilepath)
		}

		if err := yaml.Unmarshal(b, &reqVars); err != nil {
			return locale.WrapInputError(err, "err_uploadingredient_file_read", "Failed to unmarshal meta file, error received: {{.V0}}", err.Error())
		}
	}

	// Namespace
	if params.Namespace != "" {
		reqVars.Namespace = params.Namespace
	} else if reqVars.Namespace == "" && r.project != nil && r.project.Owner() != "" {
		reqVars.Namespace = model.NewOrgNamespace(r.project.Owner()).String()
	}

	// Name
	if params.Name != "" { // Validate & Set name
		reqVars.Name = params.Name
	} else if reqVars.Name == "" {
		// Attempt to extract a usable name from the filename.
		name := filepath.Base(params.Filepath)
		if ext := filepath.Ext(params.Filepath); ext != "" {
			name = name[:len(name)-len(ext)] // strip extension
		}
		name = versionRegexp.ReplaceAllString(name, "") // strip version number
		if matches := nameRegexp.FindAllString(name, 1); matches != nil {
			name = matches[0] // extract name-part
		}
		reqVars.Name = name
	}

	var ingredient *ParentIngredient

	isRevision := false
	if params.Version != "" {
		// Attempt to get the version if it already exists, it not existing is not an error though
		i, err := model.GetIngredientByNameAndVersion(params.Namespace, params.Name, params.Version)
		if err != nil {
			return locale.WrapInputError(err, "err_uploadingredient_getversion", "Could not grab ingredient by version")
		}
		ingredient = &ParentIngredient{*i.IngredientID, *i.IngredientVersionID, *i.Version, i.Dependencies}
		isRevision = true
	}

	if ingredient == nil {
		// Attempt to find the existing ingredient, if we didn't already get it from the version specific call above
		ts := time.Now()
		ingredients, err := model.SearchIngredientsStrict(reqVars.Namespace, reqVars.Name, true, false, &ts)
		if err != nil && !errs.Matches(err, &model.ErrSearch404{}) { // 404 means either the ingredient or the namespace was not found, which is fine
			return locale.WrapError(err, "err_uploadingredient_search", "Could not search for ingredient")
		}
		if len(ingredients) > 0 {
			i := ingredients[0].LatestVersion
			ingredient = &ParentIngredient{*i.IngredientID, *i.IngredientVersionID, *i.Version, i.Dependencies}
			if params.Version == "" {
				isRevision = true
			}
		}
	}

	if params.Edit {
		if ingredient == nil {
			return locale.NewInputError("err_uploadingredient_edit_not_found",
				"Could not find ingredient to edit with name: '[ACTIONABLE]{{.V0}}[/RESET]', namespace: '[ACTIONABLE]{{.V1}}[/RESET]'.",
				reqVars.Name, reqVars.Namespace)
		}
		if err := prepareEditRequest(ingredient, &reqVars, isRevision); err != nil {
			return errs.Wrap(err, "Could not prepare edit request")
		}
	} else {
		if isRevision {
			return locale.NewInputError("err_uploadingredient_exists",
				"Ingredient with namespace '[ACTIONABLE]{{.V0}}[/RESET]' and name '[ACTIONABLE]{{.V1}}[/RESET]' already exists. "+
					"To edit an existing ingredient you need to pass the '[ACTIONABLE]--edit[/RESET]' flag.",
				reqVars.Namespace, reqVars.Name)
		}
	}

	if err := prepareRequestFromParams(&reqVars, params, isRevision); err != nil {
		return errs.Wrap(err, "Could not prepare request from params")
	}

	if params.Editor {
		if !r.out.Config().Interactive {
			return locale.NewInputError("err_uploadingredient_editor_not_supported", "Opening in editor is not supported in non-interactive mode")
		}
		if err := r.OpenInEditor(&reqVars); err != nil {
			return err
		}
	}

	// Validate user input
	if params.Edit {
		// Description is not currently supported for edit
		// https://activestatef.atlassian.net/browse/DX-1886
		if reqVars.Description != "" {
			return locale.NewInputError("err_uploadingredient_edit_description_not_supported")
		}
	}

	if reqVars.Namespace == "" {
		return locale.NewInputError("err_uploadingredient_namespace_required", "You have to supply the namespace when working outside of a project context")
	}

	b, err := reqVars.MarshalYaml(false)
	if err != nil {
		return errs.Wrap(err, "Could not marshal publish variables")
	}

	cont, err := r.prompt.Confirm(
		"",
		locale.Tl("uploadingredient_confirm", `Publish following ingredient?
{{.V0}}

`, string(b)),
		ptr.To(true),
	)
	if err != nil {
		return errs.Wrap(err, "Confirmation failed")
	}
	if !cont {
		r.out.Print(locale.Tl("uploadingredient_cancel", "Publish cancelled"))
		return nil
	}

	r.out.Notice(locale.Tl("uploadingredient_uploading", "Publishing ingredient..."))

	pr, err := request.Publish(reqVars, params.Filepath)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not create publish request")
	}
	result := graphModel.PublishResult{}

	if err := r.client.Run(pr, &result); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not publish ingredient")
	}

	if result.Publish.Error != "" {
		return locale.NewError("err_uploadingredient_publish_api", "API responded with error: {{.V0}}", result.Publish.Error)
	}

	r.out.Print(output.Prepare(
		locale.Tl(
			"uploadingredient_success", "",
			result.Publish.IngredientID,
			result.Publish.IngredientVersionID,
			strconv.Itoa(result.Publish.Revision),
		),
		result.Publish,
	))

	return nil
}

var versionRegexp = regexp.MustCompile(`\d+\.\d+(\.\d+)?`)

func prepareRequestFromParams(r *request.PublishVariables, params *Params, isRevision bool) error {
	if params.Version != "" {
		r.Version = params.Version
	}
	if r.Version == "" {
		r.Version = "0.0.1"
		if matches := versionRegexp.FindAllString(params.Filepath, 1); matches != nil {
			r.Version = matches[0]
		}
	}

	if params.Description != "" {
		r.Description = params.Description
	}
	if r.Description == "" && !isRevision {
		r.Description = "Not Provided"
	}

	if len(params.Authors) != 0 {
		r.Authors = []request.PublishVariableAuthor{}
		for _, author := range params.Authors {
			r.Authors = append(r.Authors, request.PublishVariableAuthor{
				Name:  author.Name,
				Email: author.Email,
			})
		}
	}

	if len(params.Depends) != 0 {
		r.Dependencies = []request.PublishVariableDep{}
		for _, dep := range params.Depends {
			r.Dependencies = append(
				r.Dependencies,
				request.PublishVariableDep{request.Dependency{Name: dep.Name, Namespace: dep.Namespace, VersionRequirements: dep.Version}, []request.Dependency{}},
			)
		}
	}

	if len(params.Features) != 0 {
		r.Features = []request.PublishVariableFeature{}
		for _, feature := range params.Features {
			r.Features = append(
				r.Features,
				request.PublishVariableFeature{Name: feature.Name, Namespace: feature.Namespace, Version: feature.Version},
			)
		}
	}

	return nil
}

// prepareEditRequest inherits meta data from the previous ingredient revision if it exists. This should really happen
// on the API, but at the time of implementation we did this client side as the API side requires significant refactorings
// to enable this behavior.
func prepareEditRequest(ingredient *ParentIngredient, r *request.PublishVariables, isRevision bool) error {
	r.Version = ingredient.Version

	if !isRevision {
		authors, err := model.FetchAuthors(&ingredient.IngredientID, &ingredient.IngredientVersionID)
		if err != nil {
			return locale.WrapError(err, "err_uploadingredient_fetch_authors", "Could not fetch authors for ingredient")
		}
		if len(authors) > 0 {
			r.Authors = []request.PublishVariableAuthor{}
			for _, author := range authors {
				var websites []string
				for _, w := range author.Websites {
					websites = append(websites, w.String())
				}
				r.Authors = append(r.Authors, request.PublishVariableAuthor{
					Name:     ptr.From(author.Name, ""),
					Email:    author.Email.String(),
					Websites: websites,
				})
			}
		}
	}

	if len(ingredient.Dependencies) > 0 {
		r.Dependencies = []request.PublishVariableDep{}
		for _, dep := range ingredient.Dependencies {
			r.Dependencies = append(
				r.Dependencies,
				request.PublishVariableDep{request.Dependency{
					Name:                ptr.From(dep.Feature, ""),
					Namespace:           ptr.From(dep.Namespace, ""),
					VersionRequirements: model.RequirementsToString(dep.Requirements),
				}, []request.Dependency{}},
			)
		}
	}

	return nil
}

func (r *Runner) OpenInEditor(pr *request.PublishVariables) error {
	// Prepare file for editing
	b, err := pr.MarshalYaml(true)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not marshal publish request")
	}
	b = append([]byte("# Edit the following file and confirm in your terminal when done\n"), b...)
	fn, err := fileutils.WriteTempFile("*.ingredient.yaml", b)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not write publish request to file")
	}

	r.out.Notice(locale.Tr("uploadingredient_editor_opening", fn))

	// Open file
	if err := open.Start(fn); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not open publish request file")
	}

	// Wait for confirmation
	if _, err := r.prompt.Input("", locale.Tl("uploadingredient_edit_confirm", "Press enter when done editing"), ptr.To("")); err != nil {
		return errs.Wrap(err, "Confirmation failed")
	}

	eb, err := fileutils.ReadFile(fn)
	if err != nil {
		return errs.Wrap(err, "Could not read file")
	}

	// Write changes to request
	if err := pr.UnmarshalYaml(eb); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not unmarshal publish request")
	}

	return nil
}
