package secrets

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
)

type getPrimeable interface {
	primer.Outputer
	primer.Projecter
	primer.Configurer
	primer.Auther
}

// GetRunParams tracks the info required for running Get.
type GetRunParams struct {
	Name string
}

// Get manages the getting execution context.
type Get struct {
	proj *project.Project
	out  output.Outputer
	cfg  keypairs.Configurable
	auth *authentication.Auth
}

// SecretExport defines important information about a secret that should be
// displayed.
type SecretExport struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	HasValue    bool   `json:"has_value"`
	Value       string `json:"value,omitempty"`
}

// NewGet prepares a get execution context for use.
func NewGet(p getPrimeable) *Get {
	return &Get{
		out:  p.Output(),
		proj: p.Project(),
		cfg:  p.Config(),
		auth: p.Auth(),
	}
}

// Run executes the get behavior.
func (g *Get) Run(params GetRunParams) error {
	g.out.Notice(locale.Tr("operating_message", g.proj.NamespaceString(), g.proj.Dir()))
	if err := checkSecretsAccess(g.proj, g.auth); err != nil {
		return locale.WrapError(err, "secrets_err_check_access")
	}

	secret, valuePtr, err := getSecretWithValue(g.proj, params.Name, g.cfg, g.auth)
	if err != nil {
		return locale.WrapError(err, "secrets_err_values")
	}

	data := &getOutput{params.Name, secret, valuePtr}
	if err := data.Validate(g.out.Type()); err != nil {
		return locale.WrapError(err, "secrets_err_getout_invalid", "'get secret' output data invalid")
	}
	g.out.Print(data)

	return nil
}

type getOutput struct {
	reqSecret string
	secret    *project.Secret
	valuePtr  *string
}

// Validate returns a directly usable localized error.
func (o *getOutput) Validate(format output.Format) error {
	if !format.IsStructured() && o.valuePtr == nil {
		return newValuePtrIsNilError(o.reqSecret, o.secret.IsUser())
	}
	return nil
}

func (o *getOutput) MarshalOutput(format output.Format) interface{} {
	value := ""
	if o.valuePtr != nil {
		value = *o.valuePtr
	}
	return value
}

func (o *getOutput) MarshalStructured(format output.Format) interface{} {
	value := ""
	if o.valuePtr != nil {
		value = *o.valuePtr
	}
	return &SecretExport{
		o.secret.Name(),
		o.secret.Scope(),
		o.secret.Description(),
		o.valuePtr != nil,
		value,
	}

}

func newValuePtrIsNilError(reqSecret string, isUser bool) error {
	l10nKey := "secrets_err_project_not_defined"
	l10nVal := "Secret has not been defined: {{.V0}}. Either define it by running 'state secrets set {{.V0}}' or have someone in your organization sync with you by having them run 'state secrets sync'."
	if isUser {
		l10nKey = "secrets_err_user_not_defined"
		l10nVal = "Secret has not been defined: {{.V0}}. Define it by running 'state secrets set {{.V0}}'."
	}

	return locale.NewError(l10nKey, l10nVal, reqSecret)
}
