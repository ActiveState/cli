package secrets

import (
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/pkg/project"
)

type getPrimeable interface {
	primer.Outputer
	primer.Projecter
	primer.Configurer
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
	}
}

// Run executes the get behavior.
func (g *Get) Run(params GetRunParams) error {
	g.out.Print(locale.Tl("operating_message", "", g.proj.NamespaceString(), g.proj.Dir()))
	if err := checkSecretsAccess(g.proj); err != nil {
		return locale.WrapError(err, "secrets_err_check_access")
	}

	secret, valuePtr, err := getSecretWithValue(g.proj, params.Name, g.cfg)
	if err != nil {
		return locale.WrapError(err, "secrets_err_values")
	}

	data := newGetOutput(params.Name, secret, valuePtr)
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

func newGetOutput(reqSecret string, secret *project.Secret, valuePtr *string) *getOutput {
	return &getOutput{
		reqSecret: reqSecret,
		secret:    secret,
		valuePtr:  valuePtr,
	}
}

// Validate returns a directly usable localized error.
func (d *getOutput) Validate(format output.Format) error {
	switch format {
	case output.JSONFormatName, output.EditorV0FormatName, output.EditorFormatName:
		return nil
	default:
		if d.valuePtr == nil {
			return newValuePtrIsNilError(d.reqSecret, d.secret.IsUser())
		}
		return nil
	}
}

func (d *getOutput) MarshalOutput(format output.Format) interface{} {
	value := ""
	if d.valuePtr != nil {
		value = *d.valuePtr
	}

	switch format {
	case output.JSONFormatName, output.EditorV0FormatName, output.EditorFormatName:
		return &SecretExport{
			d.secret.Name(),
			d.secret.Scope(),
			d.secret.Description(),
			d.valuePtr != nil,
			value,
		}

	default:
		return value
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
