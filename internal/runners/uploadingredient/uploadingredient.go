package uploadingredient

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/prompt"
	p2 "github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	model2 "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/blang/semver"
	"github.com/machinebox/graphql"
	"github.com/skratchdot/open-golang/open"
)

type Params struct {
	NameVersion captain.NameVersion
	Filepath    string
	Edit        bool
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

	namespace := model2.NewSharedNamespace(r.project.Owner())

	name := params.NameVersion.Name()
	if name == "" {
		name = filepath.Base(params.Filepath)
	}

	ts := time.Now()
	var ingredient *model2.IngredientAndVersion
	ingredients, err := model2.SearchIngredientsStrict(namespace, name, true, false, &ts)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_search", "Could not search for ingredient")
	}
	if len(ingredients) > 0 {
		ingredient = ingredients[0]
	}

	version := params.NameVersion.Version()
	if version == "" {
		if ingredient != nil && len(ingredient.Versions) > 0 {
			v, err := semver.ParseTolerant(ingredient.Versions[0].Version)
			if err != nil {
				logging.Debug("Could not parse version: %v", err)
			} else {
				version = fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch+1)
			}
		}
		// Still empty? Set default
		if version == "" {
			version = "0.0.1"
		}
	}

	path := fmt.Sprintf("%s/%s", namespace.String(), name)

	checksum, err := fileutils.Sha256Hash(params.Filepath)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_checksum", "Could not calculate checksum for file")
	}

	desc := "not provided"
	if ingredient != nil && ingredient.Ingredient.Description != nil {
		desc = *ingredient.Ingredient.Description
	}

	p, err := request.Publish(name, desc, path, version, params.Filepath, checksum)
	if err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not create publish request")
	}

	if params.Edit {
		if err := r.Edit(p); err != nil {
			return err
		}
	}

	cont, err := r.prompt.Confirm(
		"",
		locale.Tl("uploadingredient_confirm", `Upload following ingredient?
Name: {{.V0}}
Version: {{.V1}}
Namespace: {{.V2}}
Path: {{.V3}}
Checksum: {{.V4}}

`, name, version, namespace.String(), path, checksum),
		p2.BoolP(true),
	)
	if err != nil {
		return errs.Wrap(err, "Confirmation failed")
	}
	if !cont {
		r.out.Print(locale.Tl("uploadingredient_cancel", "Upload cancelled"))
		return nil
	}

	result := model.PublishResult{}

	// Currently runs with: Content-Disposition: form-data; name="query"
	// but it should be Content-Disposition: form-data; name="operations"
	if err := r.client.Run(p, &result); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not publish ingredient")
	}

	if result.Error != "" {
		return locale.NewError("err_uploadingredient_publish", "Could not publish ingredient: {{.V0}}", result.Message)
	}

	r.out.Print(locale.Tl("uploadingredient_success", `Successfully uploaded as:
Ingredient ID: {{.V0}}
Ingredient Version ID: {{.V1}}
Revision: {{.V2}}
`, result.Publish.IngredientID, result.Publish.IngredientVersionID, strconv.Itoa(result.Publish.Revision)))

	return nil
}

func (r *Runner) Edit(p *request.PublishRequest) error {
	// Prepare file for editing
	b, err := p.MarshalYaml()
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
	if _, err := r.prompt.Input("", locale.Tl("uploadingredient_edit_confirm", "Press enter when done editing"), p2.StrP("")); err != nil {
		return errs.Wrap(err, "Confirmation failed")
	}

	eb, err := fileutils.ReadFile(fn)
	if err != nil {
		return errs.Wrap(err, "Could not read file")
	}

	v := string(eb)
	_ = v

	// Write changes to request
	if err := p.UnmarshalYaml(eb); err != nil {
		return locale.WrapError(err, "err_uploadingredient_publish", "Could not unmarshal publish request")
	}

	return nil
}
