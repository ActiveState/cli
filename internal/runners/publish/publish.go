package publish

import (
	"cmp"
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/python/wheel"
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
	Build          string
	Edit           bool
	Editor         bool
}

type Runner struct {
	auth    *authentication.Auth
	out     output.Outputer
	prompt  prompt.Prompter
	project *project.Project
	cfg     *config.Instance
	bp      *buildplanner.BuildPlanner
}

type primeable interface {
	primer.Outputer
	primer.Auther
	primer.Projecter
	primer.Prompter
	primer.Configurer
	primer.SvcModeler
}

func New(prime primeable) *Runner {
	return &Runner{
		auth:    prime.Auth(),
		out:     prime.Output(),
		prompt:  prime.Prompt(),
		project: prime.Project(),
		cfg:     prime.Config(),
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

	reqVars := request.PublishVariables{}
	var wheelMeta *wheel.Metadata

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
	metaFileHasDeps := len(reqVars.Dependencies) > 0

	// A meta file's version selects the version to publish or edit, just like --version (which takes precedence).
	// Without this, an exported meta file passed back with --edit would edit the latest version instead of its own.
	params.Version = cmp.Or(params.Version, reqVars.Version)

	if params.Build != "" {
		meta, cleanup, err := r.generateEncryptedArtifact(params, reqVars) // note: this function also mutates params
		if err != nil {
			return errs.Wrap(err, "Could not build private ingredient")
		}
		defer cleanup() // remove the temporary build directory
		wheelMeta = meta
	}

	if params.Filepath != "" {
		if !fileutils.FileExists(params.Filepath) {
			return locale.NewInputError("err_uploadingredient_file_not_found", "File not found: {{.V0}}", params.Filepath)
		}
		if !strings.HasSuffix(strings.ToLower(params.Filepath), ".zip") &&
			!strings.HasSuffix(strings.ToLower(params.Filepath), ".tar.gz") &&
			!strings.HasSuffix(strings.ToLower(params.Filepath), ".whl") {
			return locale.NewInputError("err_uploadingredient_file_not_supported", "Expected file extension: .zip, .tar.gz or .whl: '{{.V0}}'", params.Filepath)
		}
	} else if !params.Edit {
		return locale.NewInputError("err_uploadingredient_file_required", "You have to supply the source archive unless editing.")
	}

	// Namespace
	if params.Namespace != "" {
		reqVars.Namespace = params.Namespace
	} else if reqVars.Namespace == "" && r.project != nil && r.project.Owner() != "" {
		reqVars.Namespace = model.NewNamespaceOrg(r.project.Owner(), "").String()
	}

	// A --build publish is encrypted under the project org's key, so its resolved
	// namespace (flag, meta file, or default) must stay within that org.
	if params.Build != "" {
		if err := requireOrgNamespace(reqVars.Namespace, r.project.Owner()); err != nil {
			return err
		}
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

	// A new ingredient without a namespace yet has nothing to look up: a search across every namespace can time out,
	// and its first match could be any ingredient with this name. The namespace is checked once it is filled in.
	lookup := reqVars.Namespace != "" || params.Edit

	var ingredient *ParentIngredient

	latestRevisionTime, err := model.FetchLatestRevisionTimeStamp(r.auth)
	if err != nil {
		return errs.Wrap(err, "Unable to determine latest revision time")
	}

	isRevision := false
	if lookup && params.Version != "" {
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

	if lookup && ingredient == nil {
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

	// pyproject.toml dependencies apply unless the meta file lists its own;
	// --depend flags still take precedence over both.
	if wheelMeta != nil && !metaFileHasDeps {
		if err := r.applyPyprojectDependencies(&reqVars, wheelMeta.Requires); err != nil {
			return errs.Wrap(err, "Could not read pyproject.toml dependencies")
		}
	}

	if err := prepareRequestFromParams(&reqVars, params, isRevision); err != nil {
		return errs.Wrap(err, "Could not prepare request from params")
	}

	if params.Editor {
		if !r.out.Config().Interactive {
			return locale.NewInputError("err_uploadingredient_editor_not_supported", "Opening in editor is not supported in non-interactive mode")
		}
		if err := r.editRequest(&reqVars, params); err != nil {
			return err
		}
	}

	publish, err := r.confirmRequest(&reqVars, params, isRevision)
	if err != nil {
		return err
	}
	if !publish {
		return nil // saved to a meta file instead
	}

	r.out.Notice(locale.Tl("uploadingredient_uploading", "Publishing ingredient..."))

	publishResult, err := r.bp.Publish(reqVars, params.Filepath)
	if err != nil {
		return publishFailedError(err)
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

// checkEditKeepsIdentity rejects author and description changes on a revision of an existing version, which the
// platform does not allow; it would otherwise fail the publish with an unclear API error.
func checkEditKeepsIdentity(r *request.PublishVariables, edit, isRevision bool) error {
	if edit && isRevision && (len(r.Authors) > 0 || r.Description != "") {
		return locale.NewInputError("err_uploadingredient_edit_identity",
			"Authors and description cannot be changed when editing an existing version. Remove them from the meta file and flags.")
	}
	return nil
}

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
// to enable this behavior. Authors and dependencies already supplied by a meta file are kept rather than inherited.
// The version is always the edited ingredient's: an edit never changes it.
func prepareEditRequest(ingredient *ParentIngredient, r *request.PublishVariables, isRevision bool, auth *authentication.Auth) error {
	r.Version = ingredient.Version

	if !isRevision && len(r.Authors) == 0 {
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

	if len(ingredient.Dependencies) > 0 && len(r.Dependencies) == 0 {
		r.Dependencies = []request.PublishVariableDep{}
		for _, dep := range ingredient.Dependencies {
			conditions := []request.Dependency{}
			for _, c := range dep.Conditions {
				conditions = append(conditions, request.Dependency{
					Name:                ptr.From(c.Feature, ""),
					Namespace:           ptr.From(c.Namespace, ""),
					VersionRequirements: model.InventoryRequirementsToString(c.Requirements, true),
				})
			}
			r.Dependencies = append(
				r.Dependencies,
				request.PublishVariableDep{Dependency: request.Dependency{
					Name:                ptr.From(dep.Feature, ""),
					Namespace:           ptr.From(dep.Namespace, ""),
					VersionRequirements: model.InventoryRequirementsToString(dep.Requirements, true),
					Type:                request.DependencyType(dep.Type),
				}, Conditions: conditions},
			)
		}
	}

	return nil
}

// confirmRequest shows the prepared request and asks the user to publish it. Until they confirm, they can edit it or
// save it to a meta file and start over with it. Each pass completes and validates the request again, since an edit
// can undo what was done to it before. It reports whether to publish: false means the request was saved instead.
func (r *Runner) confirmRequest(reqVars *request.PublishVariables, params *Params, isRevision bool) (bool, error) {
	for {
		if params.Build != "" {
			addPrivateBuilderDependency(reqVars)
		}
		fillVersionRequirements(reqVars)
		if reqVars.Namespace == "" {
			return false, locale.NewInputError("err_uploadingredient_namespace_required", "You have to supply the namespace when working outside of a project context")
		}
		if err := checkEditKeepsIdentity(reqVars, params.Edit, isRevision); err != nil {
			return false, err
		}

		b, err := reqVars.MarshalYaml(false)
		if err != nil {
			return false, errs.Wrap(err, "Could not marshal publish variables")
		}

		cont, err := r.prompt.Confirm(
			"",
			locale.Tl("uploadingredient_confirm", `Prepared the following ingredient:

{{.V0}}

Do you want to publish this ingredient? Answer 'n' to edit it or save it to a file instead.
`, string(b)), ptr.To(true), nil)
		if err != nil {
			return false, errs.Wrap(err, "Not confirmed")
		}
		if cont {
			return true, nil
		}

		saved, err := r.reviseRequest(reqVars, params)
		if err != nil {
			return false, err
		}
		if saved {
			return false, nil
		}
	}
}

// publishFailedError wraps a failed publish request. en-us.yaml's err_uploadingredient_publish ends in
// "Error received: {{.V0}}" and overrides the fallback text, so the message must be passed in, or the user sees
// "<no value>". It is also the only place some failures are shown: an error the API reports in its publish result
// (e.g. "File to upload was not provided") is a plain errs.New, which is not printed beneath this message the way a
// user-facing BuildPlannerError is.
func publishFailedError(err error) error {
	return locale.WrapError(err, "err_uploadingredient_publish", "Could not create publish request", errs.JoinMessage(err))
}

// reviseRequest lets the user edit a request they did not confirm, or save it to a meta file to edit and publish later.
// It reports whether the request was saved, in which case nothing is published.
func (r *Runner) reviseRequest(reqVars *request.PublishVariables, params *Params) (bool, error) {
	edit := locale.Tl("uploadingredient_choice_edit", "Edit it in my editor")
	save := locale.Tl("uploadingredient_choice_save", "Save it to a meta file to edit and publish later")
	cancel := locale.Tl("uploadingredient_choice_cancel", "Cancel")
	choice, err := r.prompt.Select("", locale.Tl("uploadingredient_choice", "What would you like to do instead?"),
		[]string{edit, save, cancel}, &cancel, nil)
	if err != nil {
		return false, errs.Wrap(err, "Not selected")
	}

	switch choice {
	case edit:
		return false, r.editRequest(reqVars, params)
	case save:
		path, err := r.saveMetaFile(reqVars)
		if err != nil {
			return false, err
		}
		r.out.Notice(locale.Tl("uploadingredient_saved",
			"Saved the ingredient meta data to [ACTIONABLE]{{.V0}}[/RESET]. Edit it, then publish it with:\n\n  [ACTIONABLE]{{.V1}}[/RESET]\n\n"+
				"The meta file takes precedence over pyproject.toml, so the dependencies it lists replace the ones there.",
			path, rerunCommand(params, path)))
		return true, nil
	}
	return false, locale.NewInputError("uploadingredient_cancel", "Publish cancelled")
}

// editRequest opens the request in the user's editor. With --build or --edit, the name, namespace and version were
// already used to build the package or find the ingredient, so changing them here would publish something
// inconsistent: the edits are saved to a meta file instead, and the user is told to start over with it.
func (r *Runner) editRequest(reqVars *request.PublishVariables, params *Params) error {
	before := *reqVars
	if err := r.OpenInEditor(reqVars); err != nil {
		return err
	}
	if params.Build == "" && !params.Edit {
		return nil
	}
	if reqVars.Name == before.Name && reqVars.Namespace == before.Namespace && reqVars.Version == before.Version {
		return nil
	}

	path, err := r.saveMetaFile(reqVars)
	if err != nil {
		return err
	}
	return locale.NewInputError("err_uploadingredient_edit_identity_changed",
		"The name, namespace or version can't be changed at this point, because they were already used to build the package "+
			"or find the ingredient. Your changes were saved to [ACTIONABLE]{{.V0}}[/RESET]. Publish them with:\n\n  [ACTIONABLE]{{.V1}}[/RESET]",
		path, rerunCommand(params, path))
}

// saveMetaFile writes the request to <name>-<version>.meta.yaml in the working directory, in the format --meta reads,
// and returns its path. An existing file is only replaced if the user agrees.
func (r *Runner) saveMetaFile(reqVars *request.PublishVariables) (string, error) {
	path, err := filepath.Abs(reqVars.Name + "-" + reqVars.Version + ".meta.yaml")
	if err != nil {
		return "", errs.Wrap(err, "Could not determine meta file path")
	}
	if fileutils.TargetExists(path) {
		overwrite, err := r.prompt.Confirm("", locale.Tl("uploadingredient_meta_overwrite",
			"[ACTIONABLE]{{.V0}}[/RESET] already exists. Do you want to replace it?", path), ptr.To(false), nil)
		if err != nil {
			return "", errs.Wrap(err, "Not confirmed")
		}
		if !overwrite {
			return "", locale.NewInputError("uploadingredient_cancel", "Publish cancelled")
		}
	}

	// Plain yaml rather than MarshalYaml, whose field help comments don't belong in a file the user keeps.
	b, err := yaml.Marshal(reqVars)
	if err != nil {
		return "", errs.Wrap(err, "Could not marshal publish variables")
	}
	if err := fileutils.WriteFile(path, b); err != nil {
		return "", locale.WrapExternalError(err, "err_uploadingredient_meta_write", "Could not write meta file: {{.V0}}", path)
	}
	return path, nil
}

// rerunCommand is the publish command that picks up a saved meta file. Other flags are left out: their values are in
// the file already.
func rerunCommand(params *Params, metaPath string) string {
	args := []string{"state", "publish"}
	if params.Edit {
		args = append(args, "--edit")
	}
	if params.Build != "" {
		args = append(args, "--build", quoteArg(params.Build))
	} else if params.Filepath != "" {
		args = append(args, quoteArg(params.Filepath))
	}
	args = append(args, "--meta", quoteArg(metaPath))
	return strings.Join(args, " ")
}

func quoteArg(arg string) string {
	if strings.ContainsAny(arg, " \t") {
		return `"` + arg + `"`
	}
	return arg
}

// anyVersion is the version requirement the platform uses for a dependency or condition on any version.
const anyVersion = ">=0"

// fillVersionRequirements gives every dependency and condition without a version requirement the any-version one. The
// platform rejects them otherwise, and they come about easily: a pyproject.toml dependency without a version range, a
// --depend flag without a version, or a condition inherited from a revision that has no requirements.
func fillVersionRequirements(r *request.PublishVariables) {
	for i := range r.Dependencies {
		dep := &r.Dependencies[i]
		dep.VersionRequirements = cmp.Or(dep.VersionRequirements, anyVersion)
		for j := range dep.Conditions {
			dep.Conditions[j].VersionRequirements = cmp.Or(dep.Conditions[j].VersionRequirements, anyVersion)
		}
	}
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

	// Write changes to request. Unmarshal into a fresh value, so that fields removed in the editor are removed from the
	// request too instead of keeping their previous values.
	var edited request.PublishVariables
	if err := edited.UnmarshalYaml(eb); err != nil {
		return locale.WrapInputError(err, "err_uploadingredient_editor_yaml", "The edited file [ACTIONABLE]{{.V0}}[/RESET] is not valid: {{.V1}}", fn, err.Error())
	}
	*pr = edited

	return nil
}
