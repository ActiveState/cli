package publish

import (
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
	"gopkg.in/yaml.v3"
)

type Params struct {
	Name           string
	Version        string
	Namespace      string
	Owner          string
	Description    string
	Authors        captain.UsersValue
	Depends        captain.PackagesValue
	DependsRuntime captain.PackagesValue
	DependsBuild   captain.PackagesValue
	DependsTest    captain.PackagesValue
	Features       captain.PackagesValue
	Filepath       string
	MetaFilepath   string
	Edit           bool
	Editor         bool
}

type Runner struct {
	auth    *authentication.Auth
	out     output.Outputer
	prompt  prompt.Prompter
	project *project.Project
	bp      *buildplanner.BuildPlanner
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
	primer.Prompter
	primer.SvcModeler
}

func New(prime primeable) *Runner {
	return &Runner{
		auth:    prime.Auth(),
		out:     prime.Output(),
		prompt:  prime.Prompt(),
		project: prime.Project(),
		bp:      buildplanner.NewBuildPlannerModel(prime.Auth(), prime.SvcModel()),
	}
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
			return locale.WrapExternalError(err, "err_uploadingredient_file_read", "Could not read file: {{.V0}}", params.MetaFilepath)
		}

		if err := yaml.Unmarshal(b, &reqVars); err != nil {
			return locale.WrapExternalError(err, "err_uploadingredient_file_read", "Failed to unmarshal meta file. Error received: {{.V0}}", err.Error())
		}
	}

	// Namespace
	if params.Namespace != "" {
		reqVars.Namespace = params.Namespace
	} else if reqVars.Namespace == "" && r.project != nil && r.project.Owner() != "" {
		reqVars.Namespace = model.NewNamespaceOrg(r.project.Owner(), "").String()
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

	latestRevisionTime, err := model.FetchLatestRevisionTimeStamp(r.auth)
	if err != nil {
		return errs.Wrap(err, "Unable to determine latest revision time")
	}

	isRevision := false
	if params.Version != "" {
		// Attempt to get the version if it already exists, it not existing is not an error though
		i, err := model.GetIngredientByNameAndVersion(reqVars.Namespace, reqVars.Name, params.Version, &latestRevisionTime, r.auth)
		if err != nil {
			var notFound *inventory_operations.GetNamespaceIngredientVersionNotFound
			if !errors.As(err, &notFound) {
				return errs.Wrap(err, "could not get ingredient version")
			}
		} else {
			ingredient = &ParentIngredient{*i.IngredientID, *i.IngredientVersionID, *i.Version, i.Dependencies}
			isRevision = true
		}
	}

	if ingredient == nil {
		// Attempt to find the existing ingredient, if we didn't already get it from the version specific call above
		ingredients, err := model.SearchIngredientsStrict(reqVars.Namespace, reqVars.Name, true, false, &latestRevisionTime, r.auth)
		var errSearch404 *model.ErrSearch404
		if err != nil && !errors.As(err, &errSearch404) { // 404 means either the ingredient or the namespace was not found, which is fine
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
		if err := prepareEditRequest(ingredient, &reqVars, isRevision, r.auth); err != nil {
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

	if reqVars.Namespace == "" {
		return locale.NewInputError("err_uploadingredient_namespace_required", "You have to supply the namespace when working outside of a project context")
	}

	b, err := reqVars.MarshalYaml(false)
	if err != nil {
		return errs.Wrap(err, "Could not marshal publish variables")
	}

	cont, err := r.prompt.Confirm(
		"",
		locale.Tl("uploadingredient_confirm", `Prepared the following ingredient:

{{.V0}}

Do you want to publish this ingredient?
`, string(b)), ptr.To(true), nil)
	if err != nil {
		return errs.Wrap(err, "Not confirmed")
	}
	if !cont {
		return locale.NewInputError("uploadingredient_cancel", "Publish cancelled")
	}

	r.out.Notice(locale.Tl("uploadingredient_uploading", "Publishing ingredient..."))

	publishResult, err := r.bp.Publish(reqVars, params.Filepath)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not create publish request")
	}

	if publishResult.Error != "" {
		return locale.NewError("err_uploadingredient_publish_api", "API responded with error: {{.V0}}", publishResult.Error)
	}

	logging.Debug("Published ingredient ID: %s", publishResult.IngredientID)
	logging.Debug("Published ingredient version ID: %s", publishResult.IngredientVersionID)
	logging.Debug("Published ingredient revision: %d", publishResult.Revision)

	ingredientID := strfmt.UUID(publishResult.IngredientID)
	publishedIngredient, err := model.FetchIngredient(&ingredientID, r.auth)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_fetch", "Unable to fetch newly published ingredient")
	}
	versionID := strfmt.UUID(publishResult.IngredientVersionID)

	latestTime, err := model.FetchLatestRevisionTimeStamp(r.auth)
	if err != nil {
		return locale.WrapError(err, "err_uploadingingredient_fetch_timestamp", "Unable to fetch latest revision timestamp")
	}

	publishedVersion, err := model.FetchIngredientVersion(&ingredientID, &versionID, true, ptr.To(strfmt.DateTime(latestTime)), r.auth)
	if err != nil {
		return locale.WrapError(err, "err_uploadingingredient_fetch_version", "Unable to fetch newly published ingredient version")
	}

	ingTime, err := time.Parse(time.RFC3339, publishedVersion.RevisionTimestamp.String())
	if err != nil {
		return errs.Wrap(err, "Ingredient timestamp invalid")
	}

	// Increment time by 1 second to work around API precision issue where same second comparisons can fall on either side
	ingTime = ingTime.Add(time.Second)

	r.out.Print(output.Prepare(
		locale.Tl(
			"uploadingredient_success", "",
			publishedIngredient.NormalizedName,
			*publishedIngredient.PrimaryNamespace,
			*publishedVersion.Version,
			strconv.Itoa(int(*publishedVersion.Revision)),
			ingTime.Format(time.RFC3339),
		),
		publishResult,
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
	if r.Description == "" && !params.Edit {
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

	// User input trumps inheritance from previous ingredient
	if len(params.Depends) != 0 || len(params.DependsRuntime) != 0 || len(params.DependsBuild) != 0 || len(params.DependsTest) != 0 {
		r.Dependencies = []request.PublishVariableDep{}
	}

	if len(params.Depends) != 0 {
		for _, dep := range params.Depends {
			r.Dependencies = append(
				r.Dependencies,
				request.PublishVariableDep{
					Dependency: request.Dependency{Name: dep.Name, Namespace: dep.Namespace, VersionRequirements: dep.Version},
				},
			)
		}
	}

	if len(params.DependsRuntime) != 0 {
		for _, dep := range params.DependsRuntime {
			r.Dependencies = append(
				r.Dependencies,
				request.PublishVariableDep{
					Dependency: request.Dependency{Name: dep.Name, Namespace: dep.Namespace, VersionRequirements: dep.Version, Type: request.DependencyTypeRuntime},
				},
			)
		}
	}

	if len(params.DependsBuild) != 0 {
		for _, dep := range params.DependsBuild {
			r.Dependencies = append(
				r.Dependencies,
				request.PublishVariableDep{
					Dependency: request.Dependency{Name: dep.Name, Namespace: dep.Namespace, VersionRequirements: dep.Version, Type: request.DependencyTypeBuild},
				},
			)
		}
	}

	if len(params.DependsTest) != 0 {
		for _, dep := range params.DependsTest {
			r.Dependencies = append(
				r.Dependencies,
				request.PublishVariableDep{
					Dependency: request.Dependency{Name: dep.Name, Namespace: dep.Namespace, VersionRequirements: dep.Version, Type: request.DependencyTypeTest},
				},
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
func prepareEditRequest(ingredient *ParentIngredient, r *request.PublishVariables, isRevision bool, auth *authentication.Auth) error {
	r.Version = ingredient.Version

	if !isRevision {
		authors, err := model.FetchAuthors(&ingredient.IngredientID, &ingredient.IngredientVersionID, auth)
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
					VersionRequirements: model.InventoryRequirementsToString(dep.Requirements, true),
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
	if err := osutils.OpenEditor(fn); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not open publish request file")
	}

	// Wait for confirmation
	if _, err := r.prompt.Input("", locale.Tl("uploadingredient_edit_confirm", "Press enter when done editing"), ptr.To(""), nil); err != nil {
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
