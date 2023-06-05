package uploadingredient

import (
	"net/http"
	"path/filepath"
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
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/api"
	graphModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/machinebox/graphql"
	"github.com/skratchdot/open-golang/open"
	"gopkg.in/yaml.v3"
)

type Params struct {
	Name         string
	Version      string
	Namespace    string
	Description  string
	Authors      captain.UsersValue
	Depends      captain.PackagesValue
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
		api.GetServiceURL(api.ServiceBuildPlan).String(), 0,
		graphql.WithHTTPClient(http.DefaultClient),
		graphql.UseMultipartForm(),
	)
	client.EnableDebugLog()
	client.SetTokenProvider(prime.Auth())
	client.EnableDebugLog()
	return &Runner{auth: prime.Auth(), out: prime.Output(), prompt: prime.Prompt(), project: prime.Project(), client: client}
}

func (r *Runner) Run(params *Params) error {
	if !r.auth.Authenticated() {
		return locale.NewInputError("err_auth_required")
	}

	if !fileutils.FileExists(params.Filepath) {
		return locale.NewInputError("err_uploadingredient_file_not_found", "File not found: {{.V0}}", params.Filepath)
	}

	reqVars := request.PublishVariables{}

	// Pass input from meta file
	if params.MetaFilepath != "" {
		if !fileutils.TargetExists(params.MetaFilepath) {
			return locale.NewInputError("err_uploadingredient_file_not_found", "File not found: {{.V0}}", params.MetaFilepath)
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
	} else if reqVars.Namespace == "" {
		reqVars.Namespace = model.NewSharedNamespace(r.project.Owner()).String()
	}

	// Validate namespace
	prefix := r.project.Owner() + "/"
	if !strings.HasPrefix(reqVars.Namespace, prefix) {
		return locale.NewInputError("err_uploadingredient_namespace_invalid_org",
			"Namespace should be prefixed '[ACTIONABLE]{{.V0}}[/RESET]', received: '[ACTIONABLE]{{.V1}}[/RESET]'.", prefix, reqVars.Namespace)
	}

	// Name
	if params.Name != "" { // Validate & Set name
		reqVars.Name = params.Name
	} else if reqVars.Name == "" {
		reqVars.Name = filepath.Base(params.Filepath)
	}

	ts := time.Now()
	ingredients, err := model.SearchIngredientsStrict(reqVars.Namespace, reqVars.Name, true, false, &ts)
	if err != nil && !errs.Matches(err, &model.ErrSearch404{}) { // 404 means either the ingredient or the namespace was not found, which is fine
		return locale.WrapError(err, "err_uploadingredient_search", "Could not search for ingredient")
	}
	var ingredient *model.IngredientAndVersion

	if params.Edit {
		if len(ingredients) == 0 {
			return locale.NewInputError("err_uploadingredient_edit_not_found",
				"Could not find ingredient to edit with name: '[ACTIONABLE]{{.V0}}[/RESET]', namespace: '[ACTIONABLE]{{.V1}}[/RESET]'.",
				reqVars.Name, reqVars.Namespace)
		}
		ingredient = ingredients[0]
		if err := prepareEditRequest(ingredient, &reqVars); err != nil {
			return errs.Wrap(err, "Could not prepare edit request")
		}
	} else {
		if len(ingredients) > 0 {
			return locale.NewInputError("err_uploadingredient_exists",
				"Ingredient with namespace '[ACTIONABLE]{{.V0}}[/RESET]' and name '[ACTIONABLE]{{.V1}}[/RESET]' already exists. "+
					"To edit an existing ingredient you need to pass the '[ACTIONABLE]--edit[/RESET]' flag.",
				reqVars.Namespace, reqVars.Name)
		}
	}

	if err := prepareRequestFromParams(&reqVars, params); err != nil {
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
		// Validate that the version input is valid
		// https://activestatef.atlassian.net/browse/DX-1885
		if reqVars.Version == "" {
			return locale.NewInputError("err_uploadingredient_edit_version_required")
		} else {
			for _, v := range ingredient.Versions {
				if reqVars.Version == v.Version {
					return locale.NewInputError("err_uploadingredient_edit_version_different")
				}
			}
		}

		// Description is not currently supported for edit
		// https://activestatef.atlassian.net/browse/DX-1886
		if reqVars.Description != p.PStr(ingredient.Ingredient.Description) {
			return locale.NewInputError("err_uploadingredient_edit_description_not_supported")
		}
	}

	b, err := reqVars.MarshalYaml(false)
	if err != nil {
		return errs.Wrap(err, "Could not marshal publish variables")
	}

	cont, err := r.prompt.Confirm(
		"",
		locale.Tl("uploadingredient_confirm", `Upload following ingredient?
{{.V0}}

`, string(b)),
		p.BoolP(true),
	)
	if err != nil {
		return errs.Wrap(err, "Confirmation failed")
	}
	if !cont {
		r.out.Print(locale.Tl("uploadingredient_cancel", "Upload cancelled"))
		return nil
	}

	r.out.Notice(locale.Tl("uploadingredient_uploading", "Uploading ingredient..."))

	pr, err := request.Publish(reqVars, params.Filepath)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not create publish request")
	}
	result := graphModel.PublishResult{}

	if err := r.client.Run(pr, &result); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not publish ingredient")
	}

	if result.Error != "" {
		return locale.NewError("err_uploadingredient_publish_api", "API responded with error: {{.V0}}", result.Message)
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

func prepareRequestFromParams(r *request.PublishVariables, params *Params) error {
	if params.Version != "" {
		r.Version = params.Version
	}
	if r.Version == "" {
		r.Version = "0.0.1"
	}

	if params.Description != "" {
		r.Description = params.Description
	}
	if r.Description == "" {
		r.Description = "not provided"
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
				request.PublishVariableDep{request.Dependency{Name: dep.Name, Namespace: dep.Namespace}, []request.Dependency{}},
			)
		}
	}

	return nil
}

func prepareEditRequest(ingredient *model.IngredientAndVersion, r *request.PublishVariables) error {
	authors, err := model.FetchAuthors(ingredient.Ingredient.IngredientID, ingredient.LatestVersion.IngredientVersionID)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_fetch_authors", "Could not fetch authors for ingredient")
	}

	r.Version = p.PStr(ingredient.LatestVersion.Version)
	r.Description = p.PStr(ingredient.Ingredient.Description)

	if len(authors) > 0 {
		r.Authors = []request.PublishVariableAuthor{}
		for _, author := range authors {
			var websites []string
			for _, w := range author.Websites {
				websites = append(websites, w.String())
			}
			r.Authors = append(r.Authors, request.PublishVariableAuthor{
				Name:     p.PStr(author.Name),
				Email:    author.Email.String(),
				Websites: websites,
			})
		}
	}

	if len(ingredient.LatestVersion.Dependencies) > 0 {
		r.Dependencies = []request.PublishVariableDep{}
		for _, dep := range ingredient.LatestVersion.Dependencies {
			r.Dependencies = append(
				r.Dependencies,
				request.PublishVariableDep{request.Dependency{
					Name:                p.PStr(dep.Feature),
					Namespace:           p.PStr(dep.Namespace),
					VersionRequirements: dep.OriginalRequirement,
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

	// Open file
	if err := open.Start(fn); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not open publish request file")
	}

	// Wait for confirmation
	if _, err := r.prompt.Input("", locale.Tl("uploadingredient_edit_confirm", "Press enter when done editing"), p.StrP("")); err != nil {
		return errs.Wrap(err, "Confirmation failed")
	}

	eb, err := fileutils.ReadFile(fn)
	if err != nil {
		return errs.Wrap(err, "Could not read file")
	}

	v := string(eb)
	_ = v

	// Write changes to request
	if err := pr.UnmarshalYaml(eb); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not unmarshal publish request")
	}

	return nil
}
